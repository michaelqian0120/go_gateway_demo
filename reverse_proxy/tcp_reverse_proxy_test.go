package reverse_proxy

import (
	"net"
	"testing"
	"time"

	"go_gateway_demo/reverse_proxy/load_balance"
)

// proxyPair 起一个下游和一个只转发单条连接的代理，返回客户端连接
func proxyPair(t *testing.T, idle time.Duration, upstreamHandler func(net.Conn)) (net.Conn, <-chan error) {
	up, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { up.Close() })
	go func() {
		conn, err := up.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		upstreamHandler(conn)
	}()

	lb := load_balance.NewRandomBalance()
	lb.Add(up.Addr().String())
	proxy := &TCPReverseProxy{LB: lb, DialTimeout: time.Second, IdleTimeout: idle}

	front, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { front.Close() })
	done := make(chan error, 1)
	go func() {
		conn, err := front.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		done <- proxy.ServeTCP(conn)
	}()

	client, err := net.Dial("tcp", front.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client, done
}

func TestTCPReverseProxyIdleTimeout(t *testing.T) {
	_, done := proxyPair(t, 200*time.Millisecond, func(conn net.Conn) {
		conn.Read(make([]byte, 1)) // 一直不发数据
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("idle connection should be closed after idle timeout")
	}
}

// 客户端只收不发时，客户端方向的读超时不能断开仍在推送数据的连接
func TestTCPReverseProxyOneWayActiveNotTimeout(t *testing.T) {
	const pushes = 8
	client, _ := proxyPair(t, 200*time.Millisecond, func(conn net.Conn) {
		for i := 0; i < pushes; i++ {
			if _, err := conn.Write([]byte{'x'}); err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
	client.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1)
	for i := 0; i < pushes; i++ {
		if _, err := client.Read(buf); err != nil {
			t.Fatalf("read #%d err:%v, connection closed while upstream still active", i, err)
		}
	}
}
