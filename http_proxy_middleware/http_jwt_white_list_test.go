package http_proxy_middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
)

func TestJwtWhiteList(t *testing.T) {
	cases := []struct {
		name       string
		app        *dao.AppInfo
		remoteAddr string
		headers    map[string]string
		want       bool
	}{
		{"未配置白名单放行", &dao.AppInfo{AppID: "a"}, "1.1.1.1:1234", nil, true},
		{"白名单内放行", &dao.AppInfo{AppID: "a", WhiteIps: "10.0.0.1,1.1.1.1"}, "1.1.1.1:1234", nil, true},
		{"白名单支持换行分隔", &dao.AppInfo{AppID: "a", WhiteIps: "10.0.0.1\n1.1.1.1"}, "1.1.1.1:1234", nil, true},
		{"白名单外拒绝", &dao.AppInfo{AppID: "a", WhiteIps: "10.0.0.1"}, "1.1.1.1:1234", nil, false},
		{"伪造X-Forwarded-For无法绕过", &dao.AppInfo{AppID: "a", WhiteIps: "10.0.0.1"}, "1.1.1.1:1234",
			map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-Ip": "10.0.0.1"}, false},
	}
	for _, tc := range cases {
		app := tc.app
		r := newTestRouter(newTestService("svc", &dao.ServiceAccessControl{}), func(c *gin.Context) {
			c.Set("app", app)
			c.Next()
		}, HTTPJwtWhiteListMiddleware())
		if got := doRequest(r, tc.remoteAddr, tc.headers); got != tc.want {
			t.Errorf("%s: passed=%v, want %v", tc.name, got, tc.want)
		}
	}

	// 未识别出租户时不校验
	r := newTestRouter(newTestService("svc", &dao.ServiceAccessControl{}), HTTPJwtWhiteListMiddleware())
	if !doRequest(r, "1.1.1.1:1234", nil) {
		t.Error("request without app should pass")
	}
}
