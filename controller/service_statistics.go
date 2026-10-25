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

// ServiceStatistics godoc
// @Summary 服务流量统计
// @Description 服务流量统计(今日/昨日每小时请求量)
// @Tags 服务管理
// @ID /service/service_statistics
// @Accept  json
// @Produce  json
// @Param id query int true "服务ID"
// @Success 200 {object} middleware.Response{data=dto.ServiceStatisticsOutput} "success"
// @Router /service/service_statistics [get]
func (service *ServiceController) ServiceStatistics(c *gin.Context) {
	params := &dto.ServiceStatisticsInput{}
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

	flowCount := &dao.ServiceFlowCount{ServiceName: serviceInfo.ServiceName}

	today, err := flowCount.HourCountList(time.Now())
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	yesterday, err := flowCount.HourCountList(time.Now().AddDate(0, 0, -1))
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	middleware.ResponseSuccess(c, &dto.ServiceStatisticsOutput{
		ServiceName: serviceInfo.ServiceName,
		Today:       today,
		Yesterday:   yesterday,
	})
}
