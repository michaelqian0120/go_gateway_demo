package dao

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go_gateway_demo/reverse_proxy/load_balance"
)

// LoadBalancerManager 按服务名缓存负载均衡器，避免每次请求都重新创建
type LoadBalancerManager struct {
	lbMap  map[string]load_balance.LoadBalance
	locker sync.RWMutex
}

var LoadBalancerHandler = &LoadBalancerManager{lbMap: map[string]load_balance.LoadBalance{}}

func (m *LoadBalancerManager) GetLoadBalancer(service *ServiceDetail) (load_balance.LoadBalance, error) {
	name := service.Info.ServiceName
	m.locker.RLock()
	lb, ok := m.lbMap[name]
	m.locker.RUnlock()
	if ok {
		return lb, nil
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	if lb, ok := m.lbMap[name]; ok {
		return lb, nil
	}

	ipList := splitList(service.LoadBalance.IpList)
	if len(ipList) == 0 {
		return nil, errors.New("service " + name + " has no ip_list")
	}
	// need_https 指网关对外是否支持https，与下游协议无关；下游默认http，可在ip_list里写明协议
	addrs := make([]string, 0, len(ipList))
	for _, ip := range ipList {
		if !strings.Contains(ip, "://") {
			ip = "http://" + ip
		}
		addrs = append(addrs, ip)
	}

	// 第1步只实现random，round_type暂不区分
	rb := load_balance.NewRandomBalance()
	if err := rb.Add(addrs...); err != nil {
		return nil, err
	}
	m.lbMap[name] = rb
	return rb, nil
}

// TransportManager 按服务名缓存连接池
type TransportManager struct {
	transportMap map[string]*http.Transport
	locker       sync.RWMutex
}

var TransportHandler = &TransportManager{transportMap: map[string]*http.Transport{}}

func (m *TransportManager) GetTransport(service *ServiceDetail) *http.Transport {
	name := service.Info.ServiceName
	m.locker.RLock()
	trans, ok := m.transportMap[name]
	m.locker.RUnlock()
	if ok {
		return trans
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	if trans, ok := m.transportMap[name]; ok {
		return trans
	}

	lb := service.LoadBalance
	trans = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   secondsOrDefault(lb.UpstreamConnectTimeout, 30),
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          intOrDefault(lb.UpstreamMaxIdle, 100),
		IdleConnTimeout:       secondsOrDefault(lb.UpstreamIdleTimeout, 90),
		ResponseHeaderTimeout: secondsOrDefault(lb.UpstreamHeaderTimeout, 30),
		TLSHandshakeTimeout:   10 * time.Second,
	}
	m.transportMap[name] = trans
	return trans
}

func splitList(s string) []string {
	out := []string{}
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func secondsOrDefault(v, def int) time.Duration {
	if v <= 0 {
		v = def
	}
	return time.Duration(v) * time.Second
}

func intOrDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
