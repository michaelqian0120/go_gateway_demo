package dao

import (
	"strings"

	"gorm.io/gorm"
)

type ServiceHTTPRule struct {
	Id        int64  `json:"id" gorm:"column:id;primary_key"`
	ServiceId int64  `json:"service_id" gorm:"column:service_id"`
	RuleType  int    `json:"rule_type" gorm:"column:rule_type"`
	Rule      string `json:"rule" gorm:"column:rule"`
}

func (t *ServiceHTTPRule) TableName() string {
	return "gateway_service_http_rule"
}

func (t *ServiceHTTPRule) Find(tx *gorm.DB, serviceId int64) (*ServiceHTTPRule, error) {
	out := &ServiceHTTPRule{}
	err := tx.Where("service_id=?", serviceId).Find(out).Error
	return out, err
}

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

type ServiceGRPCRule struct {
	Id        int64 `json:"id" gorm:"column:id;primary_key"`
	ServiceId int64 `json:"service_id" gorm:"column:service_id"`
	Port      int   `json:"port" gorm:"column:port"`
}

func (t *ServiceGRPCRule) TableName() string {
	return "gateway_service_grpc_rule"
}

func (t *ServiceGRPCRule) Find(tx *gorm.DB, serviceId int64) (*ServiceGRPCRule, error) {
	out := &ServiceGRPCRule{}
	err := tx.Where("service_id=?", serviceId).Find(out).Error
	return out, err
}

type ServiceLoadBalance struct {
	Id        int64  `json:"id" gorm:"column:id;primary_key"`
	ServiceId int64  `json:"service_id" gorm:"column:service_id"`
	IpList    string `json:"ip_list" gorm:"column:ip_list"`
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
