// Package testxdb builds small deterministic XDB files for tests.
package testxdb

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// Write creates two distinct ranges covering the requested address family.
func Write(t testing.TB, version int) string {
	t.Helper()
	ipBytes := 4
	if version == 6 {
		ipBytes = 16
	}
	segmentSize := ipBytes*2 + 6
	indexStart := xdb.HeaderInfoLength + xdb.VectorIndexRows*xdb.VectorIndexCols*xdb.VectorIndexSize
	dataStart := indexStart + segmentSize*2
	regions := []string{"first|province|city|isp", "second|province|city|isp"}
	data := make([]byte, dataStart+len(regions[0])+len(regions[1]))
	binary.LittleEndian.PutUint16(data, xdb.Structure30)
	binary.LittleEndian.PutUint16(data[2:], uint16(xdb.VectorIndexPolicy))
	binary.LittleEndian.PutUint32(data[8:], uint32(indexStart))
	binary.LittleEndian.PutUint32(data[12:], uint32(indexStart+segmentSize))
	binary.LittleEndian.PutUint16(data[16:], uint16(version))
	binary.LittleEndian.PutUint16(data[18:], 4)
	for i := xdb.HeaderInfoLength; i < indexStart; i += xdb.VectorIndexSize {
		binary.LittleEndian.PutUint32(data[i:], uint32(indexStart))
		binary.LittleEndian.PutUint32(data[i+4:], uint32(indexStart+segmentSize))
	}
	for i, region := range regions {
		segment := data[indexStart+i*segmentSize:][:segmentSize]
		start, end := segment[:ipBytes], segment[ipBytes:ipBytes*2]
		for j := range end {
			end[j] = 255
		}
		if i == 0 {
			end[0] = 127
		} else {
			start[0] = 128
		}
		if version == 4 {
			for _, ip := range [][]byte{start, end} {
				ip[0], ip[3] = ip[3], ip[0]
				ip[1], ip[2] = ip[2], ip[1]
			}
		}
		binary.LittleEndian.PutUint16(segment[ipBytes*2:], uint16(len(region)))
		binary.LittleEndian.PutUint32(segment[ipBytes*2+2:], uint32(dataStart))
		copy(data[dataStart:], region)
		dataStart += len(region)
	}
	path := filepath.Join(t.TempDir(), "fixture.xdb")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
