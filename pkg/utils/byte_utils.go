package utils

import (
	"fmt"
	"net"
)

// IP type constants
const (
	IPV4 = 4
	IPV6 = 6
)

// GetIntLong 从字节数组中指定位置读取一个32位整数（小端序）
func GetIntLong(b []byte, offset int) int32 {
	if offset+4 > len(b) {
		return 0
	}
	return int32(b[offset]) | int32(b[offset+1])<<8 | int32(b[offset+2])<<16 | int32(b[offset+3])<<24
}

// GetLongLong 从字节数组中指定位置读取一个64位整数（小端序）
func GetLongLong(b []byte, offset int) int64 {
	if offset+8 > len(b) {
		return 0
	}
	return int64(b[offset]) | int64(b[offset+1])<<8 | int64(b[offset+2])<<16 | int64(b[offset+3])<<24 |
		int64(b[offset+4])<<32 | int64(b[offset+5])<<40 | int64(b[offset+6])<<48 | int64(b[offset+7])<<56
}

// GetShort 从字节数组中指定位置读取一个16位整数（小端序）
func GetShort(b []byte, offset int) int16 {
	if offset+2 > len(b) {
		return 0
	}
	return int16(b[offset]) | int16(b[offset+1])<<8
}

// GetByte 从字节数组中指定位置读取一个字节
func GetByte(b []byte, offset int) byte {
	if offset >= len(b) {
		return 0
	}
	return b[offset]
}

// GetInt1 reads a 1-byte integer from a byte slice at the given offset
func GetInt1(b []byte, offset int) int8 {
	return int8(b[offset])
}

// PrintBytesInHex prints a byte slice in hexadecimal format
func PrintBytesInHex(bytes []byte) {
	for _, b := range bytes {
		fmt.Printf("%02x ", b)
	}
	fmt.Println()
}

// PrintBytesInDecimal prints a byte slice in decimal format
func PrintBytesInDecimal(bytes []byte) {
	for _, b := range bytes {
		fmt.Printf("%d ", b)
	}
	fmt.Println()
}

// GetIPBytes converts an IP string to its byte representation
// Returns the IP bytes and an error if any
func GetIPBytes(ip string, dbType int) ([]byte, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address format: %s", ip)
	}

	// For IPv4 addresses, ParseIP returns a 16-byte representation (IPv4-mapped IPv6 address)
	// We need to extract just the IPv4 portion if dbType is IPV4
	if dbType == IPV4 {
		if parsedIP.To4() != nil {
			return parsedIP.To4(), nil
		}
		return nil, fmt.Errorf("expected IPv4 address but got IPv6: %s", ip)
	} else if dbType == IPV6 {
		// For IPv6 addresses
		if len(parsedIP) == 16 {
			return parsedIP, nil
		}
		return nil, fmt.Errorf("expected IPv6 address but got IPv4: %s", ip)
	}

	return nil, fmt.Errorf("invalid IP type: %d", dbType)
}

// CompareBytes compares two byte slices up to the specified length
// Returns a negative number if bytes1 < bytes2, 0 if bytes1 == bytes2, positive number if bytes1 > bytes2
func CompareBytes(bytes1, bytes2 []byte, length int) int {
	for i := 0; i < length; i++ {
		if i >= len(bytes1) || i >= len(bytes2) {
			return len(bytes1) - len(bytes2)
		}
		if bytes1[i] != bytes2[i] {
			return int(bytes1[i]) - int(bytes2[i])
		}
	}
	return 0
}

// Decrypt 使用 XOR 解密字节数据
func Decrypt(encryptedBytes []byte, key []byte) {
	for i := 0; i < len(encryptedBytes); i++ {
		encryptedBytes[i] = encryptedBytes[i] ^ key[i%len(key)]
	}
}

// EncodeIP 将 IP 地址转换为 uint32
func EncodeIP(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
} 