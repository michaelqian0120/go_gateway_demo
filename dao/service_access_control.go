package dao

import (
	"gorm.io/gorm"
)

type ServiceAccessControl struct {
	Id                int64  `json:"id" gorm:"column:id;primary_key"`
	ServiceId         int64  `json:"service_id" gorm:"column:service_id"`
	OpenAuth          int    `json:"open_auth" gorm:"column:open_auth"`
	BlackList         string `json:"black_list" gorm:"column:black_list"`
	WhiteList         string `json:"white_list" gorm:"column:white_list"`
	WhiteHostName     string `json:"white_host_name" gorm:"column:white_host_name"`
	ClientIPFlowLimit int    `json:"clientip_flow_limit" gorm:"column:clientip_flow_limit"`
	ServiceFlowLimit  int    `json:"service_flow_limit" gorm:"column:service_flow_limit"`
}

func (t *ServiceAccessControl) TableName() string {
	return "gateway_service_access_control"
}

func (t *ServiceAccessControl) Find(tx *gorm.DB, serviceId int64) (*ServiceAccessControl, error) {
	out := &ServiceAccessControl{}
	err := tx.Where("service_id=?", serviceId).Find(out).Error
	return out, err
}

func (t *ServiceAccessControl) Save(tx *gorm.DB) error {
	return tx.Save(t).Error
}
