package integration_test

import (
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/winezer0/ipinfo/internal/testassets"
	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/asninfo/asncsv"
	"github.com/winezer0/ipinfo/pkg/asninfo/asnmmdb"
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/multiloc"
	"github.com/winezer0/ipinfo/pkg/queryip"
)

func TestLocationConsistency(t *testing.T) {
	for _, file := range []string{"qqwry.dat", "zxipv6wry.db", "qqwry.ipdb", "ip2region_v4.xdb", "ip2region_v6.xdb", "dbip-city-ipv4.mmdb", "dbip-city-ipv6.mmdb", "geolite2-city-ipv4.mmdb", "geolite2-city-ipv6.mmdb"} {
		t.Run(file, func(t *testing.T) {
			path := testassets.Path(t, file)
			engine, err := multiloc.InitDBEngines(&multiloc.IPDbConfig{IpLocateDbs: []string{path}, XDBPoolSize: 4})
			if err != nil {
				t.Fatal(err)
			}
			defer engine.Close()
			var db iplocate.IPInfo
			for _, item := range engine.Engines {
				db = item
			}
			ips := []string{"8.8.8.8", "1.1.1.1"}
			if strings.Contains(file, "ipv6") || strings.Contains(file, "_v6") {
				ips = []string{"2001:4860:4860::8888", "2606:4700:4700::1111"}
			}
			queries := append(append([]string{}, ips...), "invalid", ips[0])
			batch := db.BatchFindFull(queries)
			if len(batch) != 3 {
				t.Fatalf("deduplicated batch size: %d", len(batch))
			}
			for _, ip := range ips {
				single := db.FindFull(ip)
				// The pinned GeoLite city snapshots omit some Cloudflare addresses.
				// Google DNS is a known hit in every fixture; all queries must agree.
				if ip == ips[0] && (!single.HasData() || single.Country == "") {
					t.Errorf("expected actual location for %s: %+v", ip, single)
				}
				if !reflect.DeepEqual(single, batch[ip]) {
					t.Errorf("single/batch mismatch for %s: single=%+v batch=%+v", ip, single, batch[ip])
				}
			}
			if batch["invalid"].HasData() {
				t.Fatal("invalid address hit")
			}
		})
	}
}

func TestASNConsistency(t *testing.T) {
	for _, file := range []string{"dbip-asn-ipv4.csv", "dbip-asn-ipv6.csv", "dbip-asn-ipv4.mmdb", "dbip-asn-ipv6.mmdb", "dbip-asn.mmdb", "geolite2-asn-ipv4.mmdb", "geolite2-asn-ipv6.mmdb", "geolite2-asn.mmdb"} {
		t.Run(file, func(t *testing.T) {
			path := testassets.Path(t, file)
			var db asninfo.ASNQuerier
			if strings.HasSuffix(file, ".csv") {
				db = asncsv.NewASNCsvQuerier(path)
			} else {
				var err error
				db, err = asnmmdb.NewMMDBManager(&asnmmdb.MMDBConfig{AsnIpvxDb: path, MaxConcurrentQueries: 4})
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Init(); err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			ip := "8.8.8.8"
			if strings.Contains(file, "ipv6") {
				ip = "2001:4860:4860::8888"
			}
			got := db.FindASN(ip)
			if got == nil || !got.FoundASN || got.OrganisationNumber != 15169 || got.OrganisationName == "" {
				t.Fatalf("expected Google ASN for %s: %+v", ip, got)
			}
			batch := db.BatchFindASN([]string{ip, "invalid"})
			if !reflect.DeepEqual(got, batch[ip]) {
				t.Fatalf("ASN batch mismatch: %+v", batch[ip])
			}
			if bad := batch["invalid"]; bad != nil && bad.FoundASN {
				t.Fatal("invalid address hit")
			}
			ranges, err := db.ASNToIPRanges(15169)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, network := range ranges {
				if network.Contains(net.ParseIP(ip)) {
					found = true
				}
			}
			if !found {
				t.Fatal("reverse ranges omit the queried IP")
			}
		})
	}
}

func TestCombinedQueryAssertions(t *testing.T) {
	cfg := &queryip.IPDbConfig{AsnIpvxDb: testassets.Path(t, "geolite2-asn.mmdb"), Ipv4LocateDb: testassets.Path(t, "ip2region_v4.xdb"), Ipv6LocateDb: testassets.Path(t, "ip2region_v6.xdb"), XDBPoolSize: 4}
	db, err := queryip.InitDBEngines(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, ip := range []string{"8.8.8.8", "2001:4860:4860::8888"} {
		got := db.QueryIP(ip)
		if got.IP != ip || !got.IPLocate.HasData() || got.ASNInfo == nil || !got.ASNInfo.FoundASN || got.ASNInfo.OrganisationNumber != 15169 {
			t.Fatalf("incomplete combined query: %+v", got)
		}
	}
}
