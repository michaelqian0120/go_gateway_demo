package http_proxy_middleware

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go_gateway_demo/dao"
	"go_gateway_demo/middleware"
)

type urlRewriteRule struct {
	re          *regexp.Regexp
	replacement string
}

// 按 url_rewrite 原始字符串缓存编译好的规则，避免每次请求都编译正则
var urlRewriteCache sync.Map

// HTTPUrlRewriteMiddleware 按 url_rewrite 规则用正则重写路径，在 strip_uri 之后执行
// 规则格式：正则 替换串，例如 ^/aaa/(.*) /bbb/$1，多条用逗号或换行分隔，按顺序依次应用
func HTTPUrlRewriteMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		service := serviceInterface.(*dao.ServiceDetail)

		rules := getUrlRewriteRules(service.HTTPRule.UrlRewrite)
		if len(rules) > 0 {
			path := c.Request.URL.Path
			for _, rule := range rules {
				path = rule.re.ReplaceAllString(path, rule.replacement)
			}
			c.Request.URL.Path = ensureLeadingSlash(path)
			// 重写后 RawPath 与 Path 不再对应，清空让 net/url 重新编码
			c.Request.URL.RawPath = ""
		}
		c.Next()
	}
}

func getUrlRewriteRules(raw string) []urlRewriteRule {
	if raw == "" {
		return nil
	}
	if cached, ok := urlRewriteCache.Load(raw); ok {
		return cached.([]urlRewriteRule)
	}

	rules := []urlRewriteRule{}
	for _, item := range splitRules(raw) {
		items := strings.Fields(item)
		if len(items) != 2 {
			log.Printf(" [WARN] invalid url_rewrite rule:%q\n", item)
			continue
		}
		re, err := regexp.Compile(items[0])
		if err != nil {
			log.Printf(" [WARN] invalid url_rewrite regexp:%q err:%v\n", items[0], err)
			continue
		}
		rules = append(rules, urlRewriteRule{re: re, replacement: items[1]})
	}
	urlRewriteCache.Store(raw, rules)
	return rules
}
