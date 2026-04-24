package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/queryip"
)

// getTestDBPath 获取测试数据库文件路径
func getTestDBPath(dbname string) string {
	return "../../../assets/" + dbname
}

// initTestEngine 初始化测试用的数据库引擎
func initTestEngine(t *testing.T) *queryip.DBEngine {
	t.Helper()

	config := &queryip.IPDbConfig{
		AsnIpvxDb:    getTestDBPath("geolite2-asn.mmdb"),
		Ipv4LocateDb: getTestDBPath("qqwry.dat"),
		Ipv6LocateDb: getTestDBPath("zxipv6wry.db"),
	}

	engine, err := queryip.InitDBEngines(config)
	if err != nil {
		t.Skipf("跳过测试，数据库加载失败: %v", err)
	}

	return engine
}

// TestHandlePing 测试健康检查端点
func TestHandlePing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	HandlePing(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !apiResp.Success {
		t.Error("期望响应成功")
	}

	if apiResp.Code != CodeOK {
		t.Errorf("期望错误码 %d，实际 %d", CodeOK, apiResp.Code)
	}
}

// TestHandleASN 测试ASN查询
func TestHandleASN(t *testing.T) {
	engine := initTestEngine(t)
	defer engine.Close()

	handler := NewAPIHandler(engine)

	tests := []struct {
		name       string
		queryParam string
		remoteAddr string
		expectIP   string
	}{
		{
			name:       "带参数查询",
			queryParam: "8.8.8.8",
			remoteAddr: "127.0.0.1:12345",
			expectIP:   "8.8.8.8",
		},
		{
			name:       "不带参数使用客户端IP",
			queryParam: "",
			remoteAddr: "1.1.1.1:12345",
			expectIP:   "1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/asn"
			if tt.queryParam != "" {
				url += "?q=" + tt.queryParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.HandleASN(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
			}

			var apiResp APIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if !apiResp.Success {
				t.Errorf("期望响应成功，实际: %v", apiResp.Error)
			}
		})
	}
}

// TestHandleASN_InvalidIP 测试ASN查询无效IP
func TestHandleASN_InvalidIP(t *testing.T) {
	engine := initTestEngine(t)
	defer engine.Close()

	handler := NewAPIHandler(engine)

	req := httptest.NewRequest(http.MethodGet, "/asn?q=invalid_ip", nil)
	w := httptest.NewRecorder()

	handler.HandleASN(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusBadRequest, resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if apiResp.Success {
		t.Error("期望响应失败")
	}

	if apiResp.Code != CodeInvalidParam {
		t.Errorf("期望错误码 %d，实际 %d", CodeInvalidParam, apiResp.Code)
	}
}

// TestHandleIP 测试IP定位查询
func TestHandleIP(t *testing.T) {
	engine := initTestEngine(t)
	defer engine.Close()

	handler := NewAPIHandler(engine)

	req := httptest.NewRequest(http.MethodGet, "/ip?q=8.8.8.8", nil)
	w := httptest.NewRecorder()

	handler.HandleIP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("期望响应成功，实际: %v", apiResp.Error)
	}
}

// TestHandleAll 测试完整查询
func TestHandleAll(t *testing.T) {
	engine := initTestEngine(t)
	defer engine.Close()

	handler := NewAPIHandler(engine)

	req := httptest.NewRequest(http.MethodGet, "/all?q=8.8.8.8", nil)
	w := httptest.NewRecorder()

	handler.HandleAll(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("期望响应成功，实际: %v", apiResp.Error)
	}
}

// TestHandleFull 测试批量IP查询
func TestHandleFull(t *testing.T) {
	engine := initTestEngine(t)
	defer engine.Close()

	handler := NewAPIHandler(engine)

	tests := []struct {
		name       string
		queryParam string
		expectCode int
	}{
		{
			name:       "单个IP查询",
			queryParam: "8.8.8.8",
			expectCode: http.StatusOK,
		},
		{
			name:       "多个IP查询",
			queryParam: "8.8.8.8,1.1.1.1,114.114.114.114",
			expectCode: http.StatusOK,
		},
		{
			name:       "包含无效IP",
			queryParam: "8.8.8.8,invalid_ip,1.1.1.1",
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/full?q=" + tt.queryParam
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.HandleFull(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectCode {
				t.Errorf("期望状态码 %d，实际 %d", tt.expectCode, resp.StatusCode)
			}

			var apiResp APIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if !apiResp.Success {
				t.Errorf("期望响应成功，实际: %v", apiResp.Error)
			}
		})
	}
}

// TestValidateIP 测试IP校验函数
func TestValidateIP(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOK bool
		wantIP string
	}{
		{"有效IPv4", "8.8.8.8", true, "8.8.8.8"},
		{"有效IPv6", "2001:db8::1", true, "2001:db8::1"},
		{"带空格IPv4", "  8.8.8.8  ", true, "8.8.8.8"},
		{"无效IP", "invalid", false, ""},
		{"空字符串", "", false, ""},
		{"超出范围", "999.999.999.999", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP, gotOK := ValidateIP(tt.input)
			if gotOK != tt.wantOK {
				t.Errorf("ValidateIP(%q) OK = %v，期望 %v", tt.input, gotOK, tt.wantOK)
			}
			if gotIP != tt.wantIP {
				t.Errorf("ValidateIP(%q) IP = %q，期望 %q", tt.input, gotIP, tt.wantIP)
			}
		})
	}
}

// TestGetClientIP 测试获取客户端IP
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		remote  string
		wantIP  string
	}{
		{
			name:    "X-Forwarded-For",
			headers: map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			remote:  "127.0.0.1:12345",
			wantIP:  "1.2.3.4",
		},
		{
			name:    "X-Real-IP",
			headers: map[string]string{"X-Real-IP": "2.3.4.5"},
			remote:  "127.0.0.1:12345",
			wantIP:  "2.3.4.5",
		},
		{
			name:    "RemoteAddr",
			headers: map[string]string{},
			remote:  "3.4.5.6:12345",
			wantIP:  "3.4.5.6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			gotIP := GetClientIP(req)
			if gotIP != tt.wantIP {
				t.Errorf("GetClientIP() = %q，期望 %q", gotIP, tt.wantIP)
			}
		})
	}
}

// TestAuthMiddleware 测试认证中间件
func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		authEnable   bool
		authHeader   string
		tokenParam   string
		path         string
		whiteList    []string
		expectStatus int
	}{
		{
			name:         "认证关闭",
			authEnable:   false,
			authHeader:   "",
			expectStatus: http.StatusOK,
		},
		{
			name:         "认证开启且Authorization头正确",
			authEnable:   true,
			authHeader:   "Bearer test-token",
			expectStatus: http.StatusOK,
		},
		{
			name:         "认证开启且URL参数token正确",
			authEnable:   true,
			tokenParam:   "test-token",
			expectStatus: http.StatusOK,
		},
		{
			name:         "认证开启但缺失所有认证信息",
			authEnable:   true,
			authHeader:   "",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "认证开启但Authorization格式错误",
			authEnable:   true,
			authHeader:   "InvalidFormat",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "认证开启但Authorization头Token错误",
			authEnable:   true,
			authHeader:   "Bearer wrong-token",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "认证开启但URL参数token错误",
			authEnable:   true,
			tokenParam:   "wrong-token",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "认证开启且Authorization优先于URL参数",
			authEnable:   true,
			authHeader:   "Bearer test-token",
			tokenParam:   "wrong-token",
			expectStatus: http.StatusOK,
		},
		{
			name:         "白名单路径跳过认证",
			authEnable:   true,
			path:         "/ping",
			whiteList:    []string{"/ping"},
			expectStatus: http.StatusOK,
		},
		{
			name:         "非白名单路径需要认证",
			authEnable:   true,
			path:         "/ip",
			whiteList:    []string{"/ping"},
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewAuthMiddleware("test-token", tt.authEnable, tt.whiteList...)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.Middleware(nextHandler)

			url := tt.path
			if url == "" {
				url = "/test"
			}
			if tt.tokenParam != "" {
				if strings.Contains(url, "?") {
					url += "&token=" + tt.tokenParam
				} else {
					url += "?token=" + tt.tokenParam
				}
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectStatus {
				t.Errorf("期望状态码 %d，实际 %d", tt.expectStatus, resp.StatusCode)
			}
		})
	}
}

// TestExtractToken 测试Token提取函数
func TestExtractToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		tokenParam string
		wantToken  string
	}{
		{
			name:       "仅Authorization头",
			authHeader: "Bearer header-token",
			tokenParam: "",
			wantToken:  "header-token",
		},
		{
			name:       "仅URL参数",
			authHeader: "",
			tokenParam: "url-token",
			wantToken:  "url-token",
		},
		{
			name:       "Authorization优先",
			authHeader: "Bearer header-token",
			tokenParam: "url-token",
			wantToken:  "header-token",
		},
		{
			name:       "两者都为空",
			authHeader: "",
			tokenParam: "",
			wantToken:  "",
		},
		{
			name:       "Authorization格式错误回退到URL参数",
			authHeader: "InvalidFormat",
			tokenParam: "url-token",
			wantToken:  "url-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.tokenParam != "" {
				url += "?token=" + tt.tokenParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			gotToken := extractToken(req)
			if gotToken != tt.wantToken {
				t.Errorf("extractToken() = %q，期望 %q", gotToken, tt.wantToken)
			}
		})
	}
}

// TestRecoveryMiddleware 测试异常恢复中间件
func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusInternalServerError, resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if apiResp.Success {
		t.Error("期望响应失败")
	}

	if apiResp.Code != CodeInternalErr {
		t.Errorf("期望错误码 %d，实际 %d", CodeInternalErr, apiResp.Code)
	}
}

// TestConvertIPLocate 测试IP定位数据转换
func TestConvertIPLocate(t *testing.T) {
	tests := []struct {
		name    string
		input   *iplocate.IPLocate
		wantNil bool
	}{
		{
			name:    "nil输入",
			input:   nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertIPLocate(tt.input)
			if tt.wantNil && result != nil {
				t.Error("期望返回 nil")
			}
		})
	}
}

// TestConvertASN 测试ASN数据转换
func TestConvertASN(t *testing.T) {
	tests := []struct {
		name    string
		input   *asninfo.ASNInfo
		wantNil bool
	}{
		{
			name:    "nil输入",
			input:   nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertASN(tt.input)
			if tt.wantNil && result != nil {
				t.Error("期望返回 nil")
			}
		})
	}
}

// TestUniqueStrings 测试字符串去重
func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "无重复",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "有重复",
			input: []string{"a", "b", "a", "c", "b"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "空切片",
			input: []string{},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueStrings(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("uniqueStrings() 长度 = %d，期望 %d", len(got), len(tt.want))
			}

			seen := make(map[string]bool)
			for _, s := range got {
				seen[s] = true
			}
			for _, s := range tt.want {
				if !seen[s] {
					t.Errorf("uniqueStrings() 缺少元素 %q", s)
				}
			}
		})
	}
}

// TestResponseFunctions 测试响应函数
func TestResponseFunctions(t *testing.T) {
	t.Run("SuccessResponse", func(t *testing.T) {
		resp := SuccessResponse(map[string]string{"key": "value"})
		if !resp.Success {
			t.Error("期望 Success = true")
		}
		if resp.Code != CodeOK {
			t.Errorf("期望 Code = %d，实际 %d", CodeOK, resp.Code)
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		resp := ErrorResponse(CodeInvalidParam, "test error")
		if resp.Success {
			t.Error("期望 Success = false")
		}
		if resp.Code != CodeInvalidParam {
			t.Errorf("期望 Code = %d，实际 %d", CodeInvalidParam, resp.Code)
		}
		if resp.Error != "test error" {
			t.Errorf("期望 Error = %q，实际 %q", "test error", resp.Error)
		}
	})
}
