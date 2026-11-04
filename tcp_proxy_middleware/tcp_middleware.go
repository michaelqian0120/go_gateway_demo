package tcp_proxy_middleware

import (
	"fmt"
	"log"
	"time"

	"go_gateway_demo/dao"
	"go_gateway_demo/public"
	"go_gateway_demo/reverse_proxy"
)

// TCPFlowCountMiddleware 每条连接计一次，统计网关总流量和服务流量，与HTTP一样被拒绝的连接也计入
func TCPFlowCountMiddleware() TCPHandlerFunc {
	return func(c *TCPContext) {
		dao.FlowCounterHandler.TotalCounter().Increase()
		dao.FlowCounterHandler.ServiceCounter(c.Service.Info.ServiceName).Increase()
		c.Next()
	}
}

// TCPFlowLimitMiddleware 服务级和客户端IP级限流，单位为每秒新建连接数，0表示不限
func TCPFlowLimitMiddleware() TCPHandlerFunc {
	return func(c *TCPContext) {
		serviceName := c.Service.Info.ServiceName
		if qps := c.Service.AccessControl.ServiceFlowLimit; qps > 0 {
			if !dao.FlowLimiterHandler.GetLimiter("tcp_service_"+serviceName, qps).Allow() {
				c.Reject(fmt.Sprintf("service flow limit %d/s", qps))
				return
			}
		}
		if qps := c.Service.AccessControl.ClientIPFlowLimit; qps > 0 {
			clientIP := c.ClientIP()
			if !dao.FlowLimiterHandler.GetLimiter("tcp_clientip_"+serviceName+"_"+clientIP, qps).Allow() {
				c.Reject(fmt.Sprintf("%s flow limit %d/s", clientIP, qps))
				return
			}
		}
		c.Next()
	}
}

// TCPWhiteListMiddleware 开启权限且配置了白名单时，只允许白名单内的IP连接
func TCPWhiteListMiddleware() TCPHandlerFunc {
	return func(c *TCPContext) {
		ac := c.Service.AccessControl
		whiteList := public.SplitRules(ac.WhiteList)
		if ac.OpenAuth == 1 && len(whiteList) > 0 {
			if clientIP := c.ClientIP(); !public.InList(clientIP, whiteList) {
				c.Reject(fmt.Sprintf("%s not in white ip list", clientIP))
				return
			}
		}
		c.Next()
	}
}

// TCPBlackListMiddleware 开启权限且未配置白名单时，拒绝黑名单内的IP；配置了白名单则以白名单为准
func TCPBlackListMiddleware() TCPHandlerFunc {
	return func(c *TCPContext) {
		ac := c.Service.AccessControl
		whiteList := public.SplitRules(ac.WhiteList)
		blackList := public.SplitRules(ac.BlackList)
		if ac.OpenAuth == 1 && len(whiteList) == 0 && len(blackList) > 0 {
			if clientIP := c.ClientIP(); public.InList(clientIP, blackList) {
				c.Reject(fmt.Sprintf("%s in black ip list", clientIP))
				return
			}
		}
		c.Next()
	}
}

// TCPReverseProxyMiddleware 链路最后一环，阻塞到连接转发结束
func TCPReverseProxyMiddleware() TCPHandlerFunc {
	return func(c *TCPContext) {
		lb, err := dao.LoadBalancerHandler.GetLoadBalancer(c.Service)
		if err != nil {
			c.Reject(err.Error())
			return
		}
		lbConf := c.Service.LoadBalance
		proxy := &reverse_proxy.TCPReverseProxy{
			LB:          lb,
			DialTimeout: secondsOrDefault(lbConf.UpstreamConnectTimeout, 30),
			IdleTimeout: time.Duration(lbConf.UpstreamIdleTimeout) * time.Second,
		}
		if err := proxy.ServeTCP(c.Conn); err != nil {
			log.Printf(" [WARN] tcp proxy service:%s client:%s err:%v\n", c.Service.Info.ServiceName, c.Conn.RemoteAddr(), err)
		}
		c.Abort()
	}
}

func secondsOrDefault(v, def int) time.Duration {
	if v <= 0 {
		v = def
	}
	return time.Duration(v) * time.Second
}
