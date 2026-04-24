package wry

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// IPDB common ip database
type IPDB[T ~uint32 | ~uint64] struct {
	Data []byte

	OffLen   uint8
	IPLen    uint8
	IPCnt    T
	IdxStart T
	IdxEnd   T
}

type Reader struct {
	s []byte
	i uint32 // current reading index
	l uint32 // last reading index

	Result Result
}

func NewReader(data []byte) Reader {
	return Reader{s: data, i: 0, l: 0, Result: Result{
		Country: "",
		Area:    "",
	}}
}

// seekAbs 绝对定位，带边界检查
func (r *Reader) seekAbs(offset uint32) {
	r.l = r.i
	if offset >= uint32(len(r.s)) {
		r.i = uint32(len(r.s))
		return
	}
	r.i = offset
}

// seek 相对定位，带边界检查
func (r *Reader) seek(offset int64) {
	r.l = r.i
	newPos := int64(r.i) + offset
	if newPos < 0 {
		r.i = 0
	} else if newPos >= int64(len(r.s)) {
		r.i = uint32(len(r.s))
	} else {
		r.i = uint32(newPos)
	}
}

// seekBack: seek to last index, can only call once
func (r *Reader) seekBack() {
	r.i = r.l
}

// read 读取指定长度的数据，带边界检查
func (r *Reader) read(length uint32) []byte {
	// 边界检查：防止数组越界
	if r.i+length > uint32(len(r.s)) {
		length = uint32(len(r.s)) - r.i
	}

	rs := make([]byte, length)
	copy(rs, r.s[r.i:])
	r.l = r.i
	r.i += length
	return rs
}

func (r *Reader) readMode() (mode byte) {
	mode = r.s[r.i]
	r.l = r.i
	r.i += 1
	return
}

// readOffset: read 3 bytes as uint32 offset
func (r *Reader) readOffset(follow bool) uint32 {
	buf := r.read(3)
	offset := Bytes3ToUint32(buf)
	if follow {
		r.l = r.i
		r.i = offset
	}
	return offset
}

func (r *Reader) readString(seek bool) string {
	length := bytes.IndexByte(r.s[r.i:], 0)
	str := string(r.s[r.i : r.i+uint32(length)])
	if seek {
		r.l = r.i
		r.i += uint32(length) + 1
	}
	return str
}

type Result struct {
	Country string `json:"country"`
	Area    string `json:"area"`
}

func (r *Result) DecodeGBK() *Result {
	enc := simplifiedchinese.GBK.NewDecoder()
	r.Country, _ = enc.String(r.Country)
	r.Area, _ = enc.String(r.Area)
	return r
}

func (r *Result) Trim() *Result {
	r.Country = strings.TrimSpace(strings.ReplaceAll(r.Country, "CZ88.NET", ""))
	r.Area = strings.TrimSpace(strings.ReplaceAll(r.Area, "CZ88.NET", ""))
	return r
}

func (r Result) String() string {
	r.Trim()
	return strings.TrimSpace(fmt.Sprintf("%s %s", r.Country, r.Area))
}

func Bytes3ToUint32(data []byte) uint32 {
	i := uint32(data[0]) & 0xff
	i |= (uint32(data[1]) << 8) & 0xff00
	i |= (uint32(data[2]) << 16) & 0xff0000
	return i
}
