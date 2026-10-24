package dao

import (
	"time"

	"gorm.io/gorm"
)

type AdminInfo struct {
	Id       int64     `json:"id" gorm:"column:id;primary_key"`
	UserName string    `json:"user_name" gorm:"column:user_name"`
	Salt     string    `json:"salt" gorm:"column:salt"`
	Password string    `json:"password" gorm:"column:password"`
	CreateAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdateAt time.Time `json:"update_at" gorm:"column:update_at"`
	IsDelete int       `json:"is_delete" gorm:"column:is_delete"`
}

func (admin *AdminInfo) TableName() string {
	return "gateway_admin"
}

func (admin *AdminInfo) Find(tx *gorm.DB, search *AdminInfo) (*AdminInfo, error) {
	out := &AdminInfo{}
	err := tx.Where(search).Find(out).Error
	return out, err
}
