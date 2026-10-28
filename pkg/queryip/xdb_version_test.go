package queryip

import (
	"errors"
	"github.com/winezer0/ipinfo/internal/testxdb"
	"os"
	"testing"
)

func TestMissingIPv6ClosesIPv4(t *testing.T) {
	path := testxdb.Write(t, 4)
	if _, err := InitDBEngines(&IPDbConfig{Ipv4LocateDb: path}); !errors.Is(err, ErrMissingIPv6Config) {
		t.Fatalf("missing IPv6: %v", err)
	}
	// On Windows a leaked open file prevents removal.
	if err := os.Remove(path); err != nil {
		t.Fatalf("IPv4 resource was not released: %v", err)
	}
}

func TestXDBVersions(t *testing.T) {
	db, err := InitDBEngines(&IPDbConfig{Ipv4LocateDb: testxdb.Write(t, 4), Ipv6LocateDb: testxdb.Write(t, 6)})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, ip := range []string{"1.1.1.1", "2001:db8::1"} {
		if got := db.QueryIP(ip).IPLocate.Location; got != "first|province|city|isp" {
			t.Fatalf("%s: %q", ip, got)
		}
	}
}
