package ip2region

import (
	"errors"
	"sync"
	"testing"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/winezer0/ipinfo/internal/testxdb"
)

func TestPoolLifecycle(t *testing.T) {
	for _, version := range []int{4, 6} {
		path := testxdb.Write(t, version)
		db, err := NewIP2RegionAutoWithPool(path, 4)
		if err != nil {
			t.Fatal(err)
		}
		ip, want := "1.1.1.1", "first|province|city|isp"
		if version == 6 {
			ip = "2001:db8::1"
		}
		var wg sync.WaitGroup
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for n := 0; n < 100; n++ {
					got := db.FindFull(ip)
					if got.HasData() && got.Location != want {
						t.Errorf("corrupt single: %+v", got)
					}
					batch := db.BatchFindFull([]string{ip, "invalid"})
					if batch[ip].HasData() && batch[ip].Location != want {
						t.Errorf("corrupt batch: %+v", batch[ip])
					}
					if batch["invalid"].HasData() {
						t.Error("invalid address hit")
					}
				}
			}()
		}
		for n := 0; n < 12; n++ {
			db.Close()
			if err := db.Init(); err != nil {
				t.Fatal(err)
			}
		}
		wg.Wait()
		if db.PoolSize() != 4 || db.FindFull(ip).Location != want {
			t.Fatal("pool failed to reopen")
		}
		db.Close()
		db.Close()
		if db.PoolSize() != 0 || db.FindFull(ip).HasData() {
			t.Fatal("closed pool still active")
		}
	}
}

func TestPoolPartialFailure(t *testing.T) {
	path := testxdb.Write(t, 4)
	db := &IP2Region{Version: 4, DbPath: path, poolSize: 4}
	var opened []*xdb.Searcher
	var index *byte
	err := db.initWithVector(nil, func(v *xdb.Version, p string, data []byte) (*xdb.Searcher, error) {
		if index == nil {
			index = &data[0]
		} else if index != &data[0] {
			t.Fatal("index was duplicated")
		}
		if len(opened) == 2 {
			return nil, errors.New("injected open failure")
		}
		s, err := xdb.NewWithVectorIndex(v, p, data)
		if err == nil {
			opened = append(opened, s)
		}
		return s, err
	})
	if err == nil || db.IsInitialized() || db.PoolSize() != 0 {
		t.Fatal("partial pool published")
	}
	for _, s := range opened {
		if _, err := s.SearchByStr("1.1.1.1"); err == nil {
			t.Error("partial handle leaked")
		}
	}
	if err := db.Init(); err != nil {
		t.Fatal(err)
	}
	db.Close()
	for _, size := range []int{0, -1, MaxPoolSize + 1} {
		if db, err := NewIP2RegionWithPool(4, path, size); err == nil {
			db.Close()
			t.Errorf("accepted size %d", size)
		}
	}
	if db, err := NewIP2RegionWithVectorIndex(4, path, []byte{1}); err == nil {
		db.Close()
		t.Fatal("short index accepted")
	}
}
