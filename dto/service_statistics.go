package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/public"
)

type ServiceStatisticsInput struct {
	Id int64 `form:"id" json:"id" comment:"服务ID" validate:"required"`
}

func (params *ServiceStatisticsInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}

type ServiceStatisticsOutput struct {
	ServiceName string  `json:"service_name"`
	Today       []int64 `json:"today"`
	Yesterday   []int64 `json:"yesterday"`
}
