package dao

import (
	"testing"
)

func newLBTestService(name, ipList string, connectTimeout int) *ServiceDetail {
	return &ServiceDetail{
		Info:          &ServiceInfo{ServiceName: name},
		HTTPRule:      &ServiceHTTPRule{},
		AccessControl: &ServiceAccessControl{},
		LoadBalance:   &ServiceLoadBalance{IpList: ipList, UpstreamConnectTimeout: connectTimeout},
	}
}

func TestLoadBalancerRebuildOnConfigChange(t *testing.T) {
	m := &LoadBalancerManager{lbMap: map[string]*lbItem{}}

	lb1, err := m.GetLoadBalancer(newLBTestService("svc", "127.0.0.1:2003", 0))
	if err != nil {
		t.Fatal(err)
	}
	lb2, _ := m.GetLoadBalancer(newLBTestService("svc", "127.0.0.1:2003", 0))
	if lb1 != lb2 {
		t.Error("same config should reuse load balancer")
	}

	// 模拟热加载后 ip_list 变化
	lb3, _ := m.GetLoadBalancer(newLBTestService("svc", "127.0.0.1:2004", 0))
	if lb3 == lb1 {
		t.Fatal("ip_list changed, load balancer should be rebuilt")
	}
	addr, _ := lb3.Get("")
	if addr != "http://127.0.0.1:2004" {
		t.Errorf("addr=%s, want http://127.0.0.1:2004", addr)
	}

	if _, err := m.GetLoadBalancer(newLBTestService("svc", "", 0)); err == nil {
		t.Error("empty ip_list should return error")
	}
}

func TestTransportRebuildOnConfigChange(t *testing.T) {
	m := &TransportManager{transportMap: map[string]*transportItem{}}

	t1 := m.GetTransport(newLBTestService("svc", "127.0.0.1:2003", 5))
	t2 := m.GetTransport(newLBTestService("svc", "127.0.0.1:2004", 5))
	if t1 != t2 {
		t.Error("ip_list change alone should reuse transport")
	}
	t3 := m.GetTransport(newLBTestService("svc", "127.0.0.1:2004", 8))
	if t3 == t1 {
		t.Error("upstream timeout changed, transport should be rebuilt")
	}
}
