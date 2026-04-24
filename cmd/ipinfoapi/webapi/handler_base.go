package webapi

import "github.com/winezer0/ipinfo/pkg/queryip"

// APIHandler API处理器结构体
type APIHandler struct {
	engine *queryip.DBEngine
}

// NewAPIHandler 创建API处理器实例
func NewAPIHandler(engine *queryip.DBEngine) *APIHandler {
	return &APIHandler{engine: engine}
}
