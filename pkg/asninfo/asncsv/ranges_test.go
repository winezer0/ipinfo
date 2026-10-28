package asncsv

import (
	"net"
	"reflect"
	"testing"
)

func TestExactRangeCIDRs(t *testing.T) {
	for _, tt := range []struct {
		start, end string
		want       []string
	}{
		{"1.0.0.1", "1.0.0.3", []string{"1.0.0.1/32", "1.0.0.2/31"}},
		{"1.0.0.1", "1.0.0.4", []string{"1.0.0.1/32", "1.0.0.2/31", "1.0.0.4/32"}},
		{"0.0.0.0", "255.255.255.255", []string{"0.0.0.0/0"}},
		{"255.255.255.255", "255.255.255.255", []string{"255.255.255.255/32"}},
		{"2001:db8::1", "2001:db8::4", []string{"2001:db8::1/128", "2001:db8::2/127", "2001:db8::4/128"}},
		{"::", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", []string{"::/0"}},
	} {
		t.Run(tt.start, func(t *testing.T) {
			var got []string
			for _, prefix := range rangeToCIDRs(net.ParseIP(tt.start), net.ParseIP(tt.end)) {
				got = append(got, prefix.String())
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
	for _, record := range [][]string{{"1.0.0.4", "1.0.0.1", "1", "bad"}, {"1.0.0.1", "2001:db8::1", "1", "bad"}} {
		if parseCSVRecord(record) != nil {
			t.Fatal("invalid range accepted")
		}
	}
}

func TestCSVReverseLifecycle(t *testing.T) {
	c := NewASNCsvQuerier("")
	if _, err := c.ASNToIPRanges(1); err == nil {
		t.Fatal("uninitialized reverse lookup accepted")
	}
	c.records = []ASNRecord{{StartIP: net.ParseIP("1.0.0.1"), EndIP: net.ParseIP("1.0.0.4"), AutonomousSystemNumber: 1}}
	c.loaded = true
	ranges, err := c.ASNToIPRanges(1)
	if err != nil || len(ranges) != 3 {
		t.Fatalf("ranges=%v err=%v", ranges, err)
	}
	c.Close()
	if _, err := c.ASNToIPRanges(1); err == nil {
		t.Fatal("closed reverse lookup accepted")
	}
}
