package zxwry

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestZXWrySafeFixture(t *testing.T) {
	data := make([]byte, 96)
	copy(data, "IPDB")
	data[6], data[7] = 3, 8
	binary.LittleEndian.PutUint64(data[8:], 2)
	binary.LittleEndian.PutUint64(data[16:], 24)
	data[32] = 64
	binary.LittleEndian.PutUint64(data[35:], uint64(1)<<63)
	data[43] = 80
	copy(data[64:], []byte("First\x00Area\x00"))
	copy(data[80:], []byte("Second\x00Area\x00"))
	path := filepath.Join(t.TempDir(), "test.db")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	db, err := NewZXWryDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for ip, want := range map[string]string{"2001:db8::1": "First", "ffff::1": "Second"} {
		if got := db.FindFull(ip); got.Country != want {
			t.Fatalf("%s: %+v", ip, got)
		}
	}
	if got := db.FindFull("1.1.1.1"); got.HasData() {
		t.Fatalf("IPv4 accepted: %+v", got)
	}
	data[6] = 0
	if checkIPv6File(data) {
		t.Fatal("unsupported offset width accepted")
	}
	data[6] = 3
	binary.LittleEndian.PutUint64(data[8:], ^uint64(0))
	if checkIPv6File(data) {
		t.Fatal("overflowed count accepted")
	}
}
