package controller

import (
	"time"

	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

type DashboardController struct {
}

func DashboardRegister(group *gin.RouterGroup) {
	dashboard := &DashboardController{}
	group.GET("/panel_group_data", middleware.SessionAuthMiddleware(), dashboard.PanelGroupData)
	group.GET("/flow_stat", middleware.SessionAuthMiddleware(), dashboard.FlowStat)
}

// PanelGroupData godoc
// @Summary 首页大盘统计
// @Description 首页大盘统计(服务数/当日请求量/当前QPS/租户数)
// @Tags 首页大盘
// @ID /dashboard/panel_group_data
// @Accept  json
// @Produce  json
// @Success 200 {object} middleware.Response{data=dto.PanelGroupDataOutput} "success"
// @Router /dashboard/panel_group_data [get]
func (dashboard *DashboardController) PanelGroupData(c *gin.Context) {
	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	serviceNum, err := (&dao.ServiceInfo{}).Count(tx)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	appNum, err := (&dao.AppInfo{}).Count(tx)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	todayCounts, err := (&dao.TotalFlowCount{}).HourCountList(time.Now())
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	var todayRequestNum int64
	for _, n := range todayCounts {
		todayRequestNum += n
	}

	middleware.ResponseSuccess(c, &dto.PanelGroupDataOutput{
		ServiceNum:      serviceNum,
		TodayRequestNum: todayRequestNum,
		CurrentQps:      0,
		AppNum:          appNum,
	})
}

// FlowStat godoc
// @Summary 首页大盘流量统计
// @Description 首页大盘流量统计(今日/昨日流量曲线 + 服务类型占比)
// @Tags 首页大盘
// @ID /dashboard/flow_stat
// @Accept  json
// @Produce  json
// @Success 200 {object} middleware.Response{data=dto.DashFlowStatOutput} "success"
// @Router /dashboard/flow_stat [get]
func (dashboard *DashboardController) FlowStat(c *gin.Context) {
	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	flowCount := &dao.TotalFlowCount{}
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

	typeCounts, err := (&dao.ServiceInfo{}).GroupByLoadType(tx)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	ratio := []dto.ServiceTypeRatioItem{
		{Name: "HTTP", Value: 0},
		{Name: "TCP", Value: 0},
		{Name: "GRPC", Value: 0},
	}
	for _, tc := range typeCounts {
		switch tc.LoadType {
		case dao.LoadTypeHTTP:
			ratio[0].Value = tc.Count
		case dao.LoadTypeTCP:
			ratio[1].Value = tc.Count
		case dao.LoadTypeGRPC:
			ratio[2].Value = tc.Count
		}
	}

	middleware.ResponseSuccess(c, &dto.DashFlowStatOutput{
		Today:            today,
		Yesterday:        yesterday,
		ServiceTypeRatio: ratio,
	})
}
