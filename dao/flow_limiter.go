package dao

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type flowLimiterItem struct {
	limiter  *rate.Limiter
	qps      int
	lastSeen time.Time
}

// FlowLimiterManager 按 key 缓存令牌桶限流器
// key 为服务名或服务名+客户端IP；长时间不用的限流器会被定期清理，避免按IP限流时内存无限增长
type FlowLimiterManager struct {
	limiterMap map[string]*flowLimiterItem
	locker     sync.Mutex
}

const flowLimiterIdleTimeout = 10 * time.Minute

var FlowLimiterHandler = NewFlowLimiterManager()

func NewFlowLimiterManager() *FlowLimiterManager {
	m := &FlowLimiterManager{limiterMap: map[string]*flowLimiterItem{}}
	go m.cleanLoop()
	return m
}

// GetLimiter 获取每秒允许 qps 个请求的限流器，突发上限同为 qps
// qps 变化时重新创建，便于后续服务配置热加载
func (m *FlowLimiterManager) GetLimiter(key string, qps int) *rate.Limiter {
	m.locker.Lock()
	defer m.locker.Unlock()
	item, ok := m.limiterMap[key]
	if !ok || item.qps != qps {
		item = &flowLimiterItem{limiter: rate.NewLimiter(rate.Limit(qps), qps), qps: qps}
		m.limiterMap[key] = item
	}
	item.lastSeen = time.Now()
	return item.limiter
}

func (m *FlowLimiterManager) cleanLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.locker.Lock()
		for key, item := range m.limiterMap {
			if time.Since(item.lastSeen) > flowLimiterIdleTimeout {
				delete(m.limiterMap, key)
			}
		}
		m.locker.Unlock()
	}
}
