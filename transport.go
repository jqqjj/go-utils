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
	"strconv"
	"strings"
)

func NewTransport(ja3 string) (*Transport, error) {
	c := &Transport{}
	if ja3 != "" {
		var err error
		if c.spec, err = c.createSpecWithStr(ja3); err != nil {
			return nil, err
		}
	}
	return c, nil
}

type Transport struct {
	tr1  http.Transport
	tr2  http2.Transport
	spec *utls.ClientHelloSpec
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Scheme {
	case "https":
		if t.spec == nil {
			return t.tr1.RoundTrip(req)
		} else {
			return t.httpsRoundTrip(req)
		}
	case "http":
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

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", req.URL.Host, port))
	if err != nil {
		return nil, fmt.Errorf("tcp net dial fail: %w", err)
	}
	defer conn.Close() // nolint

	tlsConn, err := t.tlsConnect(conn, req)
	if err != nil {
		return nil, fmt.Errorf("tls connect fail: %w", err)
	}

	httpVersion := tlsConn.ConnectionState().NegotiatedProtocol
	switch httpVersion {
	case "h2":
		conn, err := t.tr2.NewClientConn(tlsConn)
		if err != nil {
			return nil, fmt.Errorf("create http2 client with connection fail: %w", err)
		}
		defer conn.Close() // nolint
		return conn.RoundTrip(req)
	case "http/1.1", "":
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
		tlsConn = utls.UClient(conn, t.getTLSConfig(req.URL.Host), utls.HelloCustom)
		lastIndex := -1
		for i, v := range t.spec.Extensions {
			if id, _ := t.getExtensionId(v); id == 41 {
				lastIndex = i
			}
		}
		ln := len(t.spec.Extensions)
		if lastIndex != -1 {
			t.spec.Extensions[lastIndex], t.spec.Extensions[ln-1] = t.spec.Extensions[ln-1], t.spec.Extensions[lastIndex]
		}
		if err = tlsConn.ApplyPreset(t.spec); err != nil {
			return nil, err
		}
	} else {
		tlsConn = utls.UClient(conn, t.getTLSConfig(req.URL.Host), utls.HelloRandomized)
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

	lastIndex := -1
	for i, v := range clientHelloSpec.Extensions {
		if id, _ := t.getExtensionId(v); id == 41 {
			lastIndex = i
		}
	}
	ln := len(clientHelloSpec.Extensions)
	if lastIndex != -1 {
		clientHelloSpec.Extensions[lastIndex], clientHelloSpec.Extensions[ln-1] = clientHelloSpec.Extensions[ln-1], clientHelloSpec.Extensions[lastIndex]
	}

	return &clientHelloSpec, nil
}
func (t *Transport) createExtension(extensionId uint16) (utls.TLSExtension, bool) {
	var option struct {
		data []byte
		ext  utls.TLSExtension
	}
	switch extensionId {
	case 0:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SNIExtension))
			return &extV, true
		}
		extV := new(utls.SNIExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 5:
		if option.ext != nil {
			extV := *(option.ext.(*utls.StatusRequestExtension))
			return &extV, true
		}
		extV := new(utls.StatusRequestExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 10:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SupportedCurvesExtension))
			return &extV, true
		}
		extV := new(utls.SupportedCurvesExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 11:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SupportedPointsExtension))
			return &extV, true
		}
		extV := new(utls.SupportedPointsExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 13:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SignatureAlgorithmsExtension))
			return &extV, true
		}
		extV := new(utls.SignatureAlgorithmsExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.SupportedSignatureAlgorithms = []utls.SignatureScheme{
				utls.ECDSAWithP256AndSHA256,
				utls.PSSWithSHA256,
				utls.PKCS1WithSHA256,
				utls.ECDSAWithP384AndSHA384,
				utls.PSSWithSHA384,
				utls.PKCS1WithSHA384,
				utls.PSSWithSHA512,
				utls.PKCS1WithSHA512,
			}
		}
		return extV, true
	case 16:
		if option.ext != nil {
			extV := *(option.ext.(*utls.ALPNExtension))
			exts := []string{}
			for _, alp := range extV.AlpnProtocols {
				if alp != "" {
					exts = append(exts, alp)
				}
			}
			extV.AlpnProtocols = exts
			return &extV, true
		}
		extV := new(utls.ALPNExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.AlpnProtocols = []string{"h2", "http/1.1"}
		}
		return extV, true
	case 17:
		if option.ext != nil {
			extV := *(option.ext.(*utls.StatusRequestV2Extension))
			return &extV, true
		}
		extV := new(utls.StatusRequestV2Extension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 18:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SCTExtension))
			return &extV, true
		}
		extV := new(utls.SCTExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 21:
		if option.ext != nil {
			extV := *(option.ext.(*utls.UtlsPaddingExtension))
			return &extV, true
		}
		extV := new(utls.UtlsPaddingExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.GetPaddingLen = utls.BoringPaddingStyle
		}
		return extV, true
	case 23:
		if option.ext != nil {
			extV := *(option.ext.(*utls.ExtendedMasterSecretExtension))
			return &extV, true
		}
		extV := new(utls.ExtendedMasterSecretExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 24:
		if option.ext != nil {
			extV := *(option.ext.(*utls.FakeTokenBindingExtension))
			return &extV, true
		}
		extV := new(utls.FakeTokenBindingExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 27:
		if option.ext != nil {
			extV := *(option.ext.(*utls.UtlsCompressCertExtension))
			return &extV, true
		}
		extV := new(utls.UtlsCompressCertExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.Algorithms = []utls.CertCompressionAlgo{utls.CertCompressionBrotli}
		}
		return extV, true
	case 28:
		if option.ext != nil {
			extV := *(option.ext.(*utls.FakeRecordSizeLimitExtension))
			return &extV, true
		}
		extV := new(utls.FakeRecordSizeLimitExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 34:
		if option.ext != nil {
			extV := *(option.ext.(*utls.FakeDelegatedCredentialsExtension))
			return &extV, true
		}
		extV := new(utls.FakeDelegatedCredentialsExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 35:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SessionTicketExtension))
			return &extV, true
		}
		extV := new(utls.SessionTicketExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 41:
		if option.ext != nil {
			extV := *(option.ext.(*utls.UtlsPreSharedKeyExtension))
			return &extV, true
		}
		extV := new(utls.UtlsPreSharedKeyExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 43:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SupportedVersionsExtension))
			return &extV, true
		}
		extV := new(utls.SupportedVersionsExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 44:
		if option.ext != nil {
			extV := *(option.ext.(*utls.CookieExtension))
			return &extV, true
		}
		extV := new(utls.CookieExtension)
		if option.data != nil {
			extV.Cookie = option.data
		}
		return extV, true
	case 45:
		if option.ext != nil {
			extV := *(option.ext.(*utls.PSKKeyExchangeModesExtension))
			return &extV, true
		}
		extV := new(utls.PSKKeyExchangeModesExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.Modes = []uint8{utls.PskModeDHE}
		}
		return extV, true
	case 50:
		if option.ext != nil {
			extV := *(option.ext.(*utls.SignatureAlgorithmsCertExtension))
			return &extV, true
		}
		extV := new(utls.SignatureAlgorithmsCertExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.SupportedSignatureAlgorithms = []utls.SignatureScheme{
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
			}
		}
		return extV, true
	case 51:
		extV := new(utls.KeyShareExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.KeyShares = []utls.KeyShare{
				{Group: utls.CurveID(utls.GREASE_PLACEHOLDER), Data: []byte{0}},
				{Group: utls.X25519Kyber768Draft00},
				{Group: utls.X25519},
			}
		}
		return extV, true
	case 57:
		if option.ext != nil {
			extV := *(option.ext.(*utls.QUICTransportParametersExtension))
			return &extV, true
		}
		return new(utls.QUICTransportParametersExtension), true
	case 13172:
		if option.ext != nil {
			extV := *(option.ext.(*utls.NPNExtension))
			return &extV, true
		}
		extV := new(utls.NPNExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 17513:
		if option.ext != nil {
			extV := *(option.ext.(*utls.ApplicationSettingsExtension))
			return &extV, true
		}
		extV := new(utls.ApplicationSettingsExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.SupportedProtocols = []string{"h2", "http/1.1"}
		}
		return extV, true
	case 30031:
		if option.ext != nil {
			extV := *(option.ext.(*utls.FakeChannelIDExtension))
			return &extV, true
		}
		extV := new(utls.FakeChannelIDExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.OldExtensionID = true
		}
		return extV, true
	case 30032:
		if option.ext != nil {
			extV := *(option.ext.(*utls.FakeChannelIDExtension))
			return &extV, true
		}
		extV := new(utls.FakeChannelIDExtension)
		if option.data != nil {
			extV.Write(option.data)
		}
		return extV, true
	case 65037:
		if option.ext != nil {
			return option.ext, true
		}
		return utls.BoringGREASEECH(), true
	case 65281:
		if option.ext != nil {
			extV := *(option.ext.(*utls.RenegotiationInfoExtension))
			return &extV, true
		}
		extV := new(utls.RenegotiationInfoExtension)
		if option.data != nil {
			extV.Write(option.data)
		} else {
			extV.Renegotiation = utls.RenegotiateOnceAsClient
		}
		return extV, true
	default:
		if option.data != nil {
			return &utls.GenericExtension{
				Id:   extensionId,
				Data: option.data,
			}, false
		}
		return option.ext, false
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
		if i == 0 {
			if _, ok := ext.(*utls.UtlsGREASEExtension); !ok {
				allExtensions = append(allExtensions, &utls.UtlsGREASEExtension{})
			}
		}
		allExtensions = append(allExtensions, ext)
	}
	if l := len(allExtensions); l > 0 {
		if _, ok := allExtensions[l-1].(*utls.UtlsGREASEExtension); !ok {
			allExtensions = append(allExtensions, &utls.UtlsGREASEExtension{})
		}
	}
	return allExtensions, nil
}
func (t *Transport) createPointFormats(points []string) (curvesExtension utls.TLSExtension, err error) {
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
func (t *Transport) createCurves(curves []string) (curvesExtension utls.TLSExtension, err error) {
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
func (t *Transport) createTlsVersion(ver uint16) (tlsMaxVersion uint16, tlsMinVersion uint16, tlsSuppor utls.TLSExtension, err error) {
	switch ver {
	case utls.VersionTLS13:
		tlsMaxVersion = utls.VersionTLS13
		tlsMinVersion = utls.VersionTLS12
		tlsSuppor = &utls.SupportedVersionsExtension{
			Versions: []uint16{
				utls.GREASE_PLACEHOLDER,
				utls.VersionTLS13,
				utls.VersionTLS12,
			},
		}
	case utls.VersionTLS12:
		tlsMaxVersion = utls.VersionTLS12
		tlsMinVersion = utls.VersionTLS11
		tlsSuppor = &utls.SupportedVersionsExtension{
			Versions: []uint16{
				utls.GREASE_PLACEHOLDER,
				utls.VersionTLS12,
				utls.VersionTLS11,
			},
		}
	case utls.VersionTLS11:
		tlsMaxVersion = utls.VersionTLS11
		tlsMinVersion = utls.VersionTLS10
		tlsSuppor = &utls.SupportedVersionsExtension{
			Versions: []uint16{
				utls.GREASE_PLACEHOLDER,
				utls.VersionTLS11,
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
		if ext.OldExtensionID {
			return 30031, 0
		} else {
			return 30031, 0
		}
	case *utls.FakeRecordSizeLimitExtension:
		return 28, 0
	case *utls.GREASEEncryptedClientHelloExtension:
		return 65037, 0
	case *utls.RenegotiationInfoExtension:
		return 65281, 0
	case *utls.GenericExtension:
		return ext.Id, 1
	case *utls.UtlsGREASEExtension:
		return 0, 2
	default:
		return 0, 3
	}
}
