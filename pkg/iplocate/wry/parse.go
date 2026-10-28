package wry

import "fmt"

const (
	// RedirectMode1 [IP][0x01][国家和地区信息的绝对偏移地址]
	RedirectMode1 = 0x01
	// RedirectMode2 [IP][0x02][信息的绝对偏移][...] or [IP][国家][...]
	RedirectMode2 = 0x02
)

func (r *Reader) Parse(offset uint32) error {
	r.err = nil
	r.Result = Result{}
	if offset != 0 {
		r.seekAbs(offset)
	}
	for depth := 0; depth < 64 && r.err == nil; depth++ {
		switch r.readMode() {
		case RedirectMode1:
			r.readOffset(true)
			continue
		case RedirectMode2:
			r.Result.Country = r.parseRedMode2()
			r.Result.Area = r.readArea()
		default:
			r.seekBack()
			r.Result.Country = r.readString(true)
			r.Result.Area = r.readArea()
		}
		if r.err != nil {
			r.Result = Result{}
		}
		return r.err
	}
	if r.err == nil {
		r.err = fmt.Errorf("数据库重定向超过64次，可能存在循环")
	}
	r.Result = Result{}
	return r.err
}

func (r *Reader) parseRedMode2() string {
	r.readOffset(true)
	str := r.readString(false)
	r.seekBack()
	return str
}

func (r *Reader) readArea() string {
	mode := r.readMode()
	if mode == RedirectMode1 || mode == RedirectMode2 {
		offset := r.readOffset(false)
		if offset == 0 {
			return ""
		}
		r.seekAbs(offset)
	} else {
		r.seekBack()
	}
	return r.readString(false)
}
