package services

import (
	"b0gus/configs"
	"b0gus/databases"
	"bytes"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type HTTPservConf struct {
	db      *configs.RuntimeDB
	r       *gin.Engine
	ConfObj *atomic.Pointer[configs.LocalConfig]
}

// httpHook will intercept any requests sent by any kinds of http methods
// cookies will be compressed and then recorded.
func (hs *HTTPservConf) httpHook(c *gin.Context) {
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	cookies := make(map[string]string)
	for _, cookie := range c.Request.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	pbObj := &StrMapForHTTP{
		Cookie:    cookies,
		UserAgent: c.Request.UserAgent(),
	}
	defer c.Next()
	serialized, err := proto.Marshal(pbObj)
	if err != nil {
		configs.Logger.Error(err.Error())
		return
	}
	c.Request.UserAgent()
	_ = hs.db.CreateOrUpdateItemsInSeq([]configs.DBstruct{
		&databases.AddrInfo{Ip: c.RemoteIP()},
		&databases.HttpInfo{
			Method:      c.Request.Method,
			Path:        c.Request.URL.Path,
			Request:     string(bodyBytes),
			WrappedInfo: serialized,
		},
	}...)
}

func (hs *HTTPservConf) InvokeForTCPtask(net.Conn)                {}
func (hs *HTTPservConf) InvokeForUDPtask(net.Addr, []byte) []byte { return nil }
func (hs *HTTPservConf) InvokeForICMPtask(net.Addr, []byte)       {}
func (hs *HTTPservConf) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hs.r.ServeHTTP(w, r)
}

func (hs *HTTPservConf) Run(
	ConfObj *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
	db *configs.RuntimeDB, args ...any,
) {
	defer func() {
		configs.Logger.Info(configs.GetLocalizedMsg(
			"services.HTTPQuitInfo", nil,
		))
	}()
	if ConfObj == nil {
		configs.Logger.Error(configs.GetLocalizedMsg(
			"services.HTTPNullConfErr", nil,
		))
		return
	} else if len(args) > 1 {
		payload := configs.GetLocalizedMsg(
			"services.HTTPWrongParamNumErr",
			map[string]any{
				"Expect": 0,
				"Actual": len(args),
			},
		)
		configs.Logger.Error(payload)
		return
	}
	metaConf, ok := ConfObj.Load().
		SelectTerm(configs.HTTPEnum).(configs.HTTPconfig)
	if !ok {
		configs.Logger.Error(configs.GetLocalizedMsg(
			"services.HTTPConfLoadErr", nil,
		))
		return
	}
	hs.db = db
	_ = hs.db.CreateTable(&databases.HttpInfo{})
	var sb strings.Builder
	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(int(metaConf.ListenPort)))

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	r := gin.Default()
	r.Use(
		hs.httpHook,
		ginLogger(),
		// abort ginRecovery
	)
	r.GET("/robots.txt", func(c *gin.Context) {
		c.String(
			http.StatusOK,
			`User-agent: *
Disallow: /admin
Disallow: /login
Disallow: /comment
Disallow: /favicon.ico
Allow: /sitemap.txt
Allow: /sitemap.xml
`,
		)
	})
	r.NoRoute(httpInterceptor)
	hs.r = r
	var clientAux = ReentrantNetType{}
	clientAux.Init(configs.HTTPEnum, hs)
	go clientAux.EventMonitor(ConfObj, scc)
	clientAux.AlterNetFd(HTTPEnum, ConfObj)
}

func ginLogger() gin.HandlerFunc { return httpLogger }
func httpLogger(c *gin.Context) {
	start := time.Now()
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery
	c.Next()
	cost := time.Since(start)
	configs.Logger.Info(path,
		zap.Int("code", c.Writer.Status()),
		zap.String("meth", c.Request.Method),
		zap.String("path", path),
		zap.String("query", query),
		zap.String("from", c.ClientIP()),
		zap.Duration("cast", cost),
		zap.String("err", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		zap.String("ua", c.Request.UserAgent()),
	)
}

// httpInterceptor is used for
func httpInterceptor(c *gin.Context) {
	/* TODO: write something like these definitions which are pasted from
	         https://github.com/pathei-kosmos/ghoney/blob/master/main.go. Attack pattern is important

	if strings.Contains(normalized, "../") || strings.Contains(normalized, "..\\") {
		attackType = "Path Traversal"
		details = "Detected '../' in path, query, or body"
	}

	// SQL Injection (checks query and body snippet)
	// Ensure attackType is not overwritten if Path Traversal was already found

	// XML Bomb (checks body snippet)

	// Ensure attackType is not overwritten

	// Command Injection

	// SSRF

	// LFI/RFI

	// Logging based on detection or honeypot access
	isHoneypotPath := path == "/admin" || path == "/api/v1/auth" || path == "/.git/config"
	*/
	c.JSON(http.StatusOK, gin.H{
		"meth": c.Request.Method,
		"msg":  "ok",
	})
}
