package http_proxy_middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPFlowCountMiddleware 统计网关总流量、服务流量和租户流量，供大盘、服务统计和租户统计接口展示
// 租户流量只在上下文中有 "app"（*dao.AppInfo）时统计，设置 "app" 的租户认证中间件需要排在本中间件之前
func HTTPFlowCountMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		dao.FlowCounterHandler.TotalCounter().Increase()
		dao.FlowCounterHandler.ServiceCounter(service.Info.ServiceName).Increase()
		if appInterface, ok := c.Get("app"); ok {
			if app, ok := appInterface.(*dao.AppInfo); ok && app.AppID != "" {
				dao.FlowCounterHandler.AppCounter(app.AppID).Increase()
			}
		}
		c.Next()
	}
}
