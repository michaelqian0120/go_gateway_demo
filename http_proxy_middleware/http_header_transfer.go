package http_proxy_middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
	"go_gateway_demo/public"
)

// HTTPHeaderTransferMiddleware 按 header_transfor 规则修改发往下游的请求头
// 规则格式：add|edit name value 或 del name，多条用逗号或换行分隔
func HTTPHeaderTransferMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		for _, item := range public.SplitRules(service.HTTPRule.HeaderTransfor) {
			items := strings.Fields(item)
			if len(items) < 2 {
				continue
			}
			switch items[0] {
			case "add", "edit":
				if len(items) == 3 {
					c.Request.Header.Set(items[1], items[2])
				}
			case "del":
				c.Request.Header.Del(items[1])
			}
		}
		c.Next()
	}
}
