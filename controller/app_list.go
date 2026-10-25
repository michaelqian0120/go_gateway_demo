package controller

import (
	"errors"

	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

type AppController struct {
}

func AppRegister(group *gin.RouterGroup) {
	app := &AppController{}
	group.GET("/app_list", middleware.SessionAuthMiddleware(), app.AppList)
	group.GET("/app_delete", middleware.SessionAuthMiddleware(), app.AppDelete)
	group.GET("/app_detail", middleware.SessionAuthMiddleware(), app.AppDetail)
	group.GET("/app_stat", middleware.SessionAuthMiddleware(), app.AppStat)
	group.POST("/app_add", middleware.SessionAuthMiddleware(), app.AppAdd)
	group.POST("/app_update", middleware.SessionAuthMiddleware(), app.AppUpdate)
}

// AppList godoc
// @Summary 租户列表
// @Description 租户列表
// @Tags 租户管理
// @ID /app/app_list
// @Accept  json
// @Produce  json
// @Param info query string false "关键词(app_id/租户名称)"
// @Param page_no query int true "页数"
// @Param page_size query int true "每页条数"
// @Success 200 {object} middleware.Response{data=dto.AppListOutput} "success"
// @Router /app/app_list [get]
func (app *AppController) AppList(c *gin.Context) {
	params := &dto.AppListInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	list, total, err := (&dao.AppInfo{}).PageList(tx, params.Info, params.PageNo, params.PageSize)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	outList := []dto.AppListItemOutput{}
	for _, item := range list {
		outList = append(outList, dto.AppListItemOutput{
			Id:      item.Id,
			AppID:   item.AppID,
			Name:    item.Name,
			Secret:  item.Secret,
			Qps:     item.Qps,
			Qpd:     item.Qpd,
			RealQps: 0,
			RealQpd: 0,
		})
	}

	middleware.ResponseSuccess(c, &dto.AppListOutput{
		Total: total,
		List:  outList,
	})
}

// AppDelete godoc
// @Summary 租户删除
// @Description 租户删除
// @Tags 租户管理
// @ID /app/app_delete
// @Accept  json
// @Produce  json
// @Param id query int true "租户ID"
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /app/app_delete [get]
func (app *AppController) AppDelete(c *gin.Context) {
	params := &dto.AppDeleteInput{}
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

	appInfo.IsDelete = 1
	if err := appInfo.Save(tx); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	middleware.ResponseSuccess(c, "删除成功")
}
