package http_proxy_router

import (
	"crypto/subtle"
	"errors"
	"time"

	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
	"go_gateway_demo/public"
	"golang_common/lib"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type TokensOutput struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// OAuthTokens 租户用 HTTP Basic 认证（app_id:secret）换取访问token
// POST /oauth/tokens  Authorization: Basic base64(app_id:secret)
func OAuthTokens(c *gin.Context) {
	appID, secret, ok := c.Request.BasicAuth()
	if !ok || appID == "" || secret == "" {
		middleware.ResponseError(c, 2001, errors.New("need basic auth: app_id:secret"))
		return
	}

	// app_id 不存在和 secret 错误返回同样的提示，避免被用来探测有哪些租户
	app, found := dao.AppManagerHandler.GetApp(appID)
	if !found || subtle.ConstantTimeCompare([]byte(app.Secret), []byte(secret)) != 1 {
		middleware.ResponseError(c, 2002, errors.New("app_id or secret error"))
		return
	}

	expireSeconds := lib.GetIntConf("proxy.jwt.expire_seconds")
	if expireSeconds <= 0 {
		expireSeconds = 7200
	}
	now := time.Now()
	token, err := public.JwtEncode(jwt.StandardClaims{
		Issuer:    app.AppID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
	})
	if err != nil {
		middleware.ResponseError(c, 2003, err)
		return
	}
	middleware.ResponseSuccess(c, &TokensOutput{
		AccessToken: token,
		ExpiresIn:   expireSeconds,
		TokenType:   "Bearer",
		Scope:       "read_write",
	})
}
