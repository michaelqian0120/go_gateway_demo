package http_proxy_router

import (
	"context"
	"github.com/gin-gonic/gin"
	"go_gateway_demo/middleware"
	"golang_common/lib"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	HttpSrvHandler  *http.Server
	HttpsSrvHandler *http.Server
)

func newServer(confPrefix string) *http.Server {
	gin.SetMode(lib.ConfBase.DebugMode)
	r := InitRouter(middleware.RecoveryMiddleware())
	return &http.Server{
		Addr:           lib.GetStringConf(confPrefix + ".addr"),
		Handler:        r,
		ReadTimeout:    time.Duration(lib.GetIntConf(confPrefix+".read_timeout")) * time.Second,
		WriteTimeout:   time.Duration(lib.GetIntConf(confPrefix+".write_timeout")) * time.Second,
		MaxHeaderBytes: 1 << uint(lib.GetIntConf(confPrefix+".max_header_bytes")),
	}
}

func HttpServerRun() {
	HttpSrvHandler = newServer("proxy.http")
	go func() {
		log.Printf(" [INFO] HttpProxyRun:%s\n", HttpSrvHandler.Addr)
		if err := HttpSrvHandler.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf(" [ERROR] HttpProxyRun:%s err:%v\n", HttpSrvHandler.Addr, err)
		}
	}()
}

// HttpsServerRun 未配置https地址或证书文件不存在时不启动，只记录日志
func HttpsServerRun() {
	addr := lib.GetStringConf("proxy.https.addr")
	certFile := lib.GetStringConf("proxy.https.cert_file")
	keyFile := lib.GetStringConf("proxy.https.key_file")
	if addr == "" {
		return
	}
	for _, f := range []string{certFile, keyFile} {
		if _, err := os.Stat(f); err != nil {
			log.Printf(" [WARN] HttpsProxyRun skipped, cert file %q: %v\n", f, err)
			return
		}
	}

	HttpsSrvHandler = newServer("proxy.https")
	go func() {
		log.Printf(" [INFO] HttpsProxyRun:%s\n", addr)
		if err := HttpsSrvHandler.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf(" [ERROR] HttpsProxyRun:%s err:%v\n", addr, err)
		}
	}()
}

func HttpServerStop() {
	stopServer(HttpSrvHandler, "HttpProxyStop")
}

func HttpsServerStop() {
	stopServer(HttpsSrvHandler, "HttpsProxyStop")
}

func stopServer(srv *http.Server, name string) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf(" [ERROR] %s err:%v\n", name, err)
		return
	}
	log.Printf(" [INFO] %s stopped\n", name)
}
