package db

import (
	"fmt"
	"os"

	"github.com/tagphi/czdb-search-golang/pkg/utils"
)

// HyperHeaderBlock 表示超级头部块
type HyperHeaderBlock struct {
	Version           int32
	ClientId          int32
	EncryptedBlockSize int32
	DecryptedBlock    *DecryptedBlock
}

// 解密超级头部块
func DecryptHyperHeaderBlock(file *os.File, key string) (*HyperHeaderBlock, error) {
	// 读取版本号和客户端ID（共8字节）
	headerBytes := make([]byte, 8)
	bytesRead, err := file.Read(headerBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read HyperHeaderBlock: %v", err)
	}
	if bytesRead < 8 {
		return nil, fmt.Errorf("incomplete HyperHeaderBlock read: %d of 8 bytes", bytesRead)
	}
	
	// 解析超级头部块
	hyperHeader := &HyperHeaderBlock{
		Version:  utils.GetIntLong(headerBytes, 0),
		ClientId: utils.GetIntLong(headerBytes, 4),
	}
	
	// 读取加密块大小（4字节）
	encryptedBlockSizeBytes := make([]byte, 4)
	bytesRead, err = file.Read(encryptedBlockSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted block size: %v", err)
	}
	if bytesRead < 4 {
		return nil, fmt.Errorf("incomplete encrypted block size read: %d of 4 bytes", bytesRead)
	}
	
	hyperHeader.EncryptedBlockSize = utils.GetIntLong(encryptedBlockSizeBytes, 0)
	
	// 检查加密块大小是否有效
	if hyperHeader.EncryptedBlockSize <= 0 || hyperHeader.EncryptedBlockSize > 1000000 {
		return nil, fmt.Errorf("invalid encrypted block size: %d", hyperHeader.EncryptedBlockSize)
	}
	
	// 读取加密块
	encryptedBlockBytes := make([]byte, hyperHeader.EncryptedBlockSize)
	bytesRead, err = file.Read(encryptedBlockBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted block: %v", err)
	}
	if bytesRead < int(hyperHeader.EncryptedBlockSize) {
		return nil, fmt.Errorf("incomplete encrypted block read: %d of %d bytes", bytesRead, hyperHeader.EncryptedBlockSize)
	}
	
	// 解密加密块
	decryptedBytes, err := DecryptEncryptedBytes(encryptedBlockBytes, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt block: %v", err)
	}
	
	// 解析ClientId和ExpirationDate
	if len(decryptedBytes) >= 8 {
		// 按照白皮书解析，ClientId在高12位，ExpirationDate在低20位
		combinedValue := utils.GetIntLong(decryptedBytes, 0)
		
		decryptedBlock := &DecryptedBlock{
			ClientId:       combinedValue >> ClientIdShift,           // 获取高12位，右移20位
			ExpirationDate: combinedValue & ExpirationDateMask,       // 获取低20位，用掩码提取
		}
		
		// 获取随机数据大小
		if len(decryptedBytes) >= 12 {
			decryptedBlock.RandomSize = utils.GetIntLong(decryptedBytes, 4)
		}
		
		hyperHeader.DecryptedBlock = decryptedBlock
	} else {
		return nil, fmt.Errorf("decrypted data too small: %d bytes", len(decryptedBytes))
	}
	
	return hyperHeader, nil
}

// 获取超级头部块大小
func GetHyperHeaderBlockSize(hyperHeader *HyperHeaderBlock) int {
	// 版本号(4字节) + 客户端ID(4字节) + 加密块大小(4字节) + 加密块
	return 12 + int(hyperHeader.EncryptedBlockSize)
} 