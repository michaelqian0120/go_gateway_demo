package reverse_proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/middleware"
	"go_gateway_demo/reverse_proxy/load_balance"
)

func NewLoadBalanceReverseProxy(c *gin.Context, lb load_balance.LoadBalance, trans *http.Transport) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		nextAddr, err := lb.Get(req.RemoteAddr)
		if err != nil {
			panic("get next addr fail: " + err.Error())
		}
		target, err := url.Parse(nextAddr)
		if err != nil {
			panic("parse next addr fail: " + err.Error())
		}
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		if _, ok := req.Header["User-Agent"]; !ok {
			// 不让默认的User-Agent被加上
			req.Header.Set("User-Agent", "")
		}
	}

	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
	}

	return &httputil.ReverseProxy{Director: director, Transport: trans, ErrorHandler: errFunc}
}
