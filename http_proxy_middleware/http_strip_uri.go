package http_proxy_middleware

import (
	"errors"
	"strings"

	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"

	"github.com/gin-gonic/gin"
)

// HTTPStripUriMiddleware 前缀匹配的服务开启 need_strip_uri 后，去掉路径里的服务前缀
// 例如规则 /test_strip_uri： /test_strip_uri/abc -> /abc
func HTTPStripUriMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		if service.HTTPRule.RuleType == dao.HTTPRuleTypePrefixURL && service.HTTPRule.NeedStripUri == 1 {
			prefix := service.HTTPRule.Rule
			c.Request.URL.Path = ensureLeadingSlash(strings.TrimPrefix(c.Request.URL.Path, prefix))
			if c.Request.URL.RawPath != "" {
				c.Request.URL.RawPath = ensureLeadingSlash(strings.TrimPrefix(c.Request.URL.RawPath, prefix))
			}
		}
		c.Next()
	}
}

func ensureLeadingSlash(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
