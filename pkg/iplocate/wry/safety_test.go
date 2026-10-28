package wry

import (
	"encoding/binary"
	"testing"
)

func TestReaderSafeRecords(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("C\x00A\x00"),
		{1, 4, 0, 0, 'C', 0, 'A', 0},
		{2, 8, 0, 0, 'A', 0, 0, 0, 'C', 0},
	} {
		r := NewReader(data)
		if err := r.Parse(0); err != nil {
			t.Fatal(err)
		}
		if r.Result.Country != "C" || r.Result.Area != "A" {
			t.Fatalf("decoded: %+v", r.Result)
		}
	}
	// Null area pointer is a valid empty area.
	r := NewReader([]byte{'C', 0, 2, 0, 0, 0})
	if err := r.Parse(0); err != nil || r.Result.Country != "C" || r.Result.Area != "" {
		t.Fatalf("empty area: %+v %v", r.Result, err)
	}
}

func TestReaderRejectsCorruption(t *testing.T) {
	for _, data := range [][]byte{nil, {'C'}, {1}, {2, 255, 255, 255}, {1, 0, 0, 0}, {'C', 0}, {'C', 0, 'A'}} {
		r := NewReader(data)
		if err := r.Parse(0); err == nil {
			t.Fatalf("accepted corrupt data: %v", data)
		}
		if r.Result != (Result{}) {
			t.Fatal("partial result escaped")
		}
	}
	r := NewReader([]byte("C\x00A\x00"))
	if err := r.Parse(^uint32(0)); err == nil {
		t.Fatal("out of range offset accepted")
	}
	if got := Bytes3ToUint32([]byte{1}); got != 0 {
		t.Fatalf("short offset: %d", got)
	}
}

func TestSafeIndexBoundaries(t *testing.T) {
	data := make([]byte, 128)
	for i, ip := range []uint32{1, 10} {
		pos := 8 + i*7
		binary.LittleEndian.PutUint32(data[pos:], ip)
		data[pos+4] = byte(64 + i*16)
	}
	db := IPDB[uint32]{Data: data, IPLen: 4, OffLen: 3, IdxStart: 8, IdxEnd: 15}
	for ip, want := range map[uint32]uint32{0: 0, 1: 64, 5: 64, 10: 80, 11: 80} {
		if got := db.SearchIndexV4(ip); got != want {
			t.Fatalf("v4 %d: got %d want %d", ip, got, want)
		}
	}
	for i, ip := range []uint64{1, 10} {
		pos := 24 + i*11
		binary.LittleEndian.PutUint64(data[pos:], ip)
		data[pos+8] = byte(64 + i*16)
	}
	db6 := IPDB[uint64]{Data: data, IPLen: 8, OffLen: 3, IdxStart: 24, IdxEnd: 35}
	for ip, want := range map[uint64]uint32{0: 0, 1: 64, 5: 64, 10: 80, 11: 80} {
		if got := db6.SearchIndexV6(ip); got != want {
			t.Fatalf("v6 %d: got %d want %d", ip, got, want)
		}
	}
	db6.IPLen = 0
	if got := db6.SearchIndexV6(1); got != 0 {
		t.Fatalf("bad header: %d", got)
	}
	db6.IPLen = 8
	db6.IdxStart, db6.IdxEnd = ^uint64(0), ^uint64(0)
	if got := db6.SearchIndexV6(1); got != 0 {
		t.Fatalf("overflowed header: %d", got)
	}
	db.Data = nil
	if got := db.SearchIndexV4(1); got != 0 {
		t.Fatalf("closed data: %d", got)
	}
}

func FuzzReaderParse(f *testing.F) {
	f.Add([]byte("C\x00A\x00"), uint32(0))
	f.Add([]byte{1, 0, 0, 0}, uint32(0))
	f.Add([]byte{2, 255, 255, 255}, uint32(0))
	f.Fuzz(func(t *testing.T, data []byte, offset uint32) {
		if len(data) > 4096 {
			t.Skip()
		}
		r := NewReader(data)
		_ = r.Parse(offset)
	})
}
