package http_proxy_middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPProtocolCheckMiddleware 校验服务是否支持当前请求的协议
// need_https=0 的服务拒绝从https端口进来的请求；need_websocket=0 的服务拒绝websocket升级请求
func HTTPProtocolCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		if c.Request.TLS != nil && service.HTTPRule.NeedHttps != 1 {
			middleware.ResponseError(c, 4001, fmt.Errorf("service %s not support https", service.Info.ServiceName))
			c.Abort()
			return
		}
		if isWebsocketUpgrade(c.Request) && service.HTTPRule.NeedWebsocket != 1 {
			middleware.ResponseError(c, 4002, fmt.Errorf("service %s not support websocket", service.Info.ServiceName))
			c.Abort()
			return
		}
		c.Next()
	}
}

func isWebsocketUpgrade(req *http.Request) bool {
	return headerContainsToken(req.Header, "Connection", "upgrade") &&
		strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
}

// headerContainsToken Connection 头可能是 "keep-alive, Upgrade" 这样的多值形式
func headerContainsToken(h http.Header, name, token string) bool {
	for _, v := range h.Values(name) {
		for _, item := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(item), token) {
				return true
			}
		}
	}
	return false
}
