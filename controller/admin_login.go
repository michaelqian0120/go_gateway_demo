package controller

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"go_gateway_demo/public"
	"golang_common/lib"
)

type AdminLoginController struct {
}

func AdminLoginRegister(group *gin.RouterGroup) {
	admin := &AdminLoginController{}
	group.POST("/login", admin.AdminLogin)
	group.GET("/logout", admin.AdminLoginOut)
	group.GET("/login_info", middleware.SessionAuthMiddleware(), admin.AdminLoginInfo)
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

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	//1. params.Username 取得管理员信息 adminInfo
	adminInfoSearch := &dao.AdminInfo{UserName: params.Username}
	adminInfo, err := adminInfoSearch.Find(tx, adminInfoSearch)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, errors.New("用户名或密码错误"))
		return
	}

	//2. adminInfo.Salt + params.Password sha256 => saltPassword
	saltPassword := public.GenSaltPassword(adminInfo.Salt, params.Password)

	//3. saltPassword == adminInfo.Password
	if saltPassword != adminInfo.Password {
		middleware.ResponseError(c, middleware.InternalErrorCode, errors.New("用户名或密码错误"))
		return
	}

	sessionInfo := &public.AdminSessionInfo{
		Id:        adminInfo.Id,
		UserName:  adminInfo.UserName,
		LoginTime: time.Now().Unix(),
	}
	sessionBts, err := json.Marshal(sessionInfo)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	session := sessions.Default(c)
	session.Set(public.AdminSessionInfoKey, string(sessionBts))
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

// AdminLoginInfo godoc
// @Summary 获取管理员登录信息
// @Description 获取管理员登录信息
// @Tags 管理员接口
// @ID /admin_login/login_info
// @Accept  json
// @Produce  json
// @Success 200 {object} middleware.Response{data=dto.AdminInfoOutput} "success"
// @Router /admin_login/login_info [get]
func (admin *AdminLoginController) AdminLoginInfo(c *gin.Context) {
	session := sessions.Default(c)
	sessionBts, ok := session.Get(public.AdminSessionInfoKey).(string)
	if !ok {
		middleware.ResponseError(c, middleware.InternalErrorCode, errors.New("user not login"))
		return
	}

	sessionInfo := &public.AdminSessionInfo{}
	if err := json.Unmarshal([]byte(sessionBts), sessionInfo); err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	out := &dto.AdminInfoOutput{
		ID:        sessionInfo.Id,
		Name:      sessionInfo.UserName,
		LoginTime: time.Unix(sessionInfo.LoginTime, 0),
		Roles:     []string{"admin"},
	}
	middleware.ResponseSuccess(c, out)
}
