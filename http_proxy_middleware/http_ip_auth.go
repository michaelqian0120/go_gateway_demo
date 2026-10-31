package http_proxy_middleware

import (
	"errors"
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPWhiteListMiddleware 开启权限且配置了白名单时，只允许白名单内的IP访问
func HTTPWhiteListMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		whiteList := splitRules(service.AccessControl.WhiteList)
		if service.AccessControl.OpenAuth == 1 && len(whiteList) > 0 {
			clientIP := remoteIP(c)
			if !inList(clientIP, whiteList) {
				middleware.ResponseError(c, 3001, fmt.Errorf("%s not in white ip list", clientIP))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// HTTPBlackListMiddleware 开启权限且未配置白名单时，拒绝黑名单内的IP；配置了白名单则以白名单为准
func HTTPBlackListMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		whiteList := splitRules(service.AccessControl.WhiteList)
		blackList := splitRules(service.AccessControl.BlackList)
		if service.AccessControl.OpenAuth == 1 && len(whiteList) == 0 && len(blackList) > 0 {
			clientIP := remoteIP(c)
			if inList(clientIP, blackList) {
				middleware.ResponseError(c, 3002, fmt.Errorf("%s in black ip list", clientIP))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// remoteIP 取TCP连接的对端IP
// 不用 c.ClientIP()：gin 默认信任任何来源的 X-Forwarded-For，客户端伪造请求头即可绕过黑白名单和IP限流
func remoteIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

func inList(ip string, list []string) bool {
	for _, item := range list {
		if item == ip {
			return true
		}
	}
	return false
}
