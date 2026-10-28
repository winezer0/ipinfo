package multiloc

import (
	"testing"

	"github.com/winezer0/ipinfo/internal/testxdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/ip2region"
)

func TestXDBPoolConfig(t *testing.T) {
	for _, size := range []int{0, 4} {
		db, err := InitDBEngines(&IPDbConfig{IpLocateDbs: []string{testxdb.Write(t, 6)}, XDBPoolSize: size})
		if err != nil {
			t.Fatal(err)
		}
		want := size
		if want == 0 {
			want = 1
		}
		for _, engine := range db.Engines {
			if engine.(*ip2region.IP2Region).PoolSize() != want {
				t.Fatal("pool config not propagated")
			}
		}
		db.Close()
	}
	if _, err := InitDBEngines(&IPDbConfig{XDBPoolSize: 65}); err == nil {
		t.Fatal("invalid size accepted")
	}
}
