package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/public"
)

type AppStatInput struct {
	Id int64 `form:"id" json:"id" comment:"租户ID" validate:"required"`
}

func (params *AppStatInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}

type AppStatOutput struct {
	AppID     string  `json:"app_id"`
	Today     []int64 `json:"today"`
	Yesterday []int64 `json:"yesterday"`
}
