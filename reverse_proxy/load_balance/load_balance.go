package load_balance

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

type LoadBalance interface {
	Add(...string) error
	Get(key string) (string, error)
}

// RandomBalance 随机负载均衡
type RandomBalance struct {
	rss    []string
	rnd    *rand.Rand
	locker sync.Mutex
}

func NewRandomBalance() *RandomBalance {
	return &RandomBalance{rnd: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (r *RandomBalance) Add(params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	r.rss = append(r.rss, params...)
	return nil
}

func (r *RandomBalance) Get(key string) (string, error) {
	if len(r.rss) == 0 {
		return "", errors.New("no available node")
	}
	r.locker.Lock()
	defer r.locker.Unlock()
	return r.rss[r.rnd.Intn(len(r.rss))], nil
}
