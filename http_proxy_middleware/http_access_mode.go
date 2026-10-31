package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPAccessModeMiddleware 匹配接入方式，把匹配到的服务放进上下文
func HTTPAccessModeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		service, err := dao.ServiceManagerHandler.HTTPAccessMode(c)
		if err != nil {
			middleware.ResponseError(c, 1001, err)
			c.Abort()
			return
		}
		c.Set("service", service)
		c.Next()
	}
}
