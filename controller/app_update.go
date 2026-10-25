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

// AppUpdate godoc
// @Summary 租户更新
// @Description 租户更新
// @Tags 租户管理
// @ID /app/app_update
// @Accept  json
// @Produce  json
// @Param body body dto.AppUpdateInput true "body"
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /app/app_update [post]
func (app *AppController) AppUpdate(c *gin.Context) {
	params := &dto.AppUpdateInput{}
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

	appInfo.Name = params.Name
	if params.Secret != "" {
		appInfo.Secret = params.Secret
	}
	appInfo.Qps = params.Qps
	appInfo.Qpd = params.Qpd
	appInfo.UpdateAt = time.Now()

	if err := appInfo.Save(tx); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	middleware.ResponseSuccess(c, "更新成功")
}
