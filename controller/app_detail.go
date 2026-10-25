package controller

import (
	"errors"

	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

// AppDetail godoc
// @Summary 租户详情
// @Description 租户详情
// @Tags 租户管理
// @ID /app/app_detail
// @Accept  json
// @Produce  json
// @Param id query int true "租户ID"
// @Success 200 {object} middleware.Response{data=dao.AppInfo} "success"
// @Router /app/app_detail [get]
func (app *AppController) AppDetail(c *gin.Context) {
	params := &dto.AppDetailInput{}
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

	middleware.ResponseSuccess(c, appInfo)
}
