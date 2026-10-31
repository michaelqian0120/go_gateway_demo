package dao

import (
	"testing"
)

// 计数器写入的 key 必须和统计接口读取的 key 一致，否则统计接口读不到数据
func TestFlowCounterKeys(t *testing.T) {
	m := &FlowCounterManager{counterMap: map[string]*RedisFlowCounter{}}
	cases := []struct {
		name    string
		counter *RedisFlowCounter
		want    string
	}{
		{"total", m.TotalCounter(), TotalFlowCountKey},
		{"service", m.ServiceCounter("svc_a"), FlowCountKeyPrefix + (&ServiceFlowCount{ServiceName: "svc_a"}).ServiceName},
		{"app", m.AppCounter("app_a"), AppFlowCountKeyPrefix + (&AppFlowCount{AppID: "app_a"}).AppID},
	}
	for _, tc := range cases {
		if tc.counter.Key != tc.want {
			t.Errorf("%s key=%s, want %s", tc.name, tc.counter.Key, tc.want)
		}
	}
	if m.AppCounter("app_a") != m.AppCounter("app_a") {
		t.Error("same app should reuse counter")
	}
	if m.AppCounter("app_a") == m.AppCounter("app_b") {
		t.Error("different apps should use different counters")
	}
}
