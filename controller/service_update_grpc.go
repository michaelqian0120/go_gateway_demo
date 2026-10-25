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

// ServiceUpdateGRPC godoc
// @Summary 更新GRPC服务
// @Description 更新GRPC服务
// @Tags 服务管理
// @ID /service/service_update_grpc
// @Accept  json
// @Produce  json
// @Param body body dto.ServiceUpdateGRPCInput true "body"
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /service/service_update_grpc [post]
func (service *ServiceController) ServiceUpdateGRPC(c *gin.Context) {
	params := &dto.ServiceUpdateGRPCInput{}
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

	if existGRPCRule, err := (&dao.ServiceGRPCRule{}).FindByPort(tx, params.Port); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	} else if existGRPCRule.Id != 0 && existGRPCRule.ServiceId != serviceInfo.Id {
		middleware.ResponseError(c, middleware.ValidErrorCode, errors.New("端口已被占用"))
		return
	}

	txn := tx.Begin()

	serviceInfo.ServiceDesc = params.ServiceDesc
	serviceInfo.UpdateAt = time.Now()
	if err := serviceInfo.Save(txn); err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	grpcRule, err := (&dao.ServiceGRPCRule{}).Find(txn, serviceInfo.Id)
	if err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	grpcRule.Port = params.Port
	grpcRule.HeaderTransfor = params.HeaderTransfor
	if err := grpcRule.Save(txn); err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	accessControl, err := (&dao.ServiceAccessControl{}).Find(txn, serviceInfo.Id)
	if err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	accessControl.OpenAuth = params.OpenAuth
	accessControl.BlackList = params.BlackList
	accessControl.WhiteList = params.WhiteList
	accessControl.WhiteHostName = params.WhiteHostName
	accessControl.ClientIPFlowLimit = params.ClientIPFlowLimit
	accessControl.ServiceFlowLimit = params.ServiceFlowLimit
	if err := accessControl.Save(txn); err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	loadBalance, err := (&dao.ServiceLoadBalance{}).Find(txn, serviceInfo.Id)
	if err != nil {
		txn.Rollback()
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	loadBalance.RoundType = params.RoundType
	loadBalance.IpList = params.IpList
	loadBalance.WeightList = params.WeightList
	loadBalance.ForbidList = params.ForbidList
	loadBalance.CheckMethod = params.CheckMethod
	loadBalance.CheckTimeout = params.CheckTimeout
	loadBalance.CheckInterval = params.CheckInterval
	loadBalance.UpstreamConnectTimeout = params.UpstreamConnectTimeout
	loadBalance.UpstreamHeaderTimeout = params.UpstreamHeaderTimeout
	loadBalance.UpstreamIdleTimeout = params.UpstreamIdleTimeout
	loadBalance.UpstreamMaxIdle = params.UpstreamMaxIdle
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

	middleware.ResponseSuccess(c, "更新成功")
}
