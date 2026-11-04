package tcp_proxy_router

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"go_gateway_demo/dao"
	"go_gateway_demo/tcp_proxy_middleware"
)

// 测试不挂流量计数中间件，避免依赖 redis
func testHandlers() []tcp_proxy_middleware.TCPHandlerFunc {
	return []tcp_proxy_middleware.TCPHandlerFunc{
		tcp_proxy_middleware.TCPFlowLimitMiddleware(),
		tcp_proxy_middleware.TCPWhiteListMiddleware(),
		tcp_proxy_middleware.TCPBlackListMiddleware(),
		tcp_proxy_middleware.TCPReverseProxyMiddleware(),
	}
}

type serviceStore struct {
	sync.Mutex
	m map[string]*dao.ServiceDetail
}

func (s *serviceStore) set(list ...*dao.ServiceDetail) {
	s.Lock()
	defer s.Unlock()
	s.m = map[string]*dao.ServiceDetail{}
	for _, d := range list {
		s.m[d.Info.ServiceName] = d
	}
}

func (s *serviceStore) get(name string) (*dao.ServiceDetail, bool) {
	s.Lock()
	defer s.Unlock()
	d, ok := s.m[name]
	return d, ok
}

func newTCPService(name string, port int, upstream string) *dao.ServiceDetail {
	return &dao.ServiceDetail{
		Info:          &dao.ServiceInfo{ServiceName: name, LoadType: dao.LoadTypeTCP},
		TCPRule:       &dao.ServiceTCPRule{Port: port},
		AccessControl: &dao.ServiceAccessControl{},
		LoadBalance:   &dao.ServiceLoadBalance{IpList: upstream, UpstreamConnectTimeout: 2},
	}
}

func freePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// startUpstream 起一个下游服务：读完客户端发来的全部数据（直到EOF）后回写 "got:" + 数据
// 能同时验证数据往返和半关闭传递
func startUpstream(t *testing.T) (string, *int64Counter) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	accepted := &int64Counter{}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.inc()
			go func(conn net.Conn) {
				defer conn.Close()
				data, _ := io.ReadAll(conn)
				conn.Write(append([]byte("got:"), data...))
			}(conn)
		}
	}()
	return ln.Addr().String(), accepted
}

type int64Counter struct {
	sync.Mutex
	n int
}

func (c *int64Counter) inc() { c.Lock(); c.n++; c.Unlock() }
func (c *int64Counter) get() int {
	c.Lock()
	defer c.Unlock()
	return c.n
}

// roundTrip 发送 msg 后半关闭写端，读回全部响应
func roundTrip(addr, msg string) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte(msg)); err != nil {
		return "", err
	}
	conn.(*net.TCPConn).CloseWrite()
	data, err := io.ReadAll(conn)
	return string(data), err
}

func newTestManager(t *testing.T, store *serviceStore) *TCPServerManager {
	m := NewTCPServerManager("127.0.0.1", store.get, testHandlers()...)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		m.StopAll(ctx)
	})
	return m
}

func TestTCPProxyRoundTripWithHalfClose(t *testing.T) {
	upstream, _ := startUpstream(t)
	port := freePort(t)
	svc := newTCPService("tcp_roundtrip", port, upstream)
	store := &serviceStore{}
	store.set(svc)
	m := newTestManager(t, store)
	m.Sync([]*dao.ServiceDetail{svc})

	got, err := roundTrip(m.Addr(port), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if got != "got:hello" {
		t.Errorf("got %q, want %q", got, "got:hello")
	}
}

func TestTCPProxyBlackList(t *testing.T) {
	upstream, accepted := startUpstream(t)
	port := freePort(t)
	svc := newTCPService("tcp_blacklist", port, upstream)
	svc.AccessControl.OpenAuth = 1
	svc.AccessControl.BlackList = "127.0.0.1"
	store := &serviceStore{}
	store.set(svc)
	m := newTestManager(t, store)
	m.Sync([]*dao.ServiceDetail{svc})

	got, _ := roundTrip(m.Addr(port), "hello")
	if got != "" {
		t.Errorf("black listed client got %q, want connection closed", got)
	}
	if n := accepted.get(); n != 0 {
		t.Errorf("upstream accepted %d connections, want 0", n)
	}

	// 白名单优先：配置白名单后黑名单不生效
	svc2 := newTCPService("tcp_blacklist", port, upstream)
	svc2.AccessControl.OpenAuth = 1
	svc2.AccessControl.BlackList = "127.0.0.1"
	svc2.AccessControl.WhiteList = "127.0.0.1"
	store.set(svc2)
	if got, _ := roundTrip(m.Addr(port), "hi"); got != "got:hi" {
		t.Errorf("white listed client got %q, want got:hi", got)
	}
}

func TestTCPProxyServiceFlowLimit(t *testing.T) {
	upstream, _ := startUpstream(t)
	port := freePort(t)
	// 限流器按服务名全局缓存，用唯一服务名避免 -count>1 时沿用上一轮的令牌桶
	svc := newTCPService("tcp_flow_limit_"+strconv.FormatInt(time.Now().UnixNano(), 10), port, upstream)
	svc.AccessControl.ServiceFlowLimit = 1
	store := &serviceStore{}
	store.set(svc)
	m := newTestManager(t, store)
	m.Sync([]*dao.ServiceDetail{svc})

	if got, _ := roundTrip(m.Addr(port), "a"); got != "got:a" {
		t.Fatalf("first connection got %q, want got:a", got)
	}
	if got, _ := roundTrip(m.Addr(port), "b"); got != "" {
		t.Errorf("second connection within 1s got %q, want rejected", got)
	}
}

func TestTCPProxyUpstreamUnavailable(t *testing.T) {
	port := freePort(t)
	svc := newTCPService("tcp_upstream_down", port, "127.0.0.1:"+strconv.Itoa(freePort(t)))
	store := &serviceStore{}
	store.set(svc)
	m := newTestManager(t, store)
	m.Sync([]*dao.ServiceDetail{svc})

	if got, _ := roundTrip(m.Addr(port), "x"); got != "" {
		t.Errorf("got %q, want connection closed", got)
	}
}

func TestTCPServerSync(t *testing.T) {
	upstream, _ := startUpstream(t)
	portA, portB := freePort(t), freePort(t)
	store := &serviceStore{}
	m := newTestManager(t, store)

	svc := newTCPService("tcp_sync", portA, upstream)
	store.set(svc)
	m.Sync([]*dao.ServiceDetail{svc})
	addrA := m.Addr(portA)
	if got, _ := roundTrip(addrA, "a"); got != "got:a" {
		t.Fatalf("port A got %q", got)
	}

	// 端口变更：旧端口关闭，新端口开始监听
	svc2 := newTCPService("tcp_sync", portB, upstream)
	store.set(svc2)
	m.Sync([]*dao.ServiceDetail{svc2})
	if m.Addr(portA) != "" {
		t.Error("port A should be closed after port change")
	}
	if _, err := net.DialTimeout("tcp", addrA, 500*time.Millisecond); err == nil {
		t.Error("dial port A should fail after port change")
	}
	if got, _ := roundTrip(m.Addr(portB), "b"); got != "got:b" {
		t.Errorf("port B got %q", got)
	}

	// 服务删除
	store.set()
	m.Sync(nil)
	if m.Addr(portB) != "" {
		t.Error("port B should be closed after service deleted")
	}

	// 两个服务配置了同一端口时只监听一个，不 panic
	s1 := newTCPService("tcp_dup1", portA, upstream)
	s2 := newTCPService("tcp_dup2", portA, upstream)
	store.set(s1, s2)
	m.Sync([]*dao.ServiceDetail{s1, s2})
	if m.Addr(portA) == "" {
		t.Error("duplicated port should still be listened once")
	}
}

func TestTCPServerStopForceClose(t *testing.T) {
	// 下游一直不关闭连接，模拟长连接
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go io.Copy(io.Discard, conn)
		}
	}()

	port := freePort(t)
	svc := newTCPService("tcp_stop", port, ln.Addr().String())
	store := &serviceStore{}
	store.set(svc)
	m := NewTCPServerManager("127.0.0.1", store.get, testHandlers()...)
	m.Sync([]*dao.ServiceDetail{svc})

	conn, err := net.Dial("tcp", m.Addr(port))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write([]byte("keep"))
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		m.StopAll(ctx)
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("StopAll should force close connections after ctx timeout")
	}

	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Error("client connection should be closed after StopAll")
	}
	// 停止后 Sync 不再监听
	m.Sync([]*dao.ServiceDetail{svc})
	if m.Addr(port) != "" {
		t.Error("Sync after StopAll should not listen")
	}
}
