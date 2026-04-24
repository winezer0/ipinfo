package iputils

import "testing"

// TestClassifyIPs 测试IP分类函数
func TestClassifyIPs(t *testing.T) {
	input := []string{
		"192.168.1.1",
		"8.8.8.8",
		"2001:db8::1",
		"::1",
		"invalid_ip",
		"not_an_ip",
		"10.0.0.1",
		"fe80::1",
	}

	result := ClassifyIPs(input)

	// 验证IPv4结果
	expectedIPv4s := []string{"192.168.1.1", "8.8.8.8", "10.0.0.1"}
	if len(result.IPv4s) != len(expectedIPv4s) {
		t.Errorf("期望 %d 个IPv4地址，得到 %d 个", len(expectedIPv4s), len(result.IPv4s))
	}
	for _, ip := range expectedIPv4s {
		if !contains(result.IPv4s, ip) {
			t.Errorf("IPv4列表中缺少: %s", ip)
		}
	}

	// 验证IPv6结果
	expectedIPv6s := []string{"2001:db8::1", "::1", "fe80::1"}
	if len(result.IPv6s) != len(expectedIPv6s) {
		t.Errorf("期望 %d 个IPv6地址，得到 %d 个", len(expectedIPv6s), len(result.IPv6s))
	}
	for _, ip := range expectedIPv6s {
		if !contains(result.IPv6s, ip) {
			t.Errorf("IPv6列表中缺少: %s", ip)
		}
	}

	// 验证其他结果
	expectedOther := []string{"invalid_ip", "not_an_ip"}
	if len(result.Other) != len(expectedOther) {
		t.Errorf("期望 %d 个其他地址，得到 %d 个", len(expectedOther), len(result.Other))
	}
	for _, ip := range expectedOther {
		if !contains(result.Other, ip) {
			t.Errorf("其他列表中缺少: %s", ip)
		}
	}

	t.Logf("IPv4: %v", result.IPv4s)
	t.Logf("IPv6: %v", result.IPv6s)
	t.Logf("Other: %v", result.Other)
}

// TestClassifyIPs_Empty 测试空输入
func TestClassifyIPs_Empty(t *testing.T) {
	result := ClassifyIPs([]string{})

	if len(result.IPv4s) != 0 || len(result.IPv6s) != 0 || len(result.Other) != 0 {
		t.Error("空输入应返回空结果")
	}
}

// TestClassifyIPs_Nil 测试nil输入
func TestClassifyIPs_Nil(t *testing.T) {
	result := ClassifyIPs(nil)

	if len(result.IPv4s) != 0 || len(result.IPv6s) != 0 || len(result.Other) != 0 {
		t.Error("nil输入应返回空结果")
	}
}

// contains 检查切片中是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
