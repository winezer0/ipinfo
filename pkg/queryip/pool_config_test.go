package queryip

import (
	"github.com/winezer0/ipinfo/internal/testxdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/ip2region"
	"testing"
)

func TestXDBPoolConfig(t *testing.T) {
	for _, size := range []int{0, 1, 4} {
		cfg := &IPDbConfig{Ipv4LocateDb: testxdb.Write(t, 4), Ipv6LocateDb: testxdb.Write(t, 6), XDBPoolSize: size}
		db, err := InitDBEngines(cfg)
		if err != nil {
			t.Fatal(err)
		}
		want := size
		if want == 0 {
			want = 1
		}
		if db.IPv4Engine.(*ip2region.IP2Region).PoolSize() != want || db.IPv6Engine.(*ip2region.IP2Region).PoolSize() != want {
			t.Error("pool config not propagated")
		}
		db.Close()
	}
	for _, size := range []int{-1, 65} {
		if db, err := InitDBEngines(&IPDbConfig{XDBPoolSize: size}); err == nil {
			db.Close()
			t.Fatal("invalid size accepted")
		}
	}
}
