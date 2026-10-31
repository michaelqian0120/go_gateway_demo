package http_proxy_middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
	"go_gateway_demo/reverse_proxy"
)

// HTTPReverseProxyMiddleware 取出负载均衡器和连接池，转发到下游
func HTTPReverseProxyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		lb, err := dao.LoadBalancerHandler.GetLoadBalancer(service)
		if err != nil {
			middleware.ResponseError(c, 2002, err)
			c.Abort()
			return
		}
		trans := dao.TransportHandler.GetTransport(service)

		proxy := reverse_proxy.NewLoadBalanceReverseProxy(c, lb, trans)
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
