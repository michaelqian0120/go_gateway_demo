package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/public"
)

type ServiceDetailInput struct {
	Id int64 `form:"id" json:"id" comment:"服务ID" validate:"required"`
}

func (params *ServiceDetailInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}

type ServiceDetailOutput struct {
	Info          *dao.ServiceInfo          `json:"info"`
	HTTPRule      *dao.ServiceHTTPRule      `json:"http_rule,omitempty"`
	TCPRule       *dao.ServiceTCPRule       `json:"tcp_rule,omitempty"`
	GRPCRule      *dao.ServiceGRPCRule      `json:"grpc_rule,omitempty"`
	AccessControl *dao.ServiceAccessControl `json:"access_control"`
	LoadBalance   *dao.ServiceLoadBalance   `json:"load_balance"`
}
