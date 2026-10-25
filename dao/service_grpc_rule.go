package dao

import (
	"gorm.io/gorm"
)

type ServiceGRPCRule struct {
	Id             int64  `json:"id" gorm:"column:id;primary_key"`
	ServiceId      int64  `json:"service_id" gorm:"column:service_id"`
	Port           int    `json:"port" gorm:"column:port"`
	HeaderTransfor string `json:"header_transfor" gorm:"column:header_transfor"`
}

func (t *ServiceGRPCRule) TableName() string {
	return "gateway_service_grpc_rule"
}

func (t *ServiceGRPCRule) Find(tx *gorm.DB, serviceId int64) (*ServiceGRPCRule, error) {
	out := &ServiceGRPCRule{}
	err := tx.Where("service_id=?", serviceId).Find(out).Error
	return out, err
}

func (t *ServiceGRPCRule) FindByPort(tx *gorm.DB, port int) (*ServiceGRPCRule, error) {
	out := &ServiceGRPCRule{}
	err := tx.Where("port=?", port).Find(out).Error
	return out, err
}

func (t *ServiceGRPCRule) Save(tx *gorm.DB) error {
	return tx.Save(t).Error
}
