package tcp_proxy_router

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"go_gateway_demo/dao"
	"go_gateway_demo/tcp_proxy_middleware"
)

type tcpServer struct {
	port        int
	serviceName string
	ln          net.Listener
}

// TCPServerManager 每个TCP服务独占一个端口，按服务列表增删端口监听
// 连接建立时按服务名取最新配置，访问控制、限流、下游列表的修改无需重新监听
type TCPServerManager struct {
	host     string
	handlers []tcp_proxy_middleware.TCPHandlerFunc
	lookup   func(serviceName string) (*dao.ServiceDetail, bool)

	locker  sync.Mutex
	servers map[int]*tcpServer
	conns   map[net.Conn]struct{}
	wg      sync.WaitGroup
	stopped bool
}

func NewTCPServerManager(host string, lookup func(string) (*dao.ServiceDetail, bool), handlers ...tcp_proxy_middleware.TCPHandlerFunc) *TCPServerManager {
	return &TCPServerManager{
		host:     host,
		handlers: handlers,
		lookup:   lookup,
		servers:  map[int]*tcpServer{},
		conns:    map[net.Conn]struct{}{},
	}
}

// Sync 按服务列表调整监听：新端口开始监听，服务删除或端口变化时关闭旧监听，已建立的连接不受影响
// 监听失败（如端口被占用）只记录日志，下次 Sync 时重试
func (m *TCPServerManager) Sync(services []*dao.ServiceDetail) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if m.stopped {
		return
	}

	desired := map[int]string{}
	for _, s := range services {
		if s.TCPRule == nil || s.TCPRule.Port <= 0 {
			continue
		}
		if name, ok := desired[s.TCPRule.Port]; ok {
			log.Printf(" [ERROR] tcp port %d used by both %s and %s, keep %s\n", s.TCPRule.Port, name, s.Info.ServiceName, name)
			continue
		}
		desired[s.TCPRule.Port] = s.Info.ServiceName
	}

	for port, srv := range m.servers {
		if name, ok := desired[port]; !ok || name != srv.serviceName {
			srv.ln.Close()
			delete(m.servers, port)
			log.Printf(" [INFO] TcpProxyStop service:%s port:%d\n", srv.serviceName, port)
		}
	}

	for port, name := range desired {
		if _, ok := m.servers[port]; ok {
			continue
		}
		addr := net.JoinHostPort(m.host, strconv.Itoa(port))
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf(" [ERROR] TcpProxyRun service:%s addr:%s err:%v\n", name, addr, err)
			continue
		}
		srv := &tcpServer{port: port, serviceName: name, ln: ln}
		m.servers[port] = srv
		log.Printf(" [INFO] TcpProxyRun service:%s addr:%s\n", name, addr)
		go m.serve(srv)
	}
}

// Addr 返回端口对应的实际监听地址，未监听时返回空
func (m *TCPServerManager) Addr(port int) string {
	m.locker.Lock()
	defer m.locker.Unlock()
	if srv, ok := m.servers[port]; ok {
		return srv.ln.Addr().String()
	}
	return ""
}

func (m *TCPServerManager) serve(srv *tcpServer) {
	var delay time.Duration
	for {
		conn, err := srv.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// 临时错误（如文件描述符耗尽）退避后重试
			if delay == 0 {
				delay = 5 * time.Millisecond
			} else if delay *= 2; delay > time.Second {
				delay = time.Second
			}
			log.Printf(" [ERROR] tcp accept port:%d err:%v, retrying in %v\n", srv.port, err, delay)
			time.Sleep(delay)
			continue
		}
		delay = 0
		if !m.track(conn) {
			conn.Close()
			continue
		}
		go m.handleConn(srv.serviceName, conn)
	}
}

func (m *TCPServerManager) handleConn(serviceName string, conn net.Conn) {
	defer m.wg.Done()
	defer m.untrack(conn)
	defer conn.Close()
	defer func() {
		if err := recover(); err != nil {
			log.Printf(" [ERROR] tcp proxy service:%s client:%s panic:%v\n", serviceName, conn.RemoteAddr(), err)
		}
	}()

	service, ok := m.lookup(serviceName)
	if !ok || service.TCPRule == nil {
		// 服务已删除，监听还没来得及关闭
		return
	}
	tcp_proxy_middleware.NewTCPContext(conn, service, m.handlers).Next()
}

// track 登记连接以便停止时等待或强制关闭；已停止时返回 false
func (m *TCPServerManager) track(conn net.Conn) bool {
	m.locker.Lock()
	defer m.locker.Unlock()
	if m.stopped {
		return false
	}
	m.conns[conn] = struct{}{}
	m.wg.Add(1)
	return true
}

func (m *TCPServerManager) untrack(conn net.Conn) {
	m.locker.Lock()
	delete(m.conns, conn)
	m.locker.Unlock()
}

// StopAll 关闭所有监听，等待已建立的连接结束；ctx 到期后强制关闭剩余连接
func (m *TCPServerManager) StopAll(ctx context.Context) {
	m.locker.Lock()
	m.stopped = true
	for port, srv := range m.servers {
		srv.ln.Close()
		delete(m.servers, port)
	}
	m.locker.Unlock()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-ctx.Done():
	}

	m.locker.Lock()
	n := len(m.conns)
	for conn := range m.conns {
		conn.Close()
	}
	m.locker.Unlock()
	log.Printf(" [WARN] TcpProxyStop timeout, force closed %d connections\n", n)
	<-done
}
