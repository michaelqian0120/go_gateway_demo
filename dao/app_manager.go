package dao

import (
	"log"
	"sync"
	"time"

	"golang_common/lib"
)

// AppManager 租户信息加载到内存，供代理认证时按 app_id 查找
type AppManager struct {
	AppMap map[string]*AppInfo
	Locker sync.RWMutex
	init   sync.Once
	err    error
}

var AppManagerHandler = &AppManager{AppMap: map[string]*AppInfo{}}

// LoadOnce 启动时加载所有租户
func (s *AppManager) LoadOnce() error {
	s.init.Do(func() {
		s.err = s.Reload()
	})
	return s.err
}

// Reload 重新加载所有未删除的租户，整体替换内存中的数据；失败时保留旧数据
func (s *AppManager) Reload() error {
	tx, err := lib.GetGormPool("default")
	if err != nil {
		return err
	}
	list := []AppInfo{}
	if err := tx.Where("is_delete=0").Find(&list).Error; err != nil {
		return err
	}
	appMap := make(map[string]*AppInfo, len(list))
	for i := range list {
		appMap[list[i].AppID] = &list[i]
	}
	s.Locker.Lock()
	s.AppMap = appMap
	s.Locker.Unlock()
	return nil
}

// StartAutoReload 定时重新加载租户，后台修改、删除租户后无需重启代理
func (s *AppManager) StartAutoReload(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := s.Reload(); err != nil {
				log.Printf(" [ERROR] app manager reload err:%v\n", err)
			}
		}
	}()
}

// GetApp 按 app_id 查找租户，租户被删除后返回 false
func (s *AppManager) GetApp(appID string) (*AppInfo, bool) {
	s.Locker.RLock()
	defer s.Locker.RUnlock()
	app, ok := s.AppMap[appID]
	return app, ok
}
