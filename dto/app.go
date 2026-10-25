package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/public"
)

type AppListInput struct {
	Info     string `form:"info" json:"info" comment:"关键词(app_id/租户名称)" validate:""`
	PageNo   int    `form:"page_no" json:"page_no" comment:"页数" validate:"required"`
	PageSize int    `form:"page_size" json:"page_size" comment:"每页条数" validate:"required"`
}

func (params *AppListInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}

type AppListItemOutput struct {
	Id      int64  `json:"id"`
	AppID   string `json:"app_id"`
	Name    string `json:"name"`
	Secret  string `json:"secret"`
	Qps     int64  `json:"qps"`
	Qpd     int64  `json:"qpd"`
	RealQps int64  `json:"real_qps"`
	RealQpd int64  `json:"real_qpd"`
}

type AppListOutput struct {
	Total int64               `json:"total"`
	List  []AppListItemOutput `json:"list"`
}

type AppDeleteInput struct {
	Id int64 `form:"id" json:"id" comment:"租户ID" validate:"required"`
}

func (params *AppDeleteInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}
