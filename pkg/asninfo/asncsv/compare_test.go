package asncsv

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareIPNumericCompatibility(t *testing.T) {
	ips := []net.IP{nil, net.ParseIP("0.0.0.0"), net.IP{1, 2, 3, 4}, net.ParseIP("255.255.255.255"), net.ParseIP("::"), net.ParseIP("::1"), net.ParseIP("::ffff:1.2.3.4"), net.ParseIP("2001:db8::1"), net.ParseIP("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")}
	for _, a := range ips {
		for _, b := range ips {
			want := ipToBigInt(a).Cmp(ipToBigInt(b))
			got := compareIP(a, b)
			if (got < 0) != (want < 0) || (got > 0) != (want > 0) {
				t.Fatalf("compare %v %v: got %d want %d", a, b, got, want)
			}
		}
	}
}

func TestCSVLookupBoundaries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asn.csv")
	data := "start_ip,end_ip,asn,org\n1.0.0.0,1.0.0.3,1,first\n1.0.0.8,1.0.0.9,2,second\n2001:db8::,2001:db8::3,3,v6\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	db := NewASNCsvQuerier(path)
	if err := db.Init(); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for ip, want := range map[string]uint64{"0.255.255.255": 0, "1.0.0.0": 1, "1.0.0.3": 1, "1.0.0.4": 0, "1.0.0.8": 2, "1.0.0.9": 2, "2001:db8::": 3, "2001:db8::3": 3, "2001:db8::4": 0} {
		got := db.FindASN(ip)
		if got.OrganisationNumber != want || got.FoundASN != (want != 0) {
			t.Fatalf("%s: %+v", ip, got)
		}
	}
}

var comparisonResult int

func BenchmarkCompareIP(b *testing.B) {
	for _, pair := range [][2]string{{"1.2.3.4", "8.8.8.8"}, {"2001:db8::1", "2001:db8::2"}} {
		a, c := net.ParseIP(pair[0]), net.ParseIP(pair[1])
		b.Run(pair[0]+"/bytes", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				comparisonResult = compareIP(a, c)
			}
		})
		b.Run(pair[0]+"/bigint_baseline", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				comparisonResult = ipToBigInt(a).Cmp(ipToBigInt(c))
			}
		})
	}
}
