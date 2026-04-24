package webapi

import (
	"encoding/json"
	"net/http"
)

// 错误码常量
const (
	CodeOK           = 0
	CodeInvalidParam = 1001
	CodeUnauthorized = 1002
	CodeQueryFailed  = 2001
	CodeInternalErr  = 5000
)

// APIResponse 统一API响应结构体
type APIResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Code:    CodeOK,
		Data:    data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(code int, message string) APIResponse {
	return APIResponse{
		Success: false,
		Code:    code,
		Error:   message,
	}
}

// WriteJSON 写入JSON响应
func WriteJSON(w http.ResponseWriter, statusCode int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// WriteSuccess 写入成功响应
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, SuccessResponse(data))
}

// WriteError 写入错误响应
func WriteError(w http.ResponseWriter, statusCode int, code int, message string) {
	WriteJSON(w, statusCode, ErrorResponse(code, message))
}
