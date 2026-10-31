package dao

import (
	"errors"
	"strings"
	"sync"

	"golang_common/lib"

	"github.com/gin-gonic/gin"
)

const (
	HTTPRuleTypePrefixURL = 0
	HTTPRuleTypeDomain    = 1
)

type ServiceDetail struct {
	Info          *ServiceInfo          `json:"info"`
	HTTPRule      *ServiceHTTPRule      `json:"http_rule"`
	AccessControl *ServiceAccessControl `json:"access_control"`
	LoadBalance   *ServiceLoadBalance   `json:"load_balance"`
}

type ServiceManager struct {
	ServiceMap   map[string]*ServiceDetail
	ServiceSlice []*ServiceDetail
	Locker       sync.RWMutex
	init         sync.Once
	err          error
}

var ServiceManagerHandler = NewServiceManager()

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		ServiceMap:   map[string]*ServiceDetail{},
		ServiceSlice: []*ServiceDetail{},
	}
}

// LoadOnce 启动时把所有HTTP服务加载到内存
func (s *ServiceManager) LoadOnce() error {
	s.init.Do(func() {
		tx, err := lib.GetGormPool("default")
		if err != nil {
			s.err = err
			return
		}

		list := []ServiceInfo{}
		if err := tx.Where("is_delete=0 and load_type=?", LoadTypeHTTP).Order("id desc").Find(&list).Error; err != nil {
			s.err = err
			return
		}

		s.Locker.Lock()
		defer s.Locker.Unlock()
		for i := range list {
			info := &list[i]
			httpRule, err := (&ServiceHTTPRule{}).Find(tx, info.Id)
			if err != nil {
				s.err = err
				return
			}
			accessControl, err := (&ServiceAccessControl{}).Find(tx, info.Id)
			if err != nil {
				s.err = err
				return
			}
			loadBalance, err := (&ServiceLoadBalance{}).Find(tx, info.Id)
			if err != nil {
				s.err = err
				return
			}
			detail := &ServiceDetail{
				Info:          info,
				HTTPRule:      httpRule,
				AccessControl: accessControl,
				LoadBalance:   loadBalance,
			}
			s.ServiceMap[info.ServiceName] = detail
			s.ServiceSlice = append(s.ServiceSlice, detail)
		}
	})
	return s.err
}

// HTTPAccessMode 按接入方式匹配服务：域名完全匹配优先，其次取最长的URL前缀匹配
func (s *ServiceManager) HTTPAccessMode(c *gin.Context) (*ServiceDetail, error) {
	host := c.Request.Host
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	path := c.Request.URL.Path

	s.Locker.RLock()
	defer s.Locker.RUnlock()
	var matched *ServiceDetail
	for _, detail := range s.ServiceSlice {
		switch detail.HTTPRule.RuleType {
		case HTTPRuleTypeDomain:
			if detail.HTTPRule.Rule == host {
				return detail, nil
			}
		case HTTPRuleTypePrefixURL:
			// 避免 /test 抢走 /test_http_service 的请求
			if strings.HasPrefix(path, detail.HTTPRule.Rule) &&
				(matched == nil || len(detail.HTTPRule.Rule) > len(matched.HTTPRule.Rule)) {
				matched = detail
			}
		}
	}
	if matched != nil {
		return matched, nil
	}
	return nil, errors.New("not matched service")
}
