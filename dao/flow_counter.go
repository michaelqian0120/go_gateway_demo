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
// trackDay 为 true 时每个周期从 redis 读回当天总量，供租户日请求量（qpd）限流使用，
// 读回的总量包含其他网关实例写入的计数
type RedisFlowCounter struct {
	Key      string
	Interval time.Duration
	tick     int64

	trackDay  bool
	dayLocker sync.Mutex
	day       string // dayCount 对应的日期 YYYYMMDD
	dayCount  int64  // redis 中当天已写入的请求量
	flushing  int64  // 已从 tick 取出、正在写入 redis 的请求量
}

func NewRedisFlowCounter(key string, interval time.Duration) *RedisFlowCounter {
	return newRedisFlowCounter(key, interval, false)
}

func newRedisFlowCounter(key string, interval time.Duration, trackDay bool) *RedisFlowCounter {
	counter := &RedisFlowCounter{Key: key, Interval: interval, trackDay: trackDay}
	if trackDay {
		// 计数器在租户第一次请求时才创建，先同步读一次当天总量，否则网关重启后第一批请求会绕过 qpd 限流
		counter.loadDayCount()
	}
	go counter.loop()
	return counter
}

func (c *RedisFlowCounter) Increase() {
	atomic.AddInt64(&c.tick, 1)
}

// DayCount 返回当天请求量：redis 中已写入的量加上内存中还未写入的量
// 只有 trackDay 的计数器才会读回 redis，否则只有内存中未写入的部分
func (c *RedisFlowCounter) DayCount() int64 {
	c.dayLocker.Lock()
	n := c.flushing
	if c.day == time.Now().Format("20060102") {
		n += c.dayCount
	}
	c.dayLocker.Unlock()
	return n + atomic.LoadInt64(&c.tick)
}

func (c *RedisFlowCounter) loop() {
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()
	for range ticker.C {
		c.flush()
		if c.trackDay {
			c.loadDayCount()
		}
	}
}

func (c *RedisFlowCounter) loadDayCount() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf(" [ERROR] flow counter load day count key:%s panic:%v\n", c.Key, err)
		}
	}()
	now := time.Now()
	counts, err := hourCountList(c.Key, now)
	if err != nil {
		log.Printf(" [ERROR] flow counter load day count key:%s err:%v\n", c.Key, err)
		return
	}
	var total int64
	for _, n := range counts {
		total += n
	}
	c.dayLocker.Lock()
	c.day = now.Format("20060102")
	c.dayCount = total
	c.dayLocker.Unlock()
}

func (c *RedisFlowCounter) setFlushing(n int64) {
	c.dayLocker.Lock()
	c.flushing = n
	c.dayLocker.Unlock()
}

func (c *RedisFlowCounter) flush() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf(" [ERROR] flow counter flush key:%s panic:%v\n", c.Key, err)
		}
	}()

	c.dayLocker.Lock()
	n := atomic.SwapInt64(&c.tick, 0)
	c.flushing = n
	c.dayLocker.Unlock()
	if n == 0 {
		return
	}
	conn, err := lib.RedisConnFactory("default")
	if err != nil {
		// 写失败时把计数加回去，下次再试
		atomic.AddInt64(&c.tick, n)
		c.setFlushing(0)
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

func (m *FlowCounterManager) getCounter(key string, trackDay bool) *RedisFlowCounter {
	m.locker.Lock()
	defer m.locker.Unlock()
	if counter, ok := m.counterMap[key]; ok {
		return counter
	}
	counter := newRedisFlowCounter(key, time.Second, trackDay)
	m.counterMap[key] = counter
	return counter
}

func (m *FlowCounterManager) TotalCounter() *RedisFlowCounter {
	return m.getCounter(TotalFlowCountKey, false)
}

func (m *FlowCounterManager) ServiceCounter(serviceName string) *RedisFlowCounter {
	return m.getCounter(fmt.Sprintf("%s%s", FlowCountKeyPrefix, serviceName), false)
}

// AppCounter 租户计数器会跟踪当天总量，用于 qpd 限流
func (m *FlowCounterManager) AppCounter(appID string) *RedisFlowCounter {
	return m.getCounter(fmt.Sprintf("%s%s", AppFlowCountKeyPrefix, appID), true)
}
