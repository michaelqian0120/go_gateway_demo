package controller

import (
	"errors"

	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

// ServiceDetail godoc
// @Summary 服务详情
// @Description 服务详情
// @Tags 服务管理
// @ID /service/service_detail
// @Accept  json
// @Produce  json
// @Param id query int true "服务ID"
// @Success 200 {object} middleware.Response{data=dto.ServiceDetailOutput} "success"
// @Router /service/service_detail [get]
func (service *ServiceController) ServiceDetail(c *gin.Context) {
	params := &dto.ServiceDetailInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	serviceInfo, err := (&dao.ServiceInfo{}).Find(tx, &dao.ServiceInfo{Id: params.Id})
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	if serviceInfo.Id == 0 || serviceInfo.IsDelete == 1 {
		middleware.ResponseError(c, middleware.ValidErrorCode, errors.New("服务不存在"))
		return
	}

	out := &dto.ServiceDetailOutput{Info: serviceInfo}

	switch serviceInfo.LoadType {
	case dao.LoadTypeHTTP:
		httpRule, err := (&dao.ServiceHTTPRule{}).Find(tx, serviceInfo.Id)
		if err != nil {
			middleware.ResponseError(c, middleware.InternalErrorCode, err)
			return
		}
		out.HTTPRule = httpRule
	case dao.LoadTypeTCP:
		tcpRule, err := (&dao.ServiceTCPRule{}).Find(tx, serviceInfo.Id)
		if err != nil {
			middleware.ResponseError(c, middleware.InternalErrorCode, err)
			return
		}
		out.TCPRule = tcpRule
	case dao.LoadTypeGRPC:
		grpcRule, err := (&dao.ServiceGRPCRule{}).Find(tx, serviceInfo.Id)
		if err != nil {
			middleware.ResponseError(c, middleware.InternalErrorCode, err)
			return
		}
		out.GRPCRule = grpcRule
	}

	accessControl, err := (&dao.ServiceAccessControl{}).Find(tx, serviceInfo.Id)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	out.AccessControl = accessControl

	loadBalance, err := (&dao.ServiceLoadBalance{}).Find(tx, serviceInfo.Id)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	out.LoadBalance = loadBalance

	middleware.ResponseSuccess(c, out)
}
