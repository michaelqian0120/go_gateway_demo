package dao

import (
	"gorm.io/gorm"
)

type ServiceTCPRule struct {
	Id        int64 `json:"id" gorm:"column:id;primary_key"`
	ServiceId int64 `json:"service_id" gorm:"column:service_id"`
	Port      int   `json:"port" gorm:"column:port"`
}

func (t *ServiceTCPRule) TableName() string {
	return "gateway_service_tcp_rule"
}

func (t *ServiceTCPRule) Find(tx *gorm.DB, serviceId int64) (*ServiceTCPRule, error) {
	out := &ServiceTCPRule{}
	err := tx.Where("service_id=?", serviceId).Find(out).Error
	return out, err
}
