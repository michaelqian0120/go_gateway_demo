package dto

import (
	"github.com/gin-gonic/gin"
	"go_gateway_demo/public"
)

type ServiceAddTCPInput struct {
	ServiceName string `form:"service_name" json:"service_name" comment:"服务名称" validate:"required,service_name,unique_service_name"`
	ServiceDesc string `form:"service_desc" json:"service_desc" comment:"服务描述" validate:"required,max=255"`
	Port        int    `form:"port" json:"port" comment:"端口" validate:"required,min=1,max=65535"`

	OpenAuth          int    `form:"open_auth" json:"open_auth" comment:"是否开启权限 1=开启" validate:"min=0,max=1"`
	BlackList         string `form:"black_list" json:"black_list" comment:"黑名单ip"`
	WhiteList         string `form:"white_list" json:"white_list" comment:"白名单ip"`
	WhiteHostName     string `form:"white_host_name" json:"white_host_name" comment:"白名单主机"`
	ClientIPFlowLimit int    `form:"clientip_flow_limit" json:"clientip_flow_limit" comment:"客户端ip限流"`
	ServiceFlowLimit  int    `form:"service_flow_limit" json:"service_flow_limit" comment:"服务端限流"`

	RoundType              int    `form:"round_type" json:"round_type" comment:"轮询方式 0=random 1=round-robin 2=weight_round-robin 3=ip_hash" validate:"min=0,max=3"`
	IpList                 string `form:"ip_list" json:"ip_list" comment:"ip列表" validate:"required"`
	WeightList             string `form:"weight_list" json:"weight_list" comment:"权重列表" validate:"required"`
	ForbidList             string `form:"forbid_list" json:"forbid_list" comment:"禁用ip列表"`
	CheckMethod            int    `form:"check_method" json:"check_method" comment:"检查方法 0=tcpchk"`
	CheckTimeout           int    `form:"check_timeout" json:"check_timeout" comment:"check超时时间,单位s"`
	CheckInterval          int    `form:"check_interval" json:"check_interval" comment:"检查间隔,单位s"`
	UpstreamConnectTimeout int    `form:"upstream_connect_timeout" json:"upstream_connect_timeout" comment:"建立连接超时,单位s"`
	UpstreamHeaderTimeout  int    `form:"upstream_header_timeout" json:"upstream_header_timeout" comment:"获取header超时,单位s"`
	UpstreamIdleTimeout    int    `form:"upstream_idle_timeout" json:"upstream_idle_timeout" comment:"链接最大空闲时间,单位s"`
	UpstreamMaxIdle        int    `form:"upstream_max_idle" json:"upstream_max_idle" comment:"最大空闲链接数"`
}

func (params *ServiceAddTCPInput) BindingValidParams(c *gin.Context) error {
	return public.DefaultGetValidParams(c, params)
}
