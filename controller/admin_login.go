package controller

import (
	"errors"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"

	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

type AdminLoginController struct {
}

func AdminLoginRegister(group *gin.RouterGroup) {
	admin := &AdminLoginController{}
	group.POST("/login", admin.AdminLogin)
	group.GET("/logout", admin.AdminLoginOut)
}

// AdminLogin godoc
// @Summary 管理员登录
// @Description 管理员登录
// @Tags 管理员接口
// @ID /admin_login/login
// @Accept  json
// @Produce  json
// @Param body body dto.AdminLoginInput true "body"
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /admin_login/login [post]
func (admin *AdminLoginController) AdminLogin(c *gin.Context) {
	params := &dto.AdminLoginInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	adminUsername := lib.GetStringConf("base.admin.username")
	adminPassword := lib.GetStringConf("base.admin.password")
	if params.Username != adminUsername || params.Password != adminPassword {
		middleware.ResponseError(c, middleware.InternalErrorCode, errors.New("用户名或密码错误"))
		return
	}

	session := sessions.Default(c)
	session.Set("user", params.Username)
	if err := session.Save(); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	middleware.ResponseSuccess(c, "登录成功")
}

// AdminLoginOut godoc
// @Summary 管理员退出登录
// @Description 管理员退出登录
// @Tags 管理员接口
// @ID /admin_login/logout
// @Accept  json
// @Produce  json
// @Success 200 {object} middleware.Response{data=string} "success"
// @Router /admin_login/logout [get]
func (admin *AdminLoginController) AdminLoginOut(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	if err := session.Save(); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}
	middleware.ResponseSuccess(c, "退出登录成功")
}
