package dao

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang_common/lib"
)

// 小时统计数据保留2天，大盘只需要今天和昨天
const flowCountExpireSeconds = 86400 * 2

// RedisFlowCounter 请求计数先累加在内存里，每隔 interval 批量写入 redis hash
// 写入格式与 hourCountList 读取的格式一致：key 下的 field 为 YYYYMMDDHH，值为该小时请求量
type RedisFlowCounter struct {
	Key      string
	Interval time.Duration
	tick     int64
}

func NewRedisFlowCounter(key string, interval time.Duration) *RedisFlowCounter {
	counter := &RedisFlowCounter{Key: key, Interval: interval}
	go counter.loop()
	return counter
}

func (c *RedisFlowCounter) Increase() {
	atomic.AddInt64(&c.tick, 1)
}

func (c *RedisFlowCounter) loop() {
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()
	for range ticker.C {
		c.flush()
	}
}

func (c *RedisFlowCounter) flush() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf(" [ERROR] flow counter flush key:%s panic:%v\n", c.Key, err)
		}
	}()

	n := atomic.SwapInt64(&c.tick, 0)
	if n == 0 {
		return
	}
	conn, err := lib.RedisConnFactory("default")
	if err != nil {
		// 写失败时把计数加回去，下次再试
		atomic.AddInt64(&c.tick, n)
		log.Printf(" [ERROR] flow counter redis conn key:%s err:%v\n", c.Key, err)
		return
	}
	defer conn.Close()

	field := time.Now().Format("2006010215")
	conn.Send("MULTI")
	conn.Send("HINCRBY", c.Key, field, n)
	conn.Send("EXPIRE", c.Key, flowCountExpireSeconds)
	if _, err := conn.Do("EXEC"); err != nil {
		atomic.AddInt64(&c.tick, n)
		log.Printf(" [ERROR] flow counter flush key:%s err:%v\n", c.Key, err)
	}
}

// FlowCounterManager 按 redis key 缓存计数器，每个 key 只起一个刷新协程
type FlowCounterManager struct {
	counterMap map[string]*RedisFlowCounter
	locker     sync.Mutex
}

var FlowCounterHandler = &FlowCounterManager{counterMap: map[string]*RedisFlowCounter{}}

func (m *FlowCounterManager) getCounter(key string) *RedisFlowCounter {
	m.locker.Lock()
	defer m.locker.Unlock()
	if counter, ok := m.counterMap[key]; ok {
		return counter
	}
	counter := NewRedisFlowCounter(key, time.Second)
	m.counterMap[key] = counter
	return counter
}

func (m *FlowCounterManager) TotalCounter() *RedisFlowCounter {
	return m.getCounter(TotalFlowCountKey)
}

func (m *FlowCounterManager) ServiceCounter(serviceName string) *RedisFlowCounter {
	return m.getCounter(fmt.Sprintf("%s%s", FlowCountKeyPrefix, serviceName))
}

func (m *FlowCounterManager) AppCounter(appID string) *RedisFlowCounter {
	return m.getCounter(fmt.Sprintf("%s%s", AppFlowCountKeyPrefix, appID))
}
