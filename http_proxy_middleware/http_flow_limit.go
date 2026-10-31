package http_proxy_middleware

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

// HTTPFlowLimitMiddleware 服务级限流和客户端IP级限流，单位为每秒请求数，0表示不限
func HTTPFlowLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)
		serviceName := service.Info.ServiceName

		if qps := service.AccessControl.ServiceFlowLimit; qps > 0 {
			if !dao.FlowLimiterHandler.GetLimiter("service_"+serviceName, qps).Allow() {
				middleware.ResponseError(c, 5001, fmt.Errorf("service flow limit %d/s", qps))
				c.Abort()
				return
			}
		}

		if qps := service.AccessControl.ClientIPFlowLimit; qps > 0 {
			clientIP := remoteIP(c)
			if !dao.FlowLimiterHandler.GetLimiter("clientip_"+serviceName+"_"+clientIP, qps).Allow() {
				middleware.ResponseError(c, 5002, fmt.Errorf("%s flow limit %d/s", clientIP, qps))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
