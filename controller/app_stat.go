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

// AppStat godoc
// @Summary 租户统计
// @Description 租户统计(今日/昨日每小时请求量)
// @Tags 租户管理
// @ID /app/app_stat
// @Accept  json
// @Produce  json
// @Param id query int true "租户ID"
// @Success 200 {object} middleware.Response{data=dto.AppStatOutput} "success"
// @Router /app/app_stat [get]
func (app *AppController) AppStat(c *gin.Context) {
	params := &dto.AppStatInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	appInfo, err := (&dao.AppInfo{}).Find(tx, &dao.AppInfo{Id: params.Id})
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	if appInfo.Id == 0 || appInfo.IsDelete == 1 {
		middleware.ResponseError(c, middleware.ValidErrorCode, errors.New("租户不存在"))
		return
	}

	flowCount := &dao.AppFlowCount{AppID: appInfo.AppID}
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

	middleware.ResponseSuccess(c, &dto.AppStatOutput{
		AppID:     appInfo.AppID,
		Today:     today,
		Yesterday: yesterday,
	})
}
