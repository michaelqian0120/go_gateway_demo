package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/public"
)

type AppAddInput struct {
	AppID  string `form:"app_id" json:"app_id" comment:"app_id" validate:"required,service_name,unique_app_id"`
	Name   string `form:"name" json:"name" comment:"租户名称" validate:"required,max=255"`
	Secret string `form:"secret" json:"secret" comment:"密钥" validate:"omitempty,len=32"`
	Qps    int64  `form:"qps" json:"qps" comment:"QPS限流,0表示无限制" validate:"min=0"`
	Qpd    int64  `form:"qpd" json:"qpd" comment:"日请求量限流,0表示无限制" validate:"min=0"`
}

func (params *AppAddInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}
