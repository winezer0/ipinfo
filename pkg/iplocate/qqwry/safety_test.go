package qqwry

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestQQWrySafeFixture(t *testing.T) {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint32(data, 8)
	binary.LittleEndian.PutUint32(data[4:], 8)
	binary.LittleEndian.PutUint32(data[8:], 0)
	data[12] = 15
	binary.LittleEndian.PutUint32(data[15:], ^uint32(0))
	copy(data[19:], []byte("Country\x00Area\x00"))
	path := filepath.Join(t.TempDir(), "test.dat")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	db, err := NewQQWryDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := db.FindFull("1.1.1.1"); !got.HasData() {
		t.Fatalf("valid fixture: %+v", got)
	}
	db.Close()
	if got := db.FindFull("1.1.1.1"); got.HasData() {
		t.Fatalf("closed fixture: %+v", got)
	}
	data[12], data[13], data[14] = 255, 255, 255
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	db, err = NewQQWryDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := db.FindFull("1.1.1.1"); got.HasData() {
		t.Fatalf("corrupt pointer: %+v", got)
	}
	binary.LittleEndian.PutUint32(data[4:], ^uint32(0))
	if checkIPv4File(data) {
		t.Fatal("overflowed header accepted")
	}
}
