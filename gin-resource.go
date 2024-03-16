package utils

import (
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type GinConfig struct {
	TemplateEmbed    embed.FS
	TemplatePathBase string

	StaticEmbed    embed.FS
	StaticPathBase string
	StaticUrlBase  string

	StaticMiddlewares []gin.HandlerFunc

	CacheMinute  int
	Debug        bool
	AccessLogger io.Writer
	ErrorLogger  io.Writer
}

type Gin struct {
	config *GinConfig
	tmpl   *template.Template
	*gin.Engine
}

func NewGin(c GinConfig) *Gin {
	if c.AccessLogger == nil {
		gin.DefaultWriter = &NullWriter{}
	} else {
		gin.DefaultWriter = c.AccessLogger
	}
	if c.ErrorLogger == nil {
		gin.DefaultErrorWriter = &NullWriter{}
	} else {
		gin.DefaultErrorWriter = c.ErrorLogger
	}
	if c.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	g := gin.New()
	g.Use(gin.Recovery())
	return newGinServer(g, c)
}

func NewDefaultGin(c GinConfig) *Gin {
	if c.AccessLogger == nil {
		gin.DefaultWriter = &NullWriter{}
	} else {
		gin.DefaultWriter = c.AccessLogger
	}
	if c.ErrorLogger == nil {
		gin.DefaultErrorWriter = &NullWriter{}
	} else {
		gin.DefaultErrorWriter = c.ErrorLogger
	}

	if c.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	return newGinServer(gin.Default(), c)
}

func newGinServer(engine *gin.Engine, c GinConfig) *Gin {
	c.StaticPathBase = strings.Trim(c.StaticPathBase, "/")
	c.StaticUrlBase = strings.Trim(c.StaticUrlBase, "/")
	c.TemplatePathBase = strings.Trim(c.TemplatePathBase, "/")

	server := &Gin{config: &c, Engine: engine}
	server.serve()
	return server
}

func (r *Gin) serve() {
	var (
		lastModify  string
		assetGroup  *gin.RouterGroup
		middlewares = make([]gin.HandlerFunc, 0)
	)

	r.HTMLRender = r

	lastModify = time.Now().UTC().Format(time.RFC1123)

	middlewares = append(middlewares, r.config.StaticMiddlewares...)
	middlewares = append(middlewares, func(c *gin.Context) {
		if r.config.CacheMinute > 0 {
			c.Header("Expires", time.Now().UTC().Add(time.Minute*time.Duration(r.config.CacheMinute)).Format(time.RFC1123))
			c.Header("Cache-Control", "max-age="+strconv.Itoa(r.config.CacheMinute*60)+", must-revalidate")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		c.Header("Last-Modified", lastModify)
		last := c.Request.Header.Get("If-Modified-Since")
		if len(last) > 0 && last == lastModify {
			c.Status(http.StatusNotModified)
			c.Abort()
		}
	})

	assetGroup = r.Group(r.config.StaticUrlBase)
	if len(middlewares) > 0 {
		assetGroup.Use(middlewares...)
	}
	assetGroup.StaticFS("/", http.FS(r))
}

func (r *Gin) Instance(filename string, data interface{}) render.Render {
	var (
		err  error
		tmpl *template.Template
	)
	if r.config.Debug {
		if tmpl, err = r.loadTemplate(filename); err != nil {
			panic(err)
		}
	} else {
		if r.tmpl == nil {
			if r.tmpl, err = r.loadPathTemplates(r.config.TemplatePathBase); err != nil {
				panic(err)
			}
		}
		tmpl = r.tmpl
	}
	return render.HTML{
		Template: tmpl,
		Name:     filename,
		Data:     data,
	}
}

func (r *Gin) loadTemplate(name string) (*template.Template, error) {
	return r.parseTemplates([]string{name})
}
func (r *Gin) loadPathTemplates(path string) (*template.Template, error) {
	var (
		err   error
		files []string
	)
	if files, err = r.getAllTemplateFiles(path); err != nil {
		return nil, err
	}
	return r.parseTemplates(files)
}
func (r *Gin) parseTemplates(files []string) (*template.Template, error) {
	var (
		err   error
		bytes []byte
		tmpl  = template.New("").Funcs(r.FuncMap)
	)
	for _, name := range files {
		bytes = nil
		//调试模式下尽可能加载硬盘文件
		if r.config.Debug {
			bytes, _ = ioutil.ReadFile(r.config.TemplatePathBase + "/" + name)
		}
		if bytes == nil {
			if bytes, err = r.config.TemplateEmbed.ReadFile(r.config.TemplatePathBase + "/" + name); err != nil {
				return nil, err
			}
		}
		if tmpl, err = tmpl.New(name).Parse(string(bytes)); err != nil {
			return nil, err
		}
	}
	return tmpl, nil
}

func (r *Gin) getAllTemplateFiles(path string) ([]string, error) {
	var (
		err   error
		files = make([]string, 0)
		dirs  []fs.DirEntry
	)

	if path == "" {
		path = "."
	} else {
		path = strings.TrimRight(path, "/")
	}

	if dirs, err = r.config.TemplateEmbed.ReadDir(path); err != nil {
		return files, err
	}

	for _, v := range dirs {
		fullPath := strings.TrimLeft(path+"/"+v.Name(), "./")
		if v.IsDir() {
			if deepFiles, err := r.getAllTemplateFiles(fullPath); err != nil {
				return files, err
			} else {
				for _, val := range deepFiles {
					files = append(files, v.Name()+"/"+val)
				}
			}
		} else {
			files = append(files, v.Name())
		}
	}

	return files, nil
}
func (r *Gin) GetConfig() GinConfig {
	return *r.config
}

func (r *Gin) Open(name string) (fs.File, error) {
	var file *os.File
	if name == "." {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	//调试模式下尽可能加载硬盘文件
	if r.config.Debug {
		file, _ = os.Open(r.config.StaticPathBase + "/" + name)
	}
	if file != nil {
		return file, nil
	} else {
		return r.config.StaticEmbed.Open(r.config.StaticPathBase + "/" + name)
	}
}
