package dao

import (
	"gorm.io/gorm"
)

type ServiceHTTPRule struct {
	Id             int64  `json:"id" gorm:"column:id;primary_key"`
	ServiceId      int64  `json:"service_id" gorm:"column:service_id"`
	RuleType       int    `json:"rule_type" gorm:"column:rule_type"`
	Rule           string `json:"rule" gorm:"column:rule"`
	NeedHttps      int    `json:"need_https" gorm:"column:need_https"`
	NeedStripUri   int    `json:"need_strip_uri" gorm:"column:need_strip_uri"`
	NeedWebsocket  int    `json:"need_websocket" gorm:"column:need_websocket"`
	UrlRewrite     string `json:"url_rewrite" gorm:"column:url_rewrite"`
	HeaderTransfor string `json:"header_transfor" gorm:"column:header_transfor"`
}

func (t *ServiceHTTPRule) TableName() string {
	return "gateway_service_http_rule"
}

func (t *ServiceHTTPRule) Find(tx *gorm.DB, serviceId int64) (*ServiceHTTPRule, error) {
	out := &ServiceHTTPRule{}
	err := tx.Where("service_id=?", serviceId).Find(out).Error
	return out, err
}
