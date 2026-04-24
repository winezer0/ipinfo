package wry

import (
	"encoding/binary"
)

// SearchIndexV4 在IPv4索引中二分查找IP地址
// 返回对应的偏移地址，如果未找到返回0
func (db *IPDB[uint32]) SearchIndexV4(ip uint32) uint32 {
	ipLen := db.IPLen
	entryLen := uint32(db.OffLen + db.IPLen)

	l, r := db.IdxStart, db.IdxEnd
	var ipc, mid uint32
	var buf []byte

	// 添加循环次数限制，防止死循环
	maxIterations := int((r-l)/entryLen) + 1
	iterations := 0

	for l+entryLen <= r {
		// 安全检查：防止死循环
		iterations++
		if iterations > maxIterations {
			return 0
		}

		mid = (r-l)/entryLen/2*entryLen + l

		// 边界检查
		if mid+entryLen > uint32(len(db.Data)) {
			return 0
		}

		buf = db.Data[mid : mid+entryLen]
		ipc = uint32(binary.LittleEndian.Uint32(buf[:ipLen]))

		if ipc > ip {
			r = mid
		} else if ipc < ip {
			l = mid
		} else {
			return uint32(Bytes3ToUint32(buf[ipLen:entryLen]))
		}
	}

	// 处理最后一个条目
	if l+entryLen <= uint32(len(db.Data)) {
		buf = db.Data[l : l+entryLen]
		return uint32(Bytes3ToUint32(buf[ipLen:entryLen]))
	}

	return 0
}

// SearchIndexV6 在IPv6索引中二分查找IP地址
// 返回对应的偏移地址，如果未找到返回0
func (db *IPDB[uint64]) SearchIndexV6(ip uint64) uint32 {
	ipLen := db.IPLen
	entryLen := uint64(db.OffLen + db.IPLen)

	buf := make([]byte, entryLen)
	l, r, mid, ipc := db.IdxStart, db.IdxEnd, uint64(0), uint64(0)

	// 添加循环次数限制，防止死循环
	maxIterations := int((r-l)/entryLen) + 1
	iterations := 0

	for l+entryLen <= r {
		// 安全检查：防止死循环
		iterations++
		if iterations > maxIterations {
			return 0
		}

		mid = (r-l)/entryLen/2*entryLen + l

		// 边界检查
		if mid+entryLen > uint64(len(db.Data)) {
			return 0
		}

		buf = db.Data[mid : mid+entryLen]
		ipc = uint64(binary.LittleEndian.Uint64(buf[:ipLen]))

		if ipc > ip {
			r = mid
		} else if ipc < ip {
			l = mid
		} else {
			return Bytes3ToUint32(buf[ipLen:entryLen])
		}
	}

	// 处理最后一个条目
	if l+entryLen <= uint64(len(db.Data)) {
		buf = db.Data[l : l+entryLen]
		return Bytes3ToUint32(buf[ipLen:entryLen])
	}

	return 0
}
