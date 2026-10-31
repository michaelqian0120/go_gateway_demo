package http_proxy_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
)

func newTestRouter(service *dao.ServiceDetail, handlers ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("service", service)
		c.Next()
	})
	r.Use(handlers...)
	r.GET("/*any", func(c *gin.Context) {
		c.String(http.StatusOK, "passed")
	})
	return r
}

func newTestService(name string, ac *dao.ServiceAccessControl) *dao.ServiceDetail {
	return &dao.ServiceDetail{
		Info:          &dao.ServiceInfo{ServiceName: name},
		HTTPRule:      &dao.ServiceHTTPRule{},
		AccessControl: ac,
		LoadBalance:   &dao.ServiceLoadBalance{},
	}
}

// doRequest 返回下游是否收到请求，remoteAddr 模拟TCP对端地址
func doRequest(r *gin.Engine, remoteAddr string, headers map[string]string) bool {
	req := httptest.NewRequest(http.MethodGet, "/abc", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Body.String() == "passed"
}

func TestIPAuth(t *testing.T) {
	cases := []struct {
		name       string
		ac         *dao.ServiceAccessControl
		remoteAddr string
		headers    map[string]string
		want       bool
	}{
		{"未开启权限时忽略名单", &dao.ServiceAccessControl{OpenAuth: 0, WhiteList: "10.0.0.1", BlackList: "1.1.1.1"}, "1.1.1.1:1234", nil, true},
		{"白名单内放行", &dao.ServiceAccessControl{OpenAuth: 1, WhiteList: "10.0.0.1,1.1.1.1"}, "1.1.1.1:1234", nil, true},
		{"白名单外拒绝", &dao.ServiceAccessControl{OpenAuth: 1, WhiteList: "10.0.0.1"}, "1.1.1.1:1234", nil, false},
		{"白名单支持换行分隔", &dao.ServiceAccessControl{OpenAuth: 1, WhiteList: "10.0.0.1\n1.1.1.1"}, "1.1.1.1:1234", nil, true},
		{"伪造X-Forwarded-For无法绕过白名单", &dao.ServiceAccessControl{OpenAuth: 1, WhiteList: "10.0.0.1"}, "1.1.1.1:1234",
			map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-Ip": "10.0.0.1"}, false},
		{"黑名单内拒绝", &dao.ServiceAccessControl{OpenAuth: 1, BlackList: "1.1.1.1"}, "1.1.1.1:1234", nil, false},
		{"黑名单外放行", &dao.ServiceAccessControl{OpenAuth: 1, BlackList: "2.2.2.2"}, "1.1.1.1:1234", nil, true},
		{"配置了白名单时以白名单为准", &dao.ServiceAccessControl{OpenAuth: 1, WhiteList: "1.1.1.1", BlackList: "1.1.1.1"}, "1.1.1.1:1234", nil, true},
		{"IPv6地址", &dao.ServiceAccessControl{OpenAuth: 1, WhiteList: "::1"}, "[::1]:1234", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(newTestService("ip_auth", tc.ac), HTTPWhiteListMiddleware(), HTTPBlackListMiddleware())
			if got := doRequest(r, tc.remoteAddr, tc.headers); got != tc.want {
				t.Errorf("passed=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestServiceFlowLimit(t *testing.T) {
	service := newTestService("limit_service", &dao.ServiceAccessControl{ServiceFlowLimit: 3})
	r := newTestRouter(service, HTTPFlowLimitMiddleware())

	passed := 0
	for i := 0; i < 10; i++ {
		// 不同IP共享服务级配额
		if doRequest(r, "1.1.1."+string(rune('0'+i))+":1234", nil) {
			passed++
		}
	}
	if passed != 3 {
		t.Errorf("passed=%d, want 3", passed)
	}
}

func TestClientIPFlowLimit(t *testing.T) {
	service := newTestService("limit_clientip", &dao.ServiceAccessControl{ClientIPFlowLimit: 2})
	r := newTestRouter(service, HTTPFlowLimitMiddleware())

	countPassed := func(remoteAddr string, n int) int {
		passed := 0
		for i := 0; i < n; i++ {
			if doRequest(r, remoteAddr, nil) {
				passed++
			}
		}
		return passed
	}
	if got := countPassed("1.1.1.1:1234", 5); got != 2 {
		t.Errorf("ip1 passed=%d, want 2", got)
	}
	// 每个IP独立计数，换端口也算同一IP
	if got := countPassed("2.2.2.2:5678", 5); got != 2 {
		t.Errorf("ip2 passed=%d, want 2", got)
	}
	if got := countPassed("1.1.1.1:9999", 1); got != 0 {
		t.Errorf("ip1 new port passed=%d, want 0", got)
	}
}

func TestNoFlowLimit(t *testing.T) {
	r := newTestRouter(newTestService("no_limit", &dao.ServiceAccessControl{}), HTTPFlowLimitMiddleware())
	for i := 0; i < 50; i++ {
		if !doRequest(r, "1.1.1.1:1234", nil) {
			t.Fatalf("request %d blocked, want no limit", i)
		}
	}
}
