package controller

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

// ServiceAddTCP godoc
// @Summary 添加TCP服务
// @Description 添加TCP服务
// @Tags 服务管理
// @ID /service/service_add_tcp
// @Accept  json
// @Produce  json
// @Param body body dto.ServiceAddTCPInput true "body"
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /service/service_add_tcp [post]
func (service *ServiceController) ServiceAddTCP(c *gin.Context) {
	params := &dto.ServiceAddTCPInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	if existTCPRule, err := (&dao.ServiceTCPRule{}).FindByPort(tx, params.Port); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	} else if existTCPRule.Id != 0 {
		middleware.ResponseError(c, middleware.ValidErrorCode, errors.New("端口已被占用"))
		return
	}

	txn := tx.Begin()

	serviceInfo := &dao.ServiceInfo{
		LoadType:    dao.LoadTypeTCP,
		ServiceName: params.ServiceName,
		ServiceDesc: params.ServiceDesc,
		CreateAt:    time.Now(),
		UpdateAt:    time.Now(),
	}
	if err := serviceInfo.Save(txn); err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	tcpRule := &dao.ServiceTCPRule{
		ServiceId: serviceInfo.Id,
		Port:      params.Port,
	}
	if err := tcpRule.Save(txn); err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	accessControl := &dao.ServiceAccessControl{
		ServiceId:         serviceInfo.Id,
		OpenAuth:          params.OpenAuth,
		BlackList:         params.BlackList,
		WhiteList:         params.WhiteList,
		WhiteHostName:     params.WhiteHostName,
		ClientIPFlowLimit: params.ClientIPFlowLimit,
		ServiceFlowLimit:  params.ServiceFlowLimit,
	}
	if err := accessControl.Save(txn); err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	loadBalance := &dao.ServiceLoadBalance{
		ServiceId:              serviceInfo.Id,
		CheckMethod:            params.CheckMethod,
		CheckTimeout:           params.CheckTimeout,
		CheckInterval:          params.CheckInterval,
		RoundType:              params.RoundType,
		IpList:                 params.IpList,
		WeightList:             params.WeightList,
		ForbidList:             params.ForbidList,
		UpstreamConnectTimeout: params.UpstreamConnectTimeout,
		UpstreamHeaderTimeout:  params.UpstreamHeaderTimeout,
		UpstreamIdleTimeout:    params.UpstreamIdleTimeout,
		UpstreamMaxIdle:        params.UpstreamMaxIdle,
	}
	if err := loadBalance.Save(txn); err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	if err := txn.Commit().Error; err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	middleware.ResponseSuccess(c, "添加成功")
}
