package http_proxy_middleware

import (
	"errors"
	"strings"

	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
	"go_gateway_demo/public"

	"github.com/gin-gonic/gin"
)

// HTTPJwtAuthTokenMiddleware 租户认证：校验 Authorization: Bearer <token>，通过后把租户放进上下文 "app"
// open_auth=1 的服务必须带有效token；open_auth=0 的服务带了有效token也会识别租户，用于租户流量统计
// 需要排在流量统计之前
func HTTPJwtAuthTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		app, authErr := authApp(c)
		if app != nil {
			c.Set("app", app)
			// token只给网关用，不转发给下游
			c.Request.Header.Del("Authorization")
		}
		if service.AccessControl.OpenAuth == 1 && app == nil {
			middleware.ResponseError(c, 2002, authErr)
			c.Abort()
			return
		}
		c.Next()
	}
}

// authApp 返回认证通过的租户；未通过时返回原因
func authApp(c *gin.Context) (*dao.AppInfo, error) {
	token, ok := bearerToken(c.GetHeader("Authorization"))
	if !ok {
		return nil, errors.New("missing token, need header Authorization: Bearer <token>")
	}
	claims, err := public.JwtDecode(token)
	if err != nil {
		return nil, errors.New("invalid token: " + err.Error())
	}
	app, found := dao.AppManagerHandler.GetApp(claims.Issuer)
	if !found {
		// 租户在token签发后被删除
		return nil, errors.New("app not found: " + claims.Issuer)
	}
	return app, nil
}

func bearerToken(header string) (string, bool) {
	const prefix = "bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}
