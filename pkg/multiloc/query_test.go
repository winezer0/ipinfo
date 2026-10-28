package multiloc

import (
	"testing"

	"github.com/winezer0/ipinfo/internal/testxdb"
)

func TestQueryIPInfoIPv6Only(t *testing.T) {
	path := testxdb.Write(t, 6)
	engine, err := InitDBEngines(&IPDbConfig{IpLocateDbs: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { engine.Close() })
	ips := []string{"2001:db8::1", "2001:db8::1"}
	for _, classify := range []bool{false, true} {
		var info *IPDbInfo
		if classify {
			info, err = engine.QueryIPInfoByIPs(ips)
		} else {
			info, err = engine.QueryIPInfo(nil, ips)
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(info.IPv4Results) != 0 || len(info.IPv6Results) != len(ips) {
			t.Fatalf("classify=%v: unexpected result counts: %+v", classify, info)
		}
		for i, result := range info.IPv6Results {
			loc := result.IPLocateResult[Source(path)]
			if result.IP != ips[i] || loc == nil || loc.Location != "first|province|city|isp" {
				t.Fatalf("classify=%v: result[%d] = %+v, location=%+v", classify, i, result, loc)
			}
		}
	}
}
