package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/winezer0/ipinfo/cmd/ipinfoapi/webapi"
	"github.com/winezer0/ipinfo/internal/benchutil"
)

func TestValidationRejectsFalseSuccess(t *testing.T) {
	for _, endpoint := range []string{"/ip", "/asn", "/all", "/full"} {
		body, _ := json.Marshal(webapi.SuccessResponse(map[string]any{}))
		if validate(body, 200, endpoint, []string{"8.8.8.8"}) == nil {
			t.Errorf("accepted empty %s result", endpoint)
		}
	}
	if validate([]byte(`{"success":true,"code":0,"data":{"ip":"1.1.1.1","found_asn":true,"as_number":13335}}`), 200, "/asn", []string{"8.8.8.8"}) == nil {
		t.Fatal("accepted wrong IP")
	}
}

func TestMeasuredErrorsAreCounted(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("process metrics support Windows/Linux")
	}
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == metricsPath {
			u, err := benchutil.ProcessUsage()
			if err != nil {
				t.Error(err)
			}
			json.NewEncoder(w).Encode(u)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing auth")
		}
		if calls.Add(1) > 32 {
			json.NewEncoder(w).Encode(webapi.ErrorResponse(2001, "injected query failure"))
			return
		}
		json.NewEncoder(w).Encode(webapi.SuccessResponse(webapi.ASNData{IP: r.URL.Query().Get("q"), FoundASN: true, ASNumber: 15169}))
	}))
	defer server.Close()
	result, err := measure(server.URL, "/asn", 4, 20, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors != 20 || result.ErrorRate != 1 || result.FirstError == "" || result.ServerRSSMaxSampledBytes == 0 || result.P95US <= 0 {
		t.Fatalf("invalid measurement: %+v", result)
	}
}
