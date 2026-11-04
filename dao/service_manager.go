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
	TCPRule       *ServiceTCPRule       `json:"tcp_rule"`
	AccessControl *ServiceAccessControl `json:"access_control"`
	LoadBalance   *ServiceLoadBalance   `json:"load_balance"`
}

// ServiceManager 内存中的服务配置
// ServiceMap 按服务名包含所有已加载的服务；ServiceSlice 只有HTTP服务，供 HTTPAccessMode 匹配；
// TCPServiceSlice 只有TCP服务，供TCP代理按端口监听
type ServiceManager struct {
	ServiceMap      map[string]*ServiceDetail
	ServiceSlice    []*ServiceDetail
	TCPServiceSlice []*ServiceDetail
	Locker          sync.RWMutex
	init            sync.Once
	err             error

	reloadHooks []func()
	hookLocker  sync.Mutex
}

var ServiceManagerHandler = NewServiceManager()

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		ServiceMap:      map[string]*ServiceDetail{},
		ServiceSlice:    []*ServiceDetail{},
		TCPServiceSlice: []*ServiceDetail{},
	}
}

// LoadOnce 启动时把所有HTTP、TCP服务加载到内存
func (s *ServiceManager) LoadOnce() error {
	s.init.Do(func() {
		s.err = s.Reload()
	})
	return s.err
}

// Reload 从数据库重新加载所有服务，整体替换内存中的数据，成功后执行 OnReload 注册的回调
// 加载失败时保留旧数据，正在处理的请求不受影响
func (s *ServiceManager) Reload() error {
	loaded, err := loadServices()
	if err != nil {
		return err
	}
	s.Locker.Lock()
	s.ServiceMap = loaded.serviceMap
	s.ServiceSlice = loaded.httpSlice
	s.TCPServiceSlice = loaded.tcpSlice
	s.Locker.Unlock()

	s.hookLocker.Lock()
	hooks := append([]func(){}, s.reloadHooks...)
	s.hookLocker.Unlock()
	for _, hook := range hooks {
		hook()
	}
	return nil
}

// OnReload 注册服务配置重新加载成功后的回调，如TCP代理按最新服务列表增删端口监听
func (s *ServiceManager) OnReload(hook func()) {
	s.hookLocker.Lock()
	s.reloadHooks = append(s.reloadHooks, hook)
	s.hookLocker.Unlock()
}

// GetTCPServiceList 返回当前TCP服务列表的副本
func (s *ServiceManager) GetTCPServiceList() []*ServiceDetail {
	s.Locker.RLock()
	defer s.Locker.RUnlock()
	return append([]*ServiceDetail{}, s.TCPServiceSlice...)
}

// GetService 按服务名取当前配置，服务已删除时返回 false
func (s *ServiceManager) GetService(serviceName string) (*ServiceDetail, bool) {
	s.Locker.RLock()
	defer s.Locker.RUnlock()
	detail, ok := s.ServiceMap[serviceName]
	return detail, ok
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

type loadedServices struct {
	serviceMap map[string]*ServiceDetail
	httpSlice  []*ServiceDetail
	tcpSlice   []*ServiceDetail
}

// loadServices 加载HTTP、TCP服务，每张表各查一次，避免按服务逐个查询
func loadServices() (*loadedServices, error) {
	tx, err := lib.GetGormPool("default")
	if err != nil {
		return nil, err
	}

	out := &loadedServices{
		serviceMap: map[string]*ServiceDetail{},
		httpSlice:  []*ServiceDetail{},
		tcpSlice:   []*ServiceDetail{},
	}
	infoList := []ServiceInfo{}
	if err := tx.Where("is_delete=0 and load_type in ?", []int{LoadTypeHTTP, LoadTypeTCP}).Order("id desc").Find(&infoList).Error; err != nil {
		return nil, err
	}
	if len(infoList) == 0 {
		return out, nil
	}

	ids := make([]int64, 0, len(infoList))
	for _, info := range infoList {
		ids = append(ids, info.Id)
	}

	httpRules := []ServiceHTTPRule{}
	if err := tx.Where("service_id in ?", ids).Find(&httpRules).Error; err != nil {
		return nil, err
	}
	tcpRules := []ServiceTCPRule{}
	if err := tx.Where("service_id in ?", ids).Find(&tcpRules).Error; err != nil {
		return nil, err
	}
	accessControls := []ServiceAccessControl{}
	if err := tx.Where("service_id in ?", ids).Find(&accessControls).Error; err != nil {
		return nil, err
	}
	loadBalances := []ServiceLoadBalance{}
	if err := tx.Where("service_id in ?", ids).Find(&loadBalances).Error; err != nil {
		return nil, err
	}

	httpRuleMap := map[int64]*ServiceHTTPRule{}
	for i := range httpRules {
		httpRuleMap[httpRules[i].ServiceId] = &httpRules[i]
	}
	tcpRuleMap := map[int64]*ServiceTCPRule{}
	for i := range tcpRules {
		tcpRuleMap[tcpRules[i].ServiceId] = &tcpRules[i]
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
			AccessControl: accessControl,
			LoadBalance:   loadBalance,
		}

		switch info.LoadType {
		case LoadTypeHTTP:
			httpRule, ok := httpRuleMap[info.Id]
			if !ok {
				// 没有接入规则的服务无法匹配请求，跳过
				log.Printf(" [WARN] service %s has no http rule, skipped\n", info.ServiceName)
				continue
			}
			detail.HTTPRule = httpRule
			out.httpSlice = append(out.httpSlice, detail)
		case LoadTypeTCP:
			tcpRule, ok := tcpRuleMap[info.Id]
			if !ok || tcpRule.Port <= 0 {
				log.Printf(" [WARN] service %s has no tcp port, skipped\n", info.ServiceName)
				continue
			}
			detail.TCPRule = tcpRule
			out.tcpSlice = append(out.tcpSlice, detail)
		}
		out.serviceMap[info.ServiceName] = detail
	}
	return out, nil
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
