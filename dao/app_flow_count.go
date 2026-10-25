package dao

import (
	"time"
)

const AppFlowCountKeyPrefix = "flow_count_app_"

type AppFlowCount struct {
	AppID string
}

// HourCountList 返回指定日期 0点-23点 每小时的请求量
func (f *AppFlowCount) HourCountList(date time.Time) ([]int64, error) {
	return hourCountList(AppFlowCountKeyPrefix+f.AppID, date)
}
