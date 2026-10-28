package multiloc

import (
	"testing"

	"github.com/winezer0/ipinfo/internal/testxdb"
)

func TestXDBVersions(t *testing.T) {
	for _, version := range []int{4, 6} {
		db, err := createIPEngine(testxdb.Write(t, version))
		if err != nil {
			t.Fatal(err)
		}
		ip := "1.1.1.1"
		if version == 6 {
			ip = "2001:db8::1"
		}
		info := db.GetDatabaseInfo()
		if info.IsIPv4 != (version == 4) || info.IsIPv6 != (version == 6) {
			t.Fatalf("capabilities: %+v", info)
		}
		got := db.FindFull(ip).Location
		db.Close()
		if got != "first|province|city|isp" {
			t.Fatalf("%s: %q", ip, got)
		}
	}
}
