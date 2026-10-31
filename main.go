package main

import (
	"flag"
	"fmt"
	"go_gateway_demo/dao"
	"go_gateway_demo/http_proxy_router"
	"go_gateway_demo/router"
	"golang_common/lib"
	"os"
	"os/signal"
	"syscall"
)

var (
	endpoint = flag.String("endpoint", "", "input endpoint dashboard or server")
)

func main() {
	// -config 由 lib.InitModule 定义并解析，这里不能重复定义
	lib.InitModule("", []string{"base", "mysql", "redis"})
	defer lib.Destroy()
	if *endpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *endpoint == "dashboard" {
		router.HttpServerRun()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		router.HttpServerStop()
	} else {
		if err := dao.ServiceManagerHandler.LoadOnce(); err != nil {
			fmt.Println("load service fail:", err)
			os.Exit(1)
		}
		http_proxy_router.HttpServerRun()

		fmt.Println("start server")

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		http_proxy_router.HttpServerStop()
	}

}
