package ip2region

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/winezer0/ipinfo/internal/testxdb"
)

func TestCachedConcurrentQueries(t *testing.T) {
	for _, version := range []int{4, 6} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			path := testxdb.Write(t, version)
			db, err := NewIP2RegionAuto(path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if db.Version != version {
				t.Fatalf("version = %d", db.Version)
			}
			ips := []string{"1.1.1.1", "200.1.1.1"}
			if version == 6 {
				ips = []string{"2001:db8::1", "f001::1"}
			}
			want := []string{"first|province|city|isp", "second|province|city|isp"}
			for i, ip := range ips {
				if got := db.FindFull(ip).Location; got != want[i] {
					t.Fatalf("%s: %q", ip, got)
				}
			}
			// A cached lookup needs only segment and region reads, no vector read.
			if got := db.Searcher.GetIOCount(); got != 3 {
				t.Fatalf("cached IO count = %d, want 3", got)
			}
			var wg sync.WaitGroup
			for worker := 0; worker < 16; worker++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for n := 0; n < 30; n++ {
						for i, ip := range ips {
							if got := db.FindFull(ip).Location; got != want[i] {
								t.Errorf("single %s: %q", ip, got)
								return
							}
						}
						batch := db.BatchFindFull(ips)
						for i, ip := range ips {
							if got := batch[ip].Location; got != want[i] {
								t.Errorf("batch %s: %q", ip, got)
								return
							}
						}
					}
				}()
			}
			wg.Wait()
			db.Close()
			if err := db.Init(); err != nil {
				t.Fatal(err)
			}
			if got := db.FindFull(ips[0]).Location; got != want[0] {
				t.Fatalf("reopen: %q", got)
			}
			vector, err := LoadVectorIndexFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			cached, err := NewIP2RegionWithVectorIndex(version, path, vector)
			if err != nil {
				t.Fatal(err)
			}
			defer cached.Close()
			if got := cached.FindFull(ips[1]).Location; got != want[1] {
				t.Fatalf("explicit vector: %q", got)
			}
		})
	}
}

func TestAutoVersionHeader(t *testing.T) {
	path := testxdb.Write(t, 4)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy structure 2.0 has no IP-version field.
	data[0], data[1], data[16], data[17] = xdb.Structure20, 0, 0, 0
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	db, err := NewIP2RegionAuto(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if db.Version != 4 {
		t.Fatalf("legacy version = %d", db.Version)
	}
	for _, content := range [][]byte{{1}, make([]byte, 256)} {
		bad := filepath.Join(t.TempDir(), "bad.xdb")
		if err := os.WriteFile(bad, content, 0600); err != nil {
			t.Fatal(err)
		}
		if db, err := NewIP2RegionAuto(bad); err == nil {
			db.Close()
			t.Fatal("invalid header accepted")
		}
	}
}
