package dao

import (
	"time"

	"gorm.io/gorm"
)

type AppInfo struct {
	Id       int64     `json:"id" gorm:"column:id;primary_key"`
	AppID    string    `json:"app_id" gorm:"column:app_id"`
	Name     string    `json:"name" gorm:"column:name"`
	Secret   string    `json:"secret" gorm:"column:secret"`
	WhiteIps string    `json:"white_ips" gorm:"column:white_ips"`
	Qpd      int64     `json:"qpd" gorm:"column:qpd"`
	Qps      int64     `json:"qps" gorm:"column:qps"`
	CreateAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdateAt time.Time `json:"update_at" gorm:"column:update_at"`
	IsDelete int       `json:"is_delete" gorm:"column:is_delete"`
}

func (app *AppInfo) TableName() string {
	return "gateway_app"
}

func (app *AppInfo) Find(tx *gorm.DB, search *AppInfo) (*AppInfo, error) {
	out := &AppInfo{}
	err := tx.Where(search).Find(out).Error
	return out, err
}

func (app *AppInfo) FindByAppID(tx *gorm.DB, appId string) (*AppInfo, error) {
	out := &AppInfo{}
	err := tx.Where("app_id=? and is_delete=0", appId).Find(out).Error
	return out, err
}

func (app *AppInfo) Save(tx *gorm.DB) error {
	return tx.Save(app).Error
}

func (app *AppInfo) Count(tx *gorm.DB) (int64, error) {
	var total int64
	err := tx.Model(&AppInfo{}).Where("is_delete=0").Count(&total).Error
	return total, err
}

func (app *AppInfo) PageList(tx *gorm.DB, keyword string, pageNo int, pageSize int) ([]AppInfo, int64, error) {
	query := tx.Model(&AppInfo{}).Where("is_delete=0")
	if keyword != "" {
		query = query.Where("(app_id like ? or name like ?)", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	list := []AppInfo{}
	if err := query.Order("id desc").Offset((pageNo - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
