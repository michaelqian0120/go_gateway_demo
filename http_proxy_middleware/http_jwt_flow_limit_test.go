package http_proxy_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
)

func newAppTestRouter(app *dao.AppInfo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if app != nil {
			c.Set("app", app)
		}
		c.Next()
	}, HTTPJwtFlowLimitMiddleware())
	r.GET("/*any", func(c *gin.Context) {
		// 模拟流量统计中间件，放行的请求计入租户当天请求量
		if app != nil {
			dao.FlowCounterHandler.AppCounter(app.AppID).Increase()
		}
		c.String(http.StatusOK, "passed")
	})
	return r
}

func countPassed(r *gin.Engine, times int) int {
	passed := 0
	for i := 0; i < times; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/abc", nil))
		if w.Body.String() == "passed" {
			passed++
		}
	}
	return passed
}

func TestJwtFlowLimit(t *testing.T) {
	cases := []struct {
		name  string
		app   *dao.AppInfo
		times int
		want  int
	}{
		{"未识别租户不限流", nil, 20, 20},
		{"qps和qpd为0不限流", &dao.AppInfo{AppID: "test_app_unlimited"}, 20, 20},
		{"qps限流", &dao.AppInfo{AppID: "test_app_qps", Qps: 5}, 20, 5},
		{"qpd限流", &dao.AppInfo{AppID: "test_app_qpd", Qpd: 3}, 20, 3},
		{"qps和qpd同时配置取更严格的", &dao.AppInfo{AppID: "test_app_both", Qps: 10, Qpd: 4}, 20, 4},
	}
	for _, tc := range cases {
		if got := countPassed(newAppTestRouter(tc.app), tc.times); got != tc.want {
			t.Errorf("%s: passed %d, want %d", tc.name, got, tc.want)
		}
	}
}
