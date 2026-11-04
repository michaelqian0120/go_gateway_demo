package tcp_proxy_router

import (
	"context"
	"log"
	"time"

	"go_gateway_demo/dao"
	"go_gateway_demo/tcp_proxy_middleware"
	"golang_common/lib"
)

var TCPServerHandler *TCPServerManager

// DefaultHandlers TCP代理的中间件链，与HTTP代理顺序一致：先计数，再限流、黑白名单，最后转发
func DefaultHandlers() []tcp_proxy_middleware.TCPHandlerFunc {
	return []tcp_proxy_middleware.TCPHandlerFunc{
		tcp_proxy_middleware.TCPFlowCountMiddleware(),
		tcp_proxy_middleware.TCPFlowLimitMiddleware(),
		tcp_proxy_middleware.TCPWhiteListMiddleware(),
		tcp_proxy_middleware.TCPBlackListMiddleware(),
		tcp_proxy_middleware.TCPReverseProxyMiddleware(),
	}
}

// TCPServerRun 按已加载的TCP服务开始监听，并在服务配置热加载后同步端口
func TCPServerRun() {
	host := lib.GetStringConf("proxy.tcp.host")
	TCPServerHandler = NewTCPServerManager(host, dao.ServiceManagerHandler.GetService, DefaultHandlers()...)
	TCPServerHandler.Sync(dao.ServiceManagerHandler.GetTCPServiceList())
	dao.ServiceManagerHandler.OnReload(func() {
		TCPServerHandler.Sync(dao.ServiceManagerHandler.GetTCPServiceList())
	})
}

func TCPServerStop() {
	if TCPServerHandler == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	TCPServerHandler.StopAll(ctx)
	log.Printf(" [INFO] TcpProxyStop stopped\n")
}
