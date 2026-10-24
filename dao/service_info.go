package dao

import (
	"gorm.io/gorm"
)

const (
	LoadTypeHTTP = 0
	LoadTypeTCP  = 1
	LoadTypeGRPC = 2
)

type ServiceInfo struct {
	Id          int64  `json:"id" gorm:"column:id;primary_key"`
	LoadType    int    `json:"load_type" gorm:"column:load_type"`
	ServiceName string `json:"service_name" gorm:"column:service_name"`
	ServiceDesc string `json:"service_desc" gorm:"column:service_desc"`
	IsDelete    int    `json:"is_delete" gorm:"column:is_delete"`
}

func (service *ServiceInfo) TableName() string {
	return "gateway_service_info"
}

func (service *ServiceInfo) PageList(tx *gorm.DB, keyword string, pageNo int, pageSize int) ([]ServiceInfo, int64, error) {
	query := tx.Model(&ServiceInfo{}).Where("is_delete=0")
	if keyword != "" {
		query = query.Where("(service_name like ? or service_desc like ?)", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	list := []ServiceInfo{}
	if err := query.Order("id desc").Offset((pageNo - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
