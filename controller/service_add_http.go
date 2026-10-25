package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

// ServiceAddHTTP godoc
// @Summary 添加HTTP服务
// @Description 添加HTTP服务
// @Tags 服务管理
// @ID /service/service_add_http
// @Accept  json
// @Produce  json
// @Param body body dto.ServiceAddHTTPInput true "body"
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /service/service_add_http [post]
func (service *ServiceController) ServiceAddHTTP(c *gin.Context) {
	params := &dto.ServiceAddHTTPInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	if len(strings.Split(params.IpList, ",")) != len(strings.Split(params.WeightList, ",")) {
		middleware.ResponseError(c, middleware.ValidErrorCode, errors.New("ip列表与权重列表数量不一致"))
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	if existServiceInfo, err := (&dao.ServiceInfo{}).FindByName(tx, params.ServiceName); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	} else if existServiceInfo.Id != 0 {
		middleware.ResponseError(c, middleware.ValidErrorCode, errors.New("服务名称已存在"))
		return
	}

	if existHTTPRule, err := (&dao.ServiceHTTPRule{}).FindByRule(tx, params.Rule); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	} else if existHTTPRule.Id != 0 {
		middleware.ResponseError(c, middleware.ValidErrorCode, errors.New("服务接入规则已存在"))
		return
	}

	txn := tx.Begin()

	serviceInfo := &dao.ServiceInfo{
		LoadType:    dao.LoadTypeHTTP,
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

	httpRule := &dao.ServiceHTTPRule{
		ServiceId:      serviceInfo.Id,
		RuleType:       params.RuleType,
		Rule:           params.Rule,
		NeedHttps:      params.NeedHttps,
		NeedStripUri:   params.NeedStripUri,
		NeedWebsocket:  params.NeedWebsocket,
		UrlRewrite:     params.UrlRewrite,
		HeaderTransfor: params.HeaderTransfor,
	}
	if err := httpRule.Save(txn); err != nil {
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
