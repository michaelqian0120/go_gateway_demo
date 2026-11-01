package http_proxy_middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPJwtWhiteListMiddleware 租户IP白名单：租户配置了 white_ips 时，只允许名单内的IP使用该租户的token
// 只对租户认证中间件识别出的租户（上下文 "app"）生效，需要排在租户认证之后、租户限流之前，
// 这样被拒绝的IP不会占用租户的 qps/qpd 额度
func HTTPJwtWhiteListMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		appInterface, ok := c.Get("app")
		if !ok {
			c.Next()
			return
		}
		app, ok := appInterface.(*dao.AppInfo)
		if !ok || app.AppID == "" {
			c.Next()
			return
		}

		whiteIps := splitRules(app.WhiteIps)
		if len(whiteIps) > 0 {
			clientIP := remoteIP(c)
			if !inList(clientIP, whiteIps) {
				middleware.ResponseError(c, 3003, fmt.Errorf("%s not in app %s white ip list", clientIP, app.AppID))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
