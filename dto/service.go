package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/public"
)

type ServiceListInput struct {
	Info     string `form:"info" json:"info" comment:"关键词" validate:""`
	PageNo   int    `form:"page_no" json:"page_no" comment:"页数" validate:"required"`
	PageSize int    `form:"page_size" json:"page_size" comment:"每页条数" validate:"required"`
}

func (params *ServiceListInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}

type ServiceListItemOutput struct {
	Id          int64  `json:"id"`
	ServiceName string `json:"service_name"`
	ServiceDesc string `json:"service_desc"`
	ServiceType string `json:"service_type"`
	ServiceAddr string `json:"service_addr"`
	Qps         int64  `json:"qps"`
	Qpd         int64  `json:"qpd"`
	TotalNode   int    `json:"total_node"`
}

type ServiceListOutput struct {
	Total int64                   `json:"total"`
	List  []ServiceListItemOutput `json:"list"`
}
