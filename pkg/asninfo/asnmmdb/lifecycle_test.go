package asnmmdb

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/winezer0/ipinfo/pkg/asninfo"
)

func mmdbString(s string) []byte {
	if len(s) < 29 {
		return append([]byte{0x40 | byte(len(s))}, s...)
	}
	return append([]byte{0x5d, byte(len(s) - 29)}, s...)
}

func mmdbUint(n uint32) []byte {
	b := []byte{0xc4, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(b[1:], n)
	return b
}

func mmdbMap(values map[string][]byte) []byte {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	data := []byte{0xe0 | byte(len(keys))}
	for _, key := range keys {
		data = append(data, mmdbString(key)...)
		data = append(data, values[key]...)
	}
	return data
}

// writeMMDB creates a minimal IPv4 MMDB with two /1 records.
func writeMMDB(t *testing.T, path string, asn uint32) {
	t.Helper()
	record := mmdbMap(map[string][]byte{"autonomous_system_number": mmdbUint(asn), "autonomous_system_organization": mmdbString("fixture")})
	data := make([]byte, 22) // one 24-bit node and the 16-byte separator
	for _, pos := range []int{0, 3} {
		data[pos+2] = 17
	}
	data = append(data, record...)
	data = append(data, []byte("\xab\xcd\xefMaxMind.com")...)
	data = append(data, mmdbMap(map[string][]byte{
		"node_count": mmdbUint(1), "record_size": mmdbUint(24), "ip_version": mmdbUint(4),
		"binary_format_major_version": mmdbUint(2), "binary_format_minor_version": mmdbUint(0), "database_type": mmdbString("test ASN"),
	})...)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMMDBCorrectnessLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.mmdb")
	writeMMDB(t, path, 64500)
	m, err := NewMMDBManager(&MMDBConfig{AsnIpvxDb: path, MaxConcurrentQueries: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.ASNToIPRanges(64500); !errors.Is(err, asninfo.ErrDatabaseNotInitialized) {
		t.Fatalf("uninitialized: %v", err)
	}
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if got := m.FindASN("1.1.1.1"); !got.FoundASN || got.OrganisationNumber != 64500 {
		t.Fatalf("lookup: %+v", got)
	}
	if ranges, err := m.ASNToIPRanges(64500); err != nil || len(ranges) == 0 {
		t.Fatalf("reverse: %v %v", ranges, err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.FindASN("1.1.1.1")
				m.ASNToIPRanges(64500)
			}
		}()
	}
	for i := 0; i < 20; i++ {
		if err := m.Close(); err != nil {
			t.Fatal(err)
		}
		if err := m.Init(); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ASNToIPRanges(64500); !errors.Is(err, asninfo.ErrDatabaseNotInitialized) {
		t.Fatalf("closed: %v", err)
	}
	writeMMDB(t, path, 64501)
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	if ranges, err := m.ASNToIPRanges(64500); err != nil || len(ranges) != 0 {
		t.Fatalf("stale index: %v %v", ranges, err)
	}
	if ranges, err := m.ASNToIPRanges(64501); err != nil || len(ranges) == 0 {
		t.Fatalf("reopened: %v %v", ranges, err)
	}
}

func TestMMDBPermitCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.mmdb")
	writeMMDB(t, path, 64500)
	m, err := NewMMDBManager(&MMDBConfig{AsnIpvxDb: path, MaxConcurrentQueries: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.queryChan <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := m.FindASNContext(ctx, "1.1.1.1"); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("permit saturation returned immediately: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation blocked")
	}
	<-m.queryChan
	if got := m.FindASN("1.1.1.1"); !got.FoundASN {
		t.Fatal("permit not released")
	}
}
