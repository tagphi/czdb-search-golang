package db

import (
	"crypto/aes"
	"encoding/base64"
	"fmt"
)

const (
	ExpirationDateMask = 0xFFFFF                 // 低20位掩码
	ClientIdShift      = 20                      // 客户端ID需要右移20位
)

// DecryptedBlock 表示解密后的块
type DecryptedBlock struct {
	ClientId       int32 // 客户端ID (12位)
	ExpirationDate int32 // 过期日期 (20位)
	RandomSize     int32 // 随机数据大小
}

// Base64Decode 从Base64字符串解码二进制数据
func Base64Decode(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %v", err)
	}
	return decoded, nil
}

// AESECBDecrypt 使用AES ECB模式解密数据
func AESECBDecrypt(encryptedData []byte, key []byte) ([]byte, error) {
	if len(encryptedData) == 0 {
		return []byte{}, nil
	}
	
	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data length %d is not a multiple of AES block size", len(encryptedData))
	}
	
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}
	
	decrypted := make([]byte, len(encryptedData))
	
	// ECB模式下，逐个块解密
	for bs, be := 0, cipher.BlockSize(); bs < len(encryptedData); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], encryptedData[bs:be])
	}
	
	return decrypted, nil
}

// DecryptEncryptedBytes 使用给定的key解密数据
func DecryptEncryptedBytes(encryptedBytes []byte, key string) ([]byte, error) {
	keyBytes, err := Base64Decode(key)
	if err != nil {
		return nil, err
	}
	
	// 检查key长度
	if len(keyBytes) != 16 && len(keyBytes) != 24 && len(keyBytes) != 32 {
		return nil, fmt.Errorf("invalid key length, must be 16, 24, or 32 bytes (got %d)", len(keyBytes))
	}
	
	return AESECBDecrypt(encryptedBytes, keyBytes)
}