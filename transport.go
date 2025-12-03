package utils

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NewTransport creates a Transport. pass proxy like "http://127.0.0.1:3128" or "" for none.
func NewTransport(ja3 string, proxy string) (*Transport, error) {
	c := &Transport{}

	// parse proxy if provided
	if proxy != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("proxy parse fail: %w", err)
		}
		c.proxyURL = u
	}

	// init default internal transports
	c.tr1 = http.Transport{
		Proxy:                 http.ProxyFromEnvironment, // will override below if proxyURL set
		TLSHandshakeTimeout:   10 * time.Second,
		DisableKeepAlives:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	// if proxy specified for HTTP, set Proxy to ProxyURL
	if c.proxyURL != nil {
		c.tr1.Proxy = http.ProxyURL(c.proxyURL)
	}

	// http2 transport default
	c.tr2 = http2.Transport{
		AllowHTTP:                 false,
		MaxDecoderHeaderTableSize: 1 << 16,
		// We won't set DialTLS here because we need to use custom uTLS handshake
	}

	if ja3 != "" {
		var err error
		if c.spec, err = c.createSpecWithStr(ja3); err != nil {
			return nil, err
		}
	}
	return c, nil
}

type Transport struct {
	tr1      http.Transport
	tr2      http2.Transport
	spec     *utls.ClientHelloSpec
	proxyURL *url.URL
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Scheme {
	case "https":
		if t.spec == nil {
			// use standard transport (respects Proxy in tr1)
			return t.tr1.RoundTrip(req)
		} else {
			return t.httpsRoundTrip(req)
		}
	case "http":
		// http requests go through standard transport (which may have Proxy set)
		return t.tr1.RoundTrip(req)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", req.URL.Scheme)
	}
}

func (t *Transport) httpsRoundTrip(req *http.Request) (*http.Response, error) {
	port := req.URL.Port()
	if port == "" {
		port = "443"
	}

	// Decide how to connect: direct TCP to target or via HTTP proxy (CONNECT)
	var conn net.Conn
	var err error
	targetHostPort := net.JoinHostPort(req.URL.Hostname(), port)

	if t.proxyURL != nil {
		// Only support http proxy for now (CONNECT)
		if t.proxyURL.Scheme != "http" && t.proxyURL.Scheme != "HTTP" {
			return nil, fmt.Errorf("unsupported proxy scheme: %s", t.proxyURL.Scheme)
		}
		// Dial proxy
		conn, err = net.Dial("tcp", t.proxyURL.Host)
		if err != nil {
			return nil, fmt.Errorf("tcp dial proxy fail: %w", err)
		}
		// Send CONNECT
		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: Keep-Alive\r\n\r\n", targetHostPort, targetHostPort)
		if _, err = conn.Write([]byte(connectReq)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("write CONNECT to proxy fail: %w", err)
		}
		// Read proxy response
		br := bufio.NewReader(conn)
		// Read status line
		statusLine, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("read proxy response fail: %w", err)
		}
		if !strings.Contains(statusLine, "200") {
			// read remaining headers for debugging then return error
			// swallow headers
			for {
				line, _ := br.ReadString('\n')
				if line == "\r\n" || line == "\n" || line == "" {
					break
				}
			}
			conn.Close()
			return nil, fmt.Errorf("proxy CONNECT failed: %s", strings.TrimSpace(statusLine))
		}
		// consume remaining headers
		for {
			line, _ := br.ReadString('\n')
			if line == "\r\n" || line == "\n" || line == "" {
				break
			}
		}
		// now conn is a tunnel to targetHostPort; proceed to TLS over conn
	} else {
		// direct dial
		conn, err = net.Dial("tcp", targetHostPort)
		if err != nil {
			return nil, fmt.Errorf("tcp net dial fail: %w", err)
		}
	}
	// ensure closed in this func
	defer conn.Close() // nolint

	tlsConn, err := t.tlsConnect(conn, req)
	if err != nil {
		return nil, fmt.Errorf("tls connect fail: %w", err)
	}
	// do not defer tlsConn.Close() here because closing underlying conn will close it; but close explicitly
	defer tlsConn.Close() // nolint

	// ALPN / negotiated protocol
	httpVersion := tlsConn.ConnectionState().NegotiatedProtocol
	switch httpVersion {
	case "h2":
		// Use http2 client conn (we use existing tlsConn)
		clientConn, err := t.tr2.NewClientConn(tlsConn)
		if err != nil {
			return nil, fmt.Errorf("create http2 client with connection fail: %w", err)
		}
		defer clientConn.Close() // nolint
		return clientConn.RoundTrip(req)
	case "http/1.1", "":
		// write raw request to tlsConn and read response
		err := req.Write(tlsConn)
		if err != nil {
			return nil, fmt.Errorf("write http1 tls connection fail: %w", err)
		}
		return http.ReadResponse(bufio.NewReader(tlsConn), req)
	default:
		return nil, fmt.Errorf("unsuported http version: %s", httpVersion)
	}
}

func (t *Transport) getTLSConfig(host string) *utls.Config {
	return &utls.Config{
		ServerName:   host,
		OmitEmptyPsk: true,
		NextProtos:   []string{"h2", "http/1.1"},
		//InsecureSkipVerify:                 true,
		//InsecureSkipTimeVerify:             true,
		//PreferSkipResumptionOnNilExtension: true,
	}
}

func (t *Transport) tlsConnect(conn net.Conn, req *http.Request) (tlsConn *utls.UConn, err error) {
	if t.spec != nil {
		// HelloCustom with preset
		tlsConn = utls.UClient(conn, t.getTLSConfig(req.URL.Hostname()), utls.HelloCustom)

		// Ensure that if PSK extension exists we move it to last — keep original intent
		lastIndex := -1
		for i, v := range t.spec.Extensions {
			if id, _ := t.getExtensionId(v); id == 41 {
				lastIndex = i
			}
		}
		ln := len(t.spec.Extensions)
		if lastIndex != -1 && ln > 0 {
			t.spec.Extensions[lastIndex], t.spec.Extensions[ln-1] = t.spec.Extensions[ln-1], t.spec.Extensions[lastIndex]
		}
		if err = tlsConn.ApplyPreset(t.spec); err != nil {
			return nil, err
		}
	} else {
		tlsConn = utls.UClient(conn, t.getTLSConfig(req.URL.Hostname()), utls.HelloRandomized)
	}
	if err = tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("tls handshake fail: %w", err)
	}
	return tlsConn, nil
}

func (t *Transport) createSpecWithStr(ja3Str string) (*utls.ClientHelloSpec, error) {
	var (
		err             error
		clientHelloSpec utls.ClientHelloSpec
	)
	tokens := strings.Split(ja3Str, ",")
	if len(tokens) != 5 {
		return nil, errors.New("ja3Str format error")
	}
	ver, err := strconv.ParseUint(tokens[0], 10, 16)
	if err != nil {
		return nil, errors.New("ja3Str tlsVersion error")
	}
	ciphers := strings.Split(tokens[1], "-")
	extensions := strings.Split(tokens[2], "-")
	curves := strings.Split(tokens[3], "-")
	pointFormats := strings.Split(tokens[4], "-")
	tlsMaxVersion, tlsMinVersion, tlsExtension, err := t.createTlsVersion(uint16(ver))
	if err != nil {
		return nil, err
	}
	clientHelloSpec.TLSVersMax = tlsMaxVersion
	clientHelloSpec.TLSVersMin = tlsMinVersion
	if clientHelloSpec.CipherSuites, err = t.createCiphers(ciphers); err != nil {
		return nil, err
	}
	curvesExtension, err := t.createCurves(curves)
	if err != nil {
		return nil, err
	}
	pointExtension, err := t.createPointFormats(pointFormats)
	if err != nil {
		return nil, err
	}
	clientHelloSpec.CompressionMethods = []byte{0}
	clientHelloSpec.GetSessionID = sha256.Sum256
	if clientHelloSpec.Extensions, err = t.createExtensions(extensions, tlsExtension, curvesExtension, pointExtension); err != nil {
		return nil, err
	}

	// Move PSK (41) to last if present — preserve original intent
	lastIndex := -1
	for i, v := range clientHelloSpec.Extensions {
		if id, _ := t.getExtensionId(v); id == 41 {
			lastIndex = i
		}
	}
	ln := len(clientHelloSpec.Extensions)
	if lastIndex != -1 && ln > 0 {
		clientHelloSpec.Extensions[lastIndex], clientHelloSpec.Extensions[ln-1] = clientHelloSpec.Extensions[ln-1], clientHelloSpec.Extensions[lastIndex]
	}

	return &clientHelloSpec, nil
}

func (t *Transport) createExtension(extensionId uint16) (utls.TLSExtension, bool) {
	// 返回 (extension, isConcrete) — isConcrete 表示我们创建了具体的 uTLS 扩展类型（true）
	switch extensionId {
	case 0: // SNI
		// ServerName 会由 utls.Config.ServerName 提供，SNI 结构留空即可
		return &utls.SNIExtension{}, true
	case 5: // status_request (OCSP stapling)
		return &utls.StatusRequestExtension{}, true
	case 10: // supported_groups (named curves)
		// 实际曲线由 createCurves 提供，此处构造空实例作为占位
		return &utls.SupportedCurvesExtension{}, true
	case 11: // ec_point_formats
		return &utls.SupportedPointsExtension{}, true
	case 13: // signature_algorithms
		ext := &utls.SignatureAlgorithmsExtension{
			SupportedSignatureAlgorithms: []utls.SignatureScheme{
				utls.ECDSAWithP256AndSHA256,
				utls.PSSWithSHA256,
				utls.PKCS1WithSHA256,
				utls.ECDSAWithP384AndSHA384,
				utls.PSSWithSHA384,
				utls.PKCS1WithSHA384,
				utls.PSSWithSHA512,
				utls.PKCS1WithSHA512,
			},
		}
		return ext, true
	case 16: // ALPN
		// 常见顺序： h2, http/1.1 — 保持此默认
		return &utls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}}, true
	case 17:
		return &utls.StatusRequestV2Extension{}, true
	case 18:
		return &utls.SCTExtension{}, true
	case 21: // padding
		ext := &utls.UtlsPaddingExtension{}
		ext.GetPaddingLen = utls.BoringPaddingStyle
		return ext, true
	case 23: // extended_master_secret
		return &utls.ExtendedMasterSecretExtension{}, true
	case 24:
		return &utls.FakeTokenBindingExtension{}, true
	case 27: // certificate_compression
		return &utls.UtlsCompressCertExtension{Algorithms: []utls.CertCompressionAlgo{utls.CertCompressionBrotli}}, true
	case 28:
		return &utls.FakeRecordSizeLimitExtension{}, true
	case 34:
		return &utls.FakeDelegatedCredentialsExtension{}, true
	case 35: // session_ticket
		return &utls.SessionTicketExtension{}, true
	case 41: // pre_shared_key (PSK)
		return &utls.UtlsPreSharedKeyExtension{}, true
	case 43: // supported_versions
		// 实际版本序列由 createTlsVersion 生成并传入 createExtensions（你现有代码里会这样）
		return &utls.SupportedVersionsExtension{}, true
	case 44: // cookie
		return &utls.CookieExtension{}, true
	case 45: // psk_key_exchange_modes
		return &utls.PSKKeyExchangeModesExtension{Modes: []uint8{utls.PskModeDHE}}, true
	case 50:
		return &utls.SignatureAlgorithmsCertExtension{
			SupportedSignatureAlgorithms: []utls.SignatureScheme{
				utls.ECDSAWithP256AndSHA256,
				utls.ECDSAWithP384AndSHA384,
				utls.ECDSAWithP521AndSHA512,
				utls.PSSWithSHA256,
				utls.PSSWithSHA384,
				utls.PSSWithSHA512,
				utls.PKCS1WithSHA256,
				utls.PKCS1WithSHA384,
				utls.PKCS1WithSHA512,
				utls.ECDSAWithSHA1,
				utls.PKCS1WithSHA1,
			},
		}, true
	case 51: // key_share
		// 默认 keyshares：GREASE placeholder + X25519 kyber draft + X25519（与较新 Chrome 行为相似）
		return &utls.KeyShareExtension{
			KeyShares: []utls.KeyShare{
				{Group: utls.CurveID(utls.GREASE_PLACEHOLDER), Data: []byte{0}},
				{Group: utls.X25519Kyber768Draft00},
				{Group: utls.X25519},
			},
		}, true
	case 57:
		return &utls.QUICTransportParametersExtension{}, true
	case 13172:
		return &utls.NPNExtension{}, true
	case 17513:
		return &utls.ApplicationSettingsExtension{SupportedProtocols: []string{"h2", "http/1.1"}}, true
	case 30031: // fake channel id old id
		return &utls.FakeChannelIDExtension{OldExtensionID: true}, true
	case 30032:
		return &utls.FakeChannelIDExtension{}, true
	case 65037: // GREASE ECH (encrypted client hello grease)
		return utls.BoringGREASEECH(), true
	case 65281: // renegotiation_info
		return &utls.RenegotiationInfoExtension{Renegotiation: utls.RenegotiateOnceAsClient}, true
	default:
		// 未知扩展 -> 返回 GenericExtension，使用户可以在 JA3 中指定任意扩展 id
		return &utls.GenericExtension{Id: extensionId}, false
	}
}

func (t *Transport) createExtensions(extensions []string, tlsExtension, curvesExtension, pointExtension utls.TLSExtension) ([]utls.TLSExtension, error) {
	allExtensions := []utls.TLSExtension{}
	for i, extension := range extensions {
		var extensionId uint16
		if n, err := strconv.ParseUint(extension, 10, 16); err != nil {
			return nil, errors.New("ja3Str extension error,utls not support: " + extension)
		} else {
			extensionId = uint16(n)
		}
		var ext utls.TLSExtension
		switch extensionId {
		case 10:
			ext = curvesExtension
		case 11:
			ext = pointExtension
		case 43:
			ext = tlsExtension
		default:
			ext, _ = t.createExtension(extensionId)
			if ext == nil {
				ext = &utls.GenericExtension{Id: extensionId}
			}
		}
		// insert GREASE at first position if the first listed extension isn't a grease extension
		if i == 0 {
			if _, ok := ext.(*utls.UtlsGREASEExtension); !ok {
				allExtensions = append(allExtensions, &utls.UtlsGREASEExtension{})
			}
		}
		allExtensions = append(allExtensions, ext)
	}
	// ensure last extension is grease as in original logic
	if l := len(allExtensions); l > 0 {
		if _, ok := allExtensions[l-1].(*utls.UtlsGREASEExtension); !ok {
			allExtensions = append(allExtensions, &utls.UtlsGREASEExtension{})
		}
	}
	return allExtensions, nil
}

func (t *Transport) createPointFormats(points []string) (utls.TLSExtension, error) {
	supportedPoints := []uint8{}
	for _, val := range points {
		if n, err := strconv.ParseUint(val, 10, 8); err != nil {
			return nil, errors.New("ja3Str point error")
		} else {
			supportedPoints = append(supportedPoints, uint8(n))
		}
	}
	return &utls.SupportedPointsExtension{SupportedPoints: supportedPoints}, nil
}

func (t *Transport) createCurves(curves []string) (utls.TLSExtension, error) {
	curveIds := []utls.CurveID{}
	for i, val := range curves {
		var curveId utls.CurveID
		if n, err := strconv.ParseUint(val, 10, 16); err != nil {
			return nil, errors.New("ja3Str curves error")
		} else {
			curveId = utls.CurveID(uint16(n))
		}
		if i == 0 {
			if curveId != utls.GREASE_PLACEHOLDER {
				curveIds = append(curveIds, utls.GREASE_PLACEHOLDER)
			}
		}
		curveIds = append(curveIds, curveId)
	}
	return &utls.SupportedCurvesExtension{Curves: curveIds}, nil
}

func (t *Transport) createCiphers(ciphers []string) ([]uint16, error) {
	cipherSuites := []uint16{}
	for i, val := range ciphers {
		var cipherSuite uint16
		if n, err := strconv.ParseUint(val, 10, 16); err != nil {
			return nil, errors.New("ja3Str cipherSuites error")
		} else {
			cipherSuite = uint16(n)
		}
		if i == 0 {
			if cipherSuite != utls.GREASE_PLACEHOLDER {
				cipherSuites = append(cipherSuites, utls.GREASE_PLACEHOLDER)
			}
		}
		cipherSuites = append(cipherSuites, cipherSuite)
	}
	return cipherSuites, nil
}

func (t *Transport) createTlsVersion(ver uint16) (tlsMaxVersion uint16, tlsMinVersion uint16, tlsSupport utls.TLSExtension, err error) {
	switch ver {
	case utls.VersionTLS13:
		// 常见客户端偏好：GREASE, TLS1.3, TLS1.2
		tlsMaxVersion = utls.VersionTLS13
		tlsMinVersion = utls.VersionTLS12
		tlsSupport = &utls.SupportedVersionsExtension{
			Versions: []uint16{
				utls.GREASE_PLACEHOLDER,
				utls.VersionTLS13,
				utls.VersionTLS12,
			},
		}
	case utls.VersionTLS12:
		// GREASE, TLS1.2, TLS1.1
		tlsMaxVersion = utls.VersionTLS12
		tlsMinVersion = utls.VersionTLS10
		tlsSupport = &utls.SupportedVersionsExtension{
			Versions: []uint16{
				utls.GREASE_PLACEHOLDER,
				utls.VersionTLS12,
				utls.VersionTLS11,
			},
		}
	case utls.VersionTLS11:
		// GREASE, TLS1.1, TLS1.0
		tlsMaxVersion = utls.VersionTLS11
		tlsMinVersion = utls.VersionTLS10
		tlsSupport = &utls.SupportedVersionsExtension{
			Versions: []uint16{
				utls.GREASE_PLACEHOLDER,
				utls.VersionTLS11,
				utls.VersionTLS10,
			},
		}
	case utls.VersionTLS10:
		// 较旧：只报 TLS1.0
		tlsMaxVersion = utls.VersionTLS10
		tlsMinVersion = utls.VersionTLS10
		tlsSupport = &utls.SupportedVersionsExtension{
			Versions: []uint16{
				utls.VersionTLS10,
			},
		}
	default:
		err = errors.New("ja3Str tls version error")
	}
	return
}

func (t *Transport) getExtensionId(extension utls.TLSExtension) (uint16, uint8) {
	switch ext := extension.(type) {
	case *utls.SNIExtension:
		return 0, 0
	case *utls.StatusRequestExtension:
		return 5, 0
	case *utls.SupportedCurvesExtension:
		return 10, 0
	case *utls.SupportedPointsExtension:
		return 11, 0
	case *utls.SignatureAlgorithmsExtension:
		return 13, 0
	case *utls.ALPNExtension:
		return 16, 0
	case *utls.StatusRequestV2Extension:
		return 17, 0
	case *utls.SCTExtension:
		return 18, 0
	case *utls.UtlsPaddingExtension:
		return 21, 0
	case *utls.ExtendedMasterSecretExtension:
		return 23, 0
	case *utls.FakeTokenBindingExtension:
		return 24, 0
	case *utls.UtlsCompressCertExtension:
		return 27, 0
	case *utls.FakeDelegatedCredentialsExtension:
		return 34, 0
	case *utls.SessionTicketExtension:
		return 35, 0
	case *utls.UtlsPreSharedKeyExtension:
		return 41, 0
	case *utls.SupportedVersionsExtension:
		return 43, 0
	case *utls.CookieExtension:
		return 44, 0
	case *utls.PSKKeyExchangeModesExtension:
		return 45, 0
	case *utls.SignatureAlgorithmsCertExtension:
		return 50, 0
	case *utls.KeyShareExtension:
		return 51, 0
	case *utls.QUICTransportParametersExtension:
		return 57, 0
	case *utls.NPNExtension:
		return 13172, 0
	case *utls.ApplicationSettingsExtension:
		return 17513, 0
	case *utls.FakeChannelIDExtension:
		// 两个 fake channel id id (30031/30032) 均映射到 FakeChannelIDExtension；用 OldExtensionID 字段判断
		if ext.OldExtensionID {
			return 30031, 0
		}
		return 30032, 0
	case *utls.FakeRecordSizeLimitExtension:
		return 28, 0
	case *utls.GREASEEncryptedClientHelloExtension:
		return 65037, 0
	case *utls.RenegotiationInfoExtension:
		return 65281, 0
	case *utls.GenericExtension:
		return ext.Id, 1
	case *utls.UtlsGREASEExtension:
		// GREASE 占位：返回 0 并标记为 GREASE 类型
		return 0, 2
	default:
		return 0, 3
	}
}
