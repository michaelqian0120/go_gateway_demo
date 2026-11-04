package controller

import (
	"fmt"
	"net"
	"strconv"

	"golang_common/lib"
)

// gatewayReservedPortConfs 网关自身占用的端口，TCP服务不能使用，否则代理启动后监听必然失败
var gatewayReservedPortConfs = []string{
	"proxy.http.addr",
	"proxy.https.addr",
	"base.http.addr",
}

func checkTCPPortReserved(port int) error {
	for _, key := range gatewayReservedPortConfs {
		addr := lib.GetStringConf(key)
		if addr == "" {
			continue
		}
		_, p, err := net.SplitHostPort(addr)
		if err != nil {
			continue
		}
		if reserved, err := strconv.Atoi(p); err == nil && reserved == port {
			return fmt.Errorf("端口 %d 已被网关自身使用（%s）", port, key)
		}
	}
	return nil
}
