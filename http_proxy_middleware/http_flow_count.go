package http_proxy_middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPFlowCountMiddleware 统计网关总流量和服务流量，供大盘和服务统计接口展示
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
		c.Next()
	}
}
