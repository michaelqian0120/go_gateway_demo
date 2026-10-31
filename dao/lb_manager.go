package dao

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go_gateway_demo/reverse_proxy/load_balance"
)

type lbItem struct {
	lb        load_balance.LoadBalance
	signature string
}

// LoadBalancerManager 按服务名缓存负载均衡器，避免每次请求都重新创建
// 服务热加载后 ip_list 或 round_type 变化时自动重建
type LoadBalancerManager struct {
	lbMap  map[string]*lbItem
	locker sync.RWMutex
}

var LoadBalancerHandler = &LoadBalancerManager{lbMap: map[string]*lbItem{}}

func (m *LoadBalancerManager) GetLoadBalancer(service *ServiceDetail) (load_balance.LoadBalance, error) {
	name := service.Info.ServiceName
	signature := fmt.Sprintf("%d|%s", service.LoadBalance.RoundType, service.LoadBalance.IpList)
	m.locker.RLock()
	item, ok := m.lbMap[name]
	m.locker.RUnlock()
	if ok && item.signature == signature {
		return item.lb, nil
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	if item, ok := m.lbMap[name]; ok && item.signature == signature {
		return item.lb, nil
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

	// 目前只实现random，round_type暂不区分
	rb := load_balance.NewRandomBalance()
	if err := rb.Add(addrs...); err != nil {
		return nil, err
	}
	m.lbMap[name] = &lbItem{lb: rb, signature: signature}
	return rb, nil
}

type transportItem struct {
	trans     *http.Transport
	signature string
}

// TransportManager 按服务名缓存连接池，上游参数变化时重建并关闭旧连接池的空闲连接
type TransportManager struct {
	transportMap map[string]*transportItem
	locker       sync.RWMutex
}

var TransportHandler = &TransportManager{transportMap: map[string]*transportItem{}}

func (m *TransportManager) GetTransport(service *ServiceDetail) *http.Transport {
	name := service.Info.ServiceName
	lb := service.LoadBalance
	signature := fmt.Sprintf("%d|%d|%d|%d", lb.UpstreamConnectTimeout, lb.UpstreamHeaderTimeout, lb.UpstreamIdleTimeout, lb.UpstreamMaxIdle)
	m.locker.RLock()
	item, ok := m.transportMap[name]
	m.locker.RUnlock()
	if ok && item.signature == signature {
		return item.trans
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	if item, ok := m.transportMap[name]; ok {
		if item.signature == signature {
			return item.trans
		}
		// 正在使用的连接不受影响，只关闭空闲连接
		item.trans.CloseIdleConnections()
	}

	trans := &http.Transport{
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
	m.transportMap[name] = &transportItem{trans: trans, signature: signature}
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
