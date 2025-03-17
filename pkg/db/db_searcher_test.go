package db

import (
	"os"
	"testing"
)

// TestIPToUint32 测试IP地址到uint32的转换
func TestIPToUint32(t *testing.T) {
	tests := []struct {
		ip       string
		expected uint32
		hasError bool
	}{
		{"192.168.1.1", 3232235777, false},
		{"8.8.8.8", 134744072, false},
		{"255.255.255.255", 4294967295, false},
		{"0.0.0.0", 0, false},
		{"256.256.256.256", 0, true}, // 无效IP
		{"invalid", 0, true},         // 无效IP
	}

	for _, test := range tests {
		result, err := ipToUint32(test.ip)
		if test.hasError && err == nil {
			t.Errorf("ipToUint32(%s)应该返回错误", test.ip)
		}
		if !test.hasError && err != nil {
			t.Errorf("ipToUint32(%s)不应该返回错误: %v", test.ip, err)
		}
		if !test.hasError && result != test.expected {
			t.Errorf("ipToUint32(%s) = %d, 期望 %d", test.ip, result, test.expected)
		}
	}
}

// TestCleanString 测试字符串清理函数
func TestCleanString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test\x00string", "teststring"},
		{"hello\x00world\x00", "helloworld"},
		{"normal string", "normal string"},
		{"\x00\x00start", "start"},
		{"end\x00\x00", "end"},
	}

	for _, test := range tests {
		result := cleanString(test.input)
		if result != test.expected {
			t.Errorf("cleanString(%q) = %q, 期望 %q", test.input, result, test.expected)
		}
	}
}

// TestCompareBytes 测试字节比较函数
func TestCompareBytes(t *testing.T) {
	tests := []struct {
		a        []byte
		b        []byte
		length   int
		expected int
	}{
		{[]byte{1, 2, 3}, []byte{1, 2, 3}, 3, 0},
		{[]byte{1, 2, 3}, []byte{1, 2, 4}, 3, -1},
		{[]byte{1, 2, 4}, []byte{1, 2, 3}, 3, 1},
		{[]byte{1, 2, 3, 4}, []byte{1, 2, 3}, 3, 0},
		{[]byte{1, 2}, []byte{1, 2, 3}, 2, 0},
		{[]byte{1, 2, 3}, []byte{1, 2}, 3, 1}, // a更长
	}

	for _, test := range tests {
		result := compareBytes(test.a, test.b, test.length)
		if result != test.expected {
			t.Errorf("compareBytes(%v, %v, %d) = %d, 期望 %d", 
				test.a, test.b, test.length, result, test.expected)
		}
	}
}

// TestSearchTypeToString 测试搜索类型转字符串函数
func TestSearchTypeToString(t *testing.T) {
	tests := []struct {
		searchType SearchType
		expected   string
	}{
		{MEMORY, "Memory"},
		{BTREE, "BTree"},
		{SearchType(99), "Unknown"},
	}

	for _, test := range tests {
		result := searchTypeToString(test.searchType)
		if result != test.expected {
			t.Errorf("searchTypeToString(%d) = %s, 期望 %s", 
				test.searchType, result, test.expected)
		}
	}
}

// 集成测试 - 需要实际数据库文件才能运行
// 可以使用t.Skip跳过，或者在CI环境中提供测试数据
func TestIntegrationSearch(t *testing.T) {
	dbPath := os.Getenv("CZDB_TEST_DB_PATH")
	key := os.Getenv("CZDB_TEST_DB_KEY")
	
	if dbPath == "" || key == "" {
		t.Skip("跳过集成测试: 环境变量CZDB_TEST_DB_PATH或CZDB_TEST_DB_KEY未设置")
	}
	
	// 初始化数据库搜索器，使用内存模式
	dbSearcher, err := InitDBSearcher(dbPath, key, MEMORY)
	if err != nil {
		t.Fatalf("初始化数据库搜索器失败: %v", err)
	}
	defer CloseDBSearcher(dbSearcher)
	
	// 测试搜索
	ip := "8.8.8.8"
	region, err := Search(ip, dbSearcher)
	if err != nil {
		t.Errorf("搜索IP %s 失败: %v", ip, err)
	} else {
		t.Logf("IP: %s, 区域: %s", ip, region)
	}
} 