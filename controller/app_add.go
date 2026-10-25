package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

// AppAdd godoc
// @Summary 租户添加
// @Description 租户添加
// @Tags 租户管理
// @ID /app/app_add
// @Accept  json
// @Produce  json
// @Param body body dto.AppAddInput true "body"
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /app/app_add [post]
func (app *AppController) AppAdd(c *gin.Context) {
	params := &dto.AppAddInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	secret := params.Secret
	if secret == "" {
		secret = lib.GetMd5Hash(fmt.Sprintf("%s_%d", params.AppID, time.Now().UnixNano()))
	}

	appInfo := &dao.AppInfo{
		AppID:    params.AppID,
		Name:     params.Name,
		Secret:   secret,
		Qps:      params.Qps,
		Qpd:      params.Qpd,
		CreateAt: time.Now(),
		UpdateAt: time.Now(),
	}
	if err := appInfo.Save(tx); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	middleware.ResponseSuccess(c, "添加成功")
}
