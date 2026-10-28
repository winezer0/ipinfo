package main

import (
	"fmt"
	"testing"

	"github.com/winezer0/ipinfo/internal/testassets"
	"github.com/winezer0/ipinfo/internal/testxdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/ip2region"
)

func TestPoolMeasurement(t *testing.T) {
	for _, version := range []int{4, 6} {
		path := testxdb.Write(t, version)
		for _, batch := range []int{1, 16} {
			got, err := run(path, version, 4, 16, batch, 200)
			if err != nil {
				t.Fatal(err)
			}
			if got.Misses != 0 || got.Requests != 200 || got.IPs != 200*batch || got.Seconds <= 0 || got.P95Microseconds <= 0 || got.P95Microseconds < got.P50Microseconds {
				t.Fatalf("invalid concurrent measurement: %+v", got)
			}
		}
	}
}

// BenchmarkPoolSetup measures allocation costs of a shared index plus independent
// handles; it excludes query results and reports allocations, not retained RSS.
func BenchmarkPoolSetup(b *testing.B) {
	path := testassets.Path(b, "ip2region_v4.xdb")
	for _, size := range []int{1, 4} {
		b.Run(fmt.Sprintf("searchers=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				db, err := ip2region.NewIP2RegionWithPool(4, path, size)
				if err != nil {
					b.Fatal(err)
				}
				db.Close()
			}
		})
	}
}
