package reverse_proxy

import (
	"io"
	"net"
	"sync/atomic"
	"time"

	"go_gateway_demo/reverse_proxy/load_balance"
)

// TCPReverseProxy 从负载均衡器选下游节点建立连接，在客户端连接和下游连接之间双向转发字节
type TCPReverseProxy struct {
	LB          load_balance.LoadBalance
	DialTimeout time.Duration
	// IdleTimeout 两个方向都没有数据的最长时间，超过后断开连接；0 表示不限制
	IdleTimeout time.Duration
}

// ServeTCP 阻塞直到两个方向的转发都结束，返回第一个非EOF错误
func (p *TCPReverseProxy) ServeTCP(src net.Conn) error {
	addr, err := p.LB.Get(src.RemoteAddr().String())
	if err != nil {
		return err
	}
	dst, err := net.DialTimeout("tcp", addr, p.DialTimeout)
	if err != nil {
		return err
	}
	defer dst.Close()

	lastActive := time.Now().UnixNano()
	errc := make(chan error, 2)
	go p.pipe(dst, src, &lastActive, errc)
	go p.pipe(src, dst, &lastActive, errc)

	firstErr := <-errc
	if firstErr != nil {
		// 一个方向出错时关闭两端，让另一个方向的读写立即返回
		src.Close()
		dst.Close()
	}
	<-errc
	return firstErr
}

// pipe 把 src 读到的数据写入 dst；src 读到EOF时对 dst 半关闭写端，另一个方向继续转发，
// 避免客户端发完请求先关写端时下游的响应被截断
func (p *TCPReverseProxy) pipe(dst, src net.Conn, lastActive *int64, errc chan<- error) {
	buf := make([]byte, 32*1024)
	var err error
	for {
		if p.IdleTimeout > 0 {
			src.SetReadDeadline(time.Now().Add(p.IdleTimeout))
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			atomic.StoreInt64(lastActive, time.Now().UnixNano())
			if p.IdleTimeout > 0 {
				dst.SetWriteDeadline(time.Now().Add(p.IdleTimeout))
			}
			if _, werr := dst.Write(buf[:n]); werr != nil {
				err = werr
				break
			}
		}
		if rerr != nil {
			// 本方向超时但另一个方向最近有数据，连接仍算活跃
			if ne, ok := rerr.(net.Error); ok && ne.Timeout() && p.IdleTimeout > 0 &&
				time.Since(time.Unix(0, atomic.LoadInt64(lastActive))) < p.IdleTimeout {
				continue
			}
			if rerr != io.EOF {
				err = rerr
			}
			break
		}
	}
	if err == nil {
		closeWrite(dst)
	}
	errc <- err
}

func closeWrite(conn net.Conn) {
	if cw, ok := conn.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
		return
	}
	conn.Close()
}
