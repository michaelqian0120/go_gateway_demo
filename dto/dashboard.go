package dto

type PanelGroupDataOutput struct {
	ServiceNum      int64 `json:"service_num"`
	TodayRequestNum int64 `json:"today_request_num"`
	CurrentQps      int64 `json:"current_qps"`
	AppNum          int64 `json:"app_num"`
}

type ServiceTypeRatioItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type DashFlowStatOutput struct {
	Today            []int64                `json:"today_list"`
	Yesterday        []int64                `json:"yesterday_list"`
	ServiceTypeRatio []ServiceTypeRatioItem `json:"service_type_ratio"`
}
