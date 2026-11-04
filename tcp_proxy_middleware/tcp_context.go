package tcp_proxy_middleware

import (
	"io"
	"log"
	"math"
	"net"
	"time"

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

// rejectTimeout 回写拒绝原因、等待客户端关闭的最长时间，避免客户端不配合时卡住处理协程
const rejectTimeout = time.Second

// Reject 记录日志，把拒绝原因以一行纯文本回写给客户端后中止，连接由调用方关闭
// TCP没有统一的响应格式，回写的文本只方便 telnet 等工具排查，不是协议的一部分
func (c *TCPContext) Reject(reason string) {
	log.Printf(" [WARN] tcp proxy service:%s client:%s rejected: %s\n", c.Service.Info.ServiceName, c.Conn.RemoteAddr(), reason)
	c.Conn.SetDeadline(time.Now().Add(rejectTimeout))
	if _, err := c.Conn.Write([]byte(reason + "\r\n")); err == nil {
		lingerClose(c.Conn)
	}
	c.Abort()
}

// lingerClose 半关闭写端后丢弃客户端已发来的数据，直到客户端关闭或超时。
// 接收缓冲区还有未读数据时直接 Close 会发 RST，客户端可能来不及读到拒绝原因
func lingerClose(conn net.Conn) {
	cw, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		return
	}
	cw.CloseWrite()
	io.Copy(io.Discard, conn)
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
