package dao

import (
	"fmt"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"golang_common/lib"
)

const TotalFlowCountKey = "flow_count_total"

// hourCountList 返回指定 redis hash key 在指定日期 0点-23点 每小时的请求量
func hourCountList(key string, date time.Time) ([]int64, error) {
	conn, err := lib.RedisConnFactory("default")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	all, err := redis.StringMap(conn.Do("HGETALL", key))
	if err != nil && err != redis.ErrNil {
		return nil, err
	}

	datePrefix := date.Format("20060102")
	counts := make([]int64, 24)
	for hour := 0; hour < 24; hour++ {
		field := fmt.Sprintf("%s%02d", datePrefix, hour)
		if v, ok := all[field]; ok {
			n, _ := strconv.ParseInt(v, 10, 64)
			counts[hour] = n
		}
	}
	return counts, nil
}

// TotalFlowCount 网关整体流量统计（所有服务汇总）
type TotalFlowCount struct {
}

func (f *TotalFlowCount) HourCountList(date time.Time) ([]int64, error) {
	return hourCountList(TotalFlowCountKey, date)
}
