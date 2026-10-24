package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/middleware"
	"golang_common/lib"
)

type ServiceController struct {
}

func ServiceRegister(group *gin.RouterGroup) {
	service := &ServiceController{}
	group.GET("/service_list", middleware.SessionAuthMiddleware(), service.ServiceList)
}

// ServiceList godoc
// @Summary 服务列表
// @Description 服务列表
// @Tags 服务管理
// @ID /service/service_list
// @Accept  json
// @Produce  json
// @Param info query string false "关键词"
// @Param page_no query int true "页数"
// @Param page_size query int true "每页条数"
// @Success 200 {object} middleware.Response{data=dto.ServiceListOutput} "success"
// @Router /service/service_list [get]
func (service *ServiceController) ServiceList(c *gin.Context) {
	params := &dto.ServiceListInput{}
	if err := params.BindingValidParams(c); err != nil {
		middleware.ResponseError(c, middleware.ValidErrorCode, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	serviceInfo := &dao.ServiceInfo{}
	list, total, err := serviceInfo.PageList(tx, params.Info, params.PageNo, params.PageSize)
	if err != nil {
		middleware.ResponseError(c, middleware.InternalErrorCode, err)
		return
	}

	outList := []dto.ServiceListItemOutput{}
	for _, item := range list {
		serviceAddr, err := service.getServiceAddr(tx, item)
		if err != nil {
			middleware.ResponseError(c, middleware.InternalErrorCode, err)
			return
		}

		loadBalance := &dao.ServiceLoadBalance{}
		loadBalance, err = loadBalance.Find(tx, item.Id)
		if err != nil {
			middleware.ResponseError(c, middleware.InternalErrorCode, err)
			return
		}

		outList = append(outList, dto.ServiceListItemOutput{
			Id:          item.Id,
			ServiceName: item.ServiceName,
			ServiceDesc: item.ServiceDesc,
			ServiceType: service.loadTypeText(item.LoadType),
			ServiceAddr: serviceAddr,
			Qps:         0,
			Qpd:         0,
			TotalNode:   loadBalance.NodeCount(),
		})
	}

	middleware.ResponseSuccess(c, &dto.ServiceListOutput{
		Total: total,
		List:  outList,
	})
}

func (service *ServiceController) loadTypeText(loadType int) string {
	switch loadType {
	case dao.LoadTypeHTTP:
		return "HTTP"
	case dao.LoadTypeTCP:
		return "TCP"
	case dao.LoadTypeGRPC:
		return "GRPC"
	default:
		return "未知"
	}
}

func (service *ServiceController) getServiceAddr(tx *gorm.DB, item dao.ServiceInfo) (string, error) {
	clusterIp := lib.GetStringConf("base.cluster.ip")
	httpServerPort := lib.GetIntConf("base.cluster.http_server_port")

	switch item.LoadType {
	case dao.LoadTypeHTTP:
		httpRule := &dao.ServiceHTTPRule{}
		httpRule, err := httpRule.Find(tx, item.Id)
		if err != nil {
			return "", err
		}
		if httpRule.RuleType == 1 {
			return httpRule.Rule, nil
		}
		return fmt.Sprintf("%s:%d%s", clusterIp, httpServerPort, httpRule.Rule), nil
	case dao.LoadTypeTCP:
		tcpRule := &dao.ServiceTCPRule{}
		tcpRule, err := tcpRule.Find(tx, item.Id)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s:%d", clusterIp, tcpRule.Port), nil
	case dao.LoadTypeGRPC:
		grpcRule := &dao.ServiceGRPCRule{}
		grpcRule, err := grpcRule.Find(tx, item.Id)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s:%d", clusterIp, grpcRule.Port), nil
	default:
		return "", nil
	}
}
