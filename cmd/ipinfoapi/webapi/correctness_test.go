package webapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/queryip"
)

func TestTrustedProxyBoundary(t *testing.T) {
	for _, tt := range []struct {
		name, remote, xff, real, want string
		trusted                       []string
	}{
		{"direct spoof", "1.1.1.1:123", "8.8.8.8", "9.9.9.9", "1.1.1.1", nil},
		{"trusted chain", "10.0.0.1:123", "8.8.8.8, 10.0.0.2", "", "8.8.8.8", []string{"10.0.0.0/8"}},
		{"forged prefix", "10.0.0.1:123", "9.9.9.9, 8.8.8.8, 10.0.0.2", "", "8.8.8.8", []string{"10.0.0.0/8"}},
		{"invalid chain", "10.0.0.1:123", "bad, 10.0.0.2", "9.9.9.9", "10.0.0.1", []string{"10.0.0.0/8"}},
		{"real header", "10.0.0.1:123", "", "8.8.8.8", "8.8.8.8", []string{"10.0.0.0/8"}},
		{"ipv6", "[fd00::1]:123", "2001:db8::1", "", "2001:db8::1", []string{"fd00::/8"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remote
			r.Header.Set("X-Forwarded-For", tt.xff)
			r.Header.Set("X-Real-IP", tt.real)
			TrustedProxyMiddleware(tt.trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := GetClientIP(r); got != tt.want {
					t.Fatalf("got %q want %q", got, tt.want)
				}
				info := GetClientIpList(r)
				if tt.name == "forged prefix" && strings.Contains(strings.Join(info.XForwardedFor, ","), "9.9.9.9") {
					t.Fatal("forged prefix exposed as validated client IP")
				}
				if tt.name == "direct spoof" && (len(info.XForwardedFor) > 0 || info.XRealIP != "") {
					t.Fatal("untrusted headers accepted")
				}
			})).ServeHTTP(httptest.NewRecorder(), r)
		})
	}
}

type emptyLocation struct{ iplocate.IPInfo }

func (emptyLocation) FindFull(ip string) *iplocate.IPLocate {
	return &iplocate.IPLocate{IP: ip, Version: 4}
}

func TestBatchMissesAndLimits(t *testing.T) {
	h := NewAPIHandler(&queryip.DBEngine{IPv4Engine: emptyLocation{}})
	items, success, failed, err := h.batchQueryIPs(context.Background(), []string{"1.1.1.1", "bad"})
	if err != nil || success != 0 || failed != 2 || len(items) != 2 || items[0].Error == "" {
		t.Fatalf("success=%d failed=%d items=%v err=%v", success, failed, items, err)
	}
	asn := &asnCounter{}
	h.engine.AsnIPv4Engine = asn
	_, success, failed, err = h.batchQueryIPs(context.Background(), []string{"1.1.1.1"})
	if err != nil || success != 1 || failed != 0 {
		t.Fatalf("ASN hit: %d %d %v", success, failed, err)
	}
	h.maxBatchSize = 2
	w := httptest.NewRecorder()
	h.HandleFull(w, httptest.NewRequest(http.MethodGet, "/full?q=1.1.1.1,2.2.2.2,3.3.3.3", nil))
	if w.Code != http.StatusBadRequest || asn.calls != 1 {
		t.Fatalf("limit: status=%d calls=%d", w.Code, asn.calls)
	}
	w = httptest.NewRecorder()
	h.engine.AsnIPv4Engine = nil
	h.HandleFull(w, httptest.NewRequest(http.MethodGet, "/full?q=1.1.1.1", nil))
	var response struct {
		Data FullData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Success != 0 || response.Data.Failed != 1 {
		t.Fatalf("response: %+v", response)
	}
}

type cancelASN struct {
	asninfo.ASNQuerier
	cancel context.CancelFunc
	calls  int
}

func (a *cancelASN) FindASN(ip string) *asninfo.ASNInfo {
	a.calls++
	a.cancel()
	return &asninfo.ASNInfo{IP: ip}
}

func TestBatchCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	asn := &cancelASN{cancel: cancel}
	h := NewAPIHandler(&queryip.DBEngine{AsnIPv4Engine: asn})
	_, _, _, err := h.batchQueryIPs(ctx, []string{"1.1.1.1", "2.2.2.2"})
	if err != context.Canceled || asn.calls != 1 {
		t.Fatalf("calls=%d err=%v", asn.calls, err)
	}
}
