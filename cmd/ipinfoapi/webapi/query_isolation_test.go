package webapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/queryip"
)

type locationCounter struct {
	iplocate.IPInfo
	calls int
}

func (s *locationCounter) FindFull(ip string) *iplocate.IPLocate {
	s.calls++
	return &iplocate.IPLocate{IP: ip, Location: "test"}
}

type asnCounter struct {
	asninfo.ASNQuerier
	calls int
}

func (s *asnCounter) FindASN(ip string) *asninfo.ASNInfo {
	s.calls++
	return &asninfo.ASNInfo{IP: ip, FoundASN: true, OrganisationNumber: 1}
}

func TestSinglePurposeHandlers(t *testing.T) {
	for _, ip := range []string{"1.1.1.1", "2001:db8::1"} {
		t.Run(ip, func(t *testing.T) {
			loc, asn := &locationCounter{}, &asnCounter{}
			h := NewAPIHandler(&queryip.DBEngine{
				IPv4Engine: loc, IPv6Engine: loc, AsnIPv4Engine: asn, AsnIPv6Engine: asn,
			})
			w := httptest.NewRecorder()
			h.HandleIP(w, httptest.NewRequest(http.MethodGet, "/ip?q="+ip, nil))
			if w.Code != http.StatusOK || loc.calls != 1 || asn.calls != 0 {
				t.Fatalf("IP handler: status=%d location=%d asn=%d", w.Code, loc.calls, asn.calls)
			}
			w = httptest.NewRecorder()
			h.HandleASN(w, httptest.NewRequest(http.MethodGet, "/asn?q="+ip, nil))
			if w.Code != http.StatusOK || loc.calls != 1 || asn.calls != 1 {
				t.Fatalf("ASN handler: status=%d location=%d asn=%d", w.Code, loc.calls, asn.calls)
			}
		})
	}
}
