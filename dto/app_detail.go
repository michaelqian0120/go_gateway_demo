package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/public"
)

type AppDetailInput struct {
	Id int64 `form:"id" json:"id" comment:"租户ID" validate:"required"`
}

func (params *AppDetailInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}
