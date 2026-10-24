package dao

import (
	"strings"

	"gorm.io/gorm"
)

type ServiceLoadBalance struct {
	Id                     int64  `json:"id" gorm:"column:id;primary_key"`
	ServiceId              int64  `json:"service_id" gorm:"column:service_id"`
	CheckMethod            int    `json:"check_method" gorm:"column:check_method"`
	CheckTimeout           int    `json:"check_timeout" gorm:"column:check_timeout"`
	CheckInterval          int    `json:"check_interval" gorm:"column:check_interval"`
	RoundType              int    `json:"round_type" gorm:"column:round_type"`
	IpList                 string `json:"ip_list" gorm:"column:ip_list"`
	WeightList             string `json:"weight_list" gorm:"column:weight_list"`
	ForbidList             string `json:"forbid_list" gorm:"column:forbid_list"`
	UpstreamConnectTimeout int    `json:"upstream_connect_timeout" gorm:"column:upstream_connect_timeout"`
	UpstreamHeaderTimeout  int    `json:"upstream_header_timeout" gorm:"column:upstream_header_timeout"`
	UpstreamIdleTimeout    int    `json:"upstream_idle_timeout" gorm:"column:upstream_idle_timeout"`
	UpstreamMaxIdle        int    `json:"upstream_max_idle" gorm:"column:upstream_max_idle"`
}

func (t *ServiceLoadBalance) TableName() string {
	return "gateway_service_load_balance"
}

func (t *ServiceLoadBalance) Find(tx *gorm.DB, serviceId int64) (*ServiceLoadBalance, error) {
	out := &ServiceLoadBalance{}
	err := tx.Where("service_id=?", serviceId).Find(out).Error
	return out, err
}

func (t *ServiceLoadBalance) NodeCount() int {
	if t.IpList == "" {
		return 0
	}
	return len(strings.Split(t.IpList, ","))
}
