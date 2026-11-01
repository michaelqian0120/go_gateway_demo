package http_proxy_middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPJwtFlowLimitMiddleware 租户限流：qps 为每秒请求数，qpd 为每天请求数，0表示不限
// 只对租户认证中间件识别出的租户（上下文 "app"）生效，需要排在租户认证之后、流量统计之前，
// 这样被租户限流拒绝的请求不计入租户当天请求量
// qpd 依据每秒从 redis 读回的当天总量判断，多实例部署时可能超出约1秒的流量
func HTTPJwtFlowLimitMiddleware() gin.HandlerFunc {
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

		if qpd := app.Qpd; qpd > 0 {
			if n := dao.FlowCounterHandler.AppCounter(app.AppID).DayCount(); n >= qpd {
				middleware.ResponseError(c, 5004, fmt.Errorf("app %s day flow limit %d, today %d", app.AppID, qpd, n))
				c.Abort()
				return
			}
		}

		if qps := app.Qps; qps > 0 {
			if !dao.FlowLimiterHandler.GetLimiter("app_"+app.AppID, int(qps)).Allow() {
				middleware.ResponseError(c, 5003, fmt.Errorf("app %s flow limit %d/s", app.AppID, qps))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
