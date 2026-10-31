package dao

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

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
		s.err = s.Reload()
	})
	return s.err
}

// Reload 从数据库重新加载所有HTTP服务，整体替换内存中的数据
// 加载失败时保留旧数据，正在处理的请求不受影响
func (s *ServiceManager) Reload() error {
	serviceMap, serviceSlice, err := loadHTTPServices()
	if err != nil {
		return err
	}
	s.Locker.Lock()
	s.ServiceMap = serviceMap
	s.ServiceSlice = serviceSlice
	s.Locker.Unlock()
	return nil
}

// StartAutoReload 定时重新加载服务配置，后台修改、删除服务后无需重启代理
func (s *ServiceManager) StartAutoReload(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := s.Reload(); err != nil {
				log.Printf(" [ERROR] service manager reload err:%v\n", err)
			}
		}
	}()
}

// loadHTTPServices 每张表各查一次，避免按服务逐个查询
func loadHTTPServices() (map[string]*ServiceDetail, []*ServiceDetail, error) {
	tx, err := lib.GetGormPool("default")
	if err != nil {
		return nil, nil, err
	}

	infoList := []ServiceInfo{}
	if err := tx.Where("is_delete=0 and load_type=?", LoadTypeHTTP).Order("id desc").Find(&infoList).Error; err != nil {
		return nil, nil, err
	}
	serviceMap := map[string]*ServiceDetail{}
	serviceSlice := []*ServiceDetail{}
	if len(infoList) == 0 {
		return serviceMap, serviceSlice, nil
	}

	ids := make([]int64, 0, len(infoList))
	for _, info := range infoList {
		ids = append(ids, info.Id)
	}

	httpRules := []ServiceHTTPRule{}
	if err := tx.Where("service_id in ?", ids).Find(&httpRules).Error; err != nil {
		return nil, nil, err
	}
	accessControls := []ServiceAccessControl{}
	if err := tx.Where("service_id in ?", ids).Find(&accessControls).Error; err != nil {
		return nil, nil, err
	}
	loadBalances := []ServiceLoadBalance{}
	if err := tx.Where("service_id in ?", ids).Find(&loadBalances).Error; err != nil {
		return nil, nil, err
	}

	httpRuleMap := map[int64]*ServiceHTTPRule{}
	for i := range httpRules {
		httpRuleMap[httpRules[i].ServiceId] = &httpRules[i]
	}
	accessControlMap := map[int64]*ServiceAccessControl{}
	for i := range accessControls {
		accessControlMap[accessControls[i].ServiceId] = &accessControls[i]
	}
	loadBalanceMap := map[int64]*ServiceLoadBalance{}
	for i := range loadBalances {
		loadBalanceMap[loadBalances[i].ServiceId] = &loadBalances[i]
	}

	for i := range infoList {
		info := &infoList[i]
		httpRule, ok := httpRuleMap[info.Id]
		if !ok {
			// 没有接入规则的服务无法匹配请求，跳过
			log.Printf(" [WARN] service %s has no http rule, skipped\n", info.ServiceName)
			continue
		}
		accessControl, ok := accessControlMap[info.Id]
		if !ok {
			accessControl = &ServiceAccessControl{ServiceId: info.Id}
		}
		loadBalance, ok := loadBalanceMap[info.Id]
		if !ok {
			loadBalance = &ServiceLoadBalance{ServiceId: info.Id}
		}
		detail := &ServiceDetail{
			Info:          info,
			HTTPRule:      httpRule,
			AccessControl: accessControl,
			LoadBalance:   loadBalance,
		}
		serviceMap[info.ServiceName] = detail
		serviceSlice = append(serviceSlice, detail)
	}
	return serviceMap, serviceSlice, nil
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
