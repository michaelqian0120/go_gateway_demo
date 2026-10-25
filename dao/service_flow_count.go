package dao

import (
	"time"
)

const FlowCountKeyPrefix = "flow_count_service_"

type ServiceFlowCount struct {
	ServiceName string
}

// HourCountList 返回指定日期 0点-23点 每小时的请求量
func (f *ServiceFlowCount) HourCountList(date time.Time) ([]int64, error) {
	return hourCountList(FlowCountKeyPrefix+f.ServiceName, date)
}
