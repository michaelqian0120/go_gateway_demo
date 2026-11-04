package tcp_proxy_middleware

import (
	"log"
	"math"
	"net"

	"go_gateway_demo/dao"
)

// TCPHandlerFunc TCP连接的中间件，与gin中间件用法一致：调用 c.Next() 进入下一个，调用 c.Abort() 中止
type TCPHandlerFunc func(c *TCPContext)

const abortIndex = math.MaxInt32 / 2

// TCPContext 一条客户端连接的处理上下文，链路结束后由调用方关闭连接
type TCPContext struct {
	Conn    net.Conn
	Service *dao.ServiceDetail

	handlers []TCPHandlerFunc
	index    int
}

func NewTCPContext(conn net.Conn, service *dao.ServiceDetail, handlers []TCPHandlerFunc) *TCPContext {
	return &TCPContext{Conn: conn, Service: service, handlers: handlers, index: -1}
}

func (c *TCPContext) Next() {
	c.index++
	for c.index < len(c.handlers) {
		c.handlers[c.index](c)
		c.index++
	}
}

func (c *TCPContext) Abort() {
	c.index = abortIndex
}

func (c *TCPContext) IsAborted() bool {
	return c.index >= abortIndex
}

// Reject TCP没有统一的响应格式，拒绝时只记录日志并中止，连接由调用方关闭
func (c *TCPContext) Reject(reason string) {
	log.Printf(" [WARN] tcp proxy service:%s client:%s rejected: %s\n", c.Service.Info.ServiceName, c.Conn.RemoteAddr(), reason)
	c.Abort()
}

// ClientIP 取TCP连接的对端IP
func (c *TCPContext) ClientIP() string {
	addr := c.Conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
