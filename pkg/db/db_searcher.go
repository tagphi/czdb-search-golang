package db

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/cz88/czdb-search-golang/pkg/utils"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	// 常量定义，参考白皮书
	SuperPartLength    = 17 // SuperBlock 长度
	IndexBlockLength   = 9  // 索引块长度
	HeaderBlockLength  = 20 // 头部块长度，16 字节 IP + 4 字节数据指针
)

// SearchType 表示搜索类型
type SearchType int

const (
	MEMORY SearchType = iota // 内存模式
	BTREE                    // B-tree模式
)

// SuperBlock 表示超级块
type SuperBlock struct {
	DbType          byte   // 数据库类型 (0 表示 IPv4, 1 表示 IPv6)
	DbSize          int32  // 数据库大小
	HeaderBlockSize int32  // 头部块大小
	StartIndexPtr   int32  // 起始索引指针
	EndIndexPtr     int32  // 结束索引指针
}

// BtreeModeParam 表示B-tree模式参数
type BtreeModeParam struct {
	HeaderLength int      // 头部长度
	HeaderPtr    []int32  // 头部指针
	HeaderSip    [][]byte // 头部起始IP
}

// DBSearcher 表示数据库搜索器
type DBSearcher struct {
	IPType          int32       // IP地址类型 (IPv4 或 IPv6)
	SearchType      SearchType  // 搜索类型 (BTREE 或 MEMORY)
	File            *os.File    // 数据库文件
	DBBin           []byte      // 数据库二进制数据 (内存模式使用)
	DataSize        int32       // 数据大小
	DBKey           string      // 数据库密钥
	FileOffset      int64       // 文件偏移量 (HyperHeader + EncryptedBlock + RandomData 的大小)
	
	// B-tree搜索相关字段
	IPBytesLength     int        // IP字节长度
	StartIndexPtr     int32      // 起始索引指针
	EndIndexPtr       int32      // 结束索引指针
	IndexLength       int32      // 索引长度
	ColumnSelection   int32      // 列选择
	GeoMapData        []byte     // 地理映射数据
	
	// 新增字段
	HyperHeader       *HyperHeaderBlock // 超级头部
	DecryptedBlock    *DecryptedBlock   // 解密后的块
	SuperBlock        *SuperBlock       // 超级块
	BtreeModeParam    *BtreeModeParam   // B-tree模式参数
	HeaderBlock       []byte            // 头部块数据
	HeaderBlockSize   int32             // 头部块大小
}

// 解析SuperBlock
func parseSuperBlock(data []byte) (*SuperBlock, error) {
	if len(data) < SuperPartLength {
		return nil, fmt.Errorf("SuperBlock data too short: %d bytes, expected at least %d bytes", len(data), SuperPartLength)
	}
	
	// 按照白皮书格式解析
	// +--------+---------+---------+---------+---------+
	// | 1bytes | 4bytes  | 4bytes  | 4bytes  | 4bytes  |
	// +--------+---------+---------+---------+---------+
	// |db type |db size  |first    |header   |end      |
	// |        |         |index ptr|block size|index ptr|
	superBlock := &SuperBlock{
		DbType:          data[0],
		DbSize:          utils.GetIntLong(data, 1),
		StartIndexPtr:   utils.GetIntLong(data, 5),
		HeaderBlockSize: utils.GetIntLong(data, 9),
		EndIndexPtr:     utils.GetIntLong(data, 13),
	}
	
	// 打印调试信息
	fmt.Printf("Debug: Parsed SuperBlock: Type=%d, Size=%d, StartPtr=%d, HeaderSize=%d, EndPtr=%d\n", 
		superBlock.DbType, superBlock.DbSize, superBlock.StartIndexPtr, superBlock.HeaderBlockSize, superBlock.EndIndexPtr)
	
	return superBlock, nil
}

// 初始化B-tree模式参数
func initBtreeModeParam(file *os.File, offset int64, superBlock *SuperBlock) (*BtreeModeParam, error) {
	// 不再重复读取和解析SuperBlock，直接使用传入的superBlock参数
	
	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}
	realFileSize := fileInfo.Size() - offset
	
	// 检查文件大小是否匹配
	if int64(superBlock.DbSize) != realFileSize {
		fmt.Printf("Warning: db file size mismatch, expected [%d], real [%d]\n", superBlock.DbSize, realFileSize)
	}
	
	// 读取HeaderBlock
	_, err = file.Seek(offset+SuperPartLength, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to HeaderBlock position: %v", err)
	}
	
	headerBlockSize := superBlock.HeaderBlockSize
	if headerBlockSize <= 0 {
		return nil, fmt.Errorf("invalid HeaderBlockSize: %d", headerBlockSize)
	}
	
	b := make([]byte, headerBlockSize)
	bytesRead, err := file.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to read HeaderBlock: %v", err)
	}
	if bytesRead < int(headerBlockSize) {
		fmt.Printf("Warning: incomplete HeaderBlock read: %d of %d bytes\n", bytesRead, headerBlockSize)
		b = b[:bytesRead]
	}
	
	// 解析HeaderBlock
	lenEntries := int(headerBlockSize) / HeaderBlockLength
	headerSip := make([][]byte, lenEntries)
	headerPtr := make([]int32, lenEntries)
	
	idx := 0
	var dataPtr int32
	for i := 0; i < int(headerBlockSize); i += HeaderBlockLength {
		if i+16 >= len(b) {
			break
		}
		
		dataPtr = utils.GetIntLong(b, i+16)
		if dataPtr == 0 {
			break
		}
		
		sipBytes := make([]byte, 16)
		copy(sipBytes, b[i:i+16])
		headerSip[idx] = sipBytes
		headerPtr[idx] = dataPtr
		idx++
	}
	
	// 创建BtreeModeParam
	param := &BtreeModeParam{
		HeaderLength: idx,
		HeaderPtr:    headerPtr[:idx],
		HeaderSip:    headerSip[:idx],
	}
	
	return param, nil
}

// 加载地理数据映射
func loadGeoMapping(dbSearcher *DBSearcher, offset int64) error {
	file := dbSearcher.File
	endIndexPtr := dbSearcher.EndIndexPtr
	
	// 检查 endIndexPtr 是否有效
	if endIndexPtr <= 0 {
		return fmt.Errorf("invalid end index pointer: %d", endIndexPtr)
	}
	
	// 计算 ColumnSelection 的位置
	columnSelectionPtr := offset + int64(endIndexPtr) + int64(dbSearcher.IPBytesLength*2+5)
	
	// 读取 ColumnSelection
	_, err := file.Seek(columnSelectionPtr, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to column selection position: %v", err)
	}
	
	columnSelectionBytes := make([]byte, 4)
	bytesRead, err := file.Read(columnSelectionBytes)
	if err != nil {
		return fmt.Errorf("failed to read column selection: %v", err)
	}
	if bytesRead < 4 {
		return fmt.Errorf("incomplete column selection read: %d of 4 bytes", bytesRead)
	}
	
	// 设置 ColumnSelection
	dbSearcher.ColumnSelection = utils.GetIntLong(columnSelectionBytes, 0)
	fmt.Printf("Debug: Column Selection: %d\n", dbSearcher.ColumnSelection)
	
	// column selection == 0 表示不使用地理映射
	if dbSearcher.ColumnSelection == 0 {
		fmt.Println("Warning: Column Selection is 0, not using geo mapping")
		dbSearcher.GeoMapData = make([]byte, 0)
		return nil
	}
	
	// 计算地理数据起始位置 - 修正为正确位置
	// 地理数据位于 ColumnSelection 之后
	geoDataStart := columnSelectionPtr + 4
	fmt.Printf("Debug: Geo data start position: %d\n", geoDataStart)
	
	// 跳转到地理数据起始位置
	_, err = file.Seek(geoDataStart, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to geo data position: %v", err)
	}
	
	// 读取地理数据大小
	geoSizeBytes := make([]byte, 4)
	bytesRead, err = file.Read(geoSizeBytes)
	if err != nil {
		return fmt.Errorf("failed to read geo size: %v", err)
	}
	if bytesRead < 4 {
		return fmt.Errorf("incomplete geo size read: %d of 4 bytes", bytesRead)
	}
	
	geoSize := utils.GetIntLong(geoSizeBytes, 0)
	fmt.Printf("Debug: Geo map size: %d bytes\n", geoSize)
	
	// 检查地理数据大小
	if geoSize <= 0 {
		fmt.Println("Warning: No geo data available")
		dbSearcher.GeoMapData = make([]byte, 0)
		return nil
	}
	
	// 限制地理数据大小，防止内存溢出
	if geoSize > 100000000 {
		fmt.Printf("Warning: Geo data size too large (%d), limiting to 100MB\n", geoSize)
		geoSize = 100000000 // 限制为100MB
	}
	
	// 读取加密的地理数据
	encryptedGeoBytes := make([]byte, geoSize)
	bytesRead, err = file.Read(encryptedGeoBytes)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read encrypted geo data: %v", err)
	}
	
	if bytesRead < int(geoSize) {
		fmt.Printf("Warning: Read %d of %d bytes for geo data\n", bytesRead, geoSize)
		encryptedGeoBytes = encryptedGeoBytes[:bytesRead]
	}
	
	// 解密地理数据 - 使用异或操作解密，按照白皮书描述
	keyBytes, err := base64.StdEncoding.DecodeString(dbSearcher.DBKey)
	if err != nil {
		return fmt.Errorf("failed to decode key: %v", err)
	}
	
	fmt.Printf("Debug: Key length (after base64 decode): %d bytes\n", len(keyBytes))
	
	// 逐字节异或解密
	decryptedGeoBytes := make([]byte, len(encryptedGeoBytes))
	for i := 0; i < len(encryptedGeoBytes); i++ {
		decryptedGeoBytes[i] = encryptedGeoBytes[i] ^ keyBytes[i%len(keyBytes)]
	}
	
	fmt.Printf("Debug: Loaded and decrypted %d bytes of geo data\n", len(decryptedGeoBytes))
	
	// 设置地理数据
	dbSearcher.GeoMapData = decryptedGeoBytes
	return nil
}

// 初始化数据库搜索器
func InitDBSearcher(dbPath string, key string, searchType SearchType) (*DBSearcher, error) {
	// 打开数据库文件
	file, err := os.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database file: %v", err)
	}
	
	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}
	fileSize := fileInfo.Size()
	fmt.Printf("Database file size: %d bytes\n", fileSize)
	
	// 创建数据库搜索器
	dbSearcher := &DBSearcher{
		File:       file,
		SearchType: searchType,
		DBKey:      key,
	}
	
	// 解密HyperHeaderBlock
	hyperHeader, err := DecryptHyperHeaderBlock(file, key)
	if err != nil {
		file.Close()
		return nil, err
	}
	
	dbSearcher.HyperHeader = hyperHeader
	dbSearcher.DecryptedBlock = hyperHeader.DecryptedBlock
	
	// 计算文件偏移量，包括随机填充数据的大小
	offset := int64(GetHyperHeaderBlockSize(hyperHeader)) + int64(hyperHeader.DecryptedBlock.RandomSize)
	dbSearcher.FileOffset = offset
	
	// 跳过随机数据
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to seek past random data: %v", err)
	}
	
	// 读取SuperBlock
	superBytes := make([]byte, SuperPartLength)
	bytesRead, err := file.Read(superBytes)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read SuperBlock: %v", err)
	}
	if bytesRead < SuperPartLength {
		file.Close()
		return nil, fmt.Errorf("incomplete SuperBlock read: %d of %d bytes", bytesRead, SuperPartLength)
	}
	
	// 解析SuperBlock
	superBlock, err := parseSuperBlock(superBytes)
	if err != nil {
		file.Close()
		return nil, err
	}
	
	dbSearcher.SuperBlock = superBlock
	
	// 设置IP类型
	if superBlock.DbType == 0 {
		dbSearcher.IPType = int32(utils.IPV4)
		dbSearcher.IPBytesLength = 4 // IPv4为4字节
	} else {
		dbSearcher.IPType = int32(utils.IPV6)
		dbSearcher.IPBytesLength = 16 // IPv6为16字节
	}
	
	// 设置索引指针
	dbSearcher.StartIndexPtr = superBlock.StartIndexPtr
	dbSearcher.EndIndexPtr = superBlock.EndIndexPtr
	dbSearcher.HeaderBlockSize = superBlock.HeaderBlockSize
	dbSearcher.IndexLength = int32(dbSearcher.IPBytesLength*2 + 5) // 计算索引长度
	
	fmt.Printf("Debug: IPType: %d, IPBytesLength: %d\n", dbSearcher.IPType, dbSearcher.IPBytesLength)
	fmt.Printf("Debug: SuperBlock HeaderBlockSize: %d\n", superBlock.HeaderBlockSize)
	fmt.Printf("Debug: StartIndexPtr: %d, EndIndexPtr: %d\n", dbSearcher.StartIndexPtr, dbSearcher.EndIndexPtr)
	
	// 初始化B-tree模式参数，传递已解析的SuperBlock
	btreeModeParam, err := initBtreeModeParam(file, offset, superBlock)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to initialize btree mode parameters: %v", err)
	}
	
	dbSearcher.BtreeModeParam = btreeModeParam
	
	// 加载地理数据映射
	err = loadGeoMapping(dbSearcher, offset)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to load geo mapping: %v", err)
	}
	
	return dbSearcher, nil
}

// 搜索IP地址
func Search(ip string, dbSearcher *DBSearcher) (string, error) {
	if dbSearcher == nil {
		return "", fmt.Errorf("dbSearcher is nil")
	}
	
	if dbSearcher.SearchType == MEMORY {
		// 内存模式搜索
		return MemorySearch(dbSearcher, ip)
	} else if dbSearcher.SearchType == BTREE {
		// B-tree模式搜索
		return BTreeSearch(dbSearcher, ip)
	}
	
	return "", fmt.Errorf("unsupported search type")
}

// 内存模式搜索
func MemorySearch(dbSearcher *DBSearcher, ip string) (string, error) {
	// 将IP转换为uint32
	ipLong, err := ipToUint32(ip)
	if err != nil {
		return "", fmt.Errorf("invalid IP address format: %v", err)
	}
	
	fmt.Printf("Debug: Searching for IP: %s (Decimal: %d) in memory mode\n", ip, ipLong)
	
	// 懒加载方式，如果DBBin为空，则一次性读取整个数据库文件到内存
	if dbSearcher.DBBin == nil || len(dbSearcher.DBBin) == 0 {
		err = loadDBIntoMemory(dbSearcher)
		if err != nil {
			return "", fmt.Errorf("failed to load database into memory: %v", err)
		}
	}
	
	// 准备IP字节
	ipBytes := make([]byte, dbSearcher.IPBytesLength)
	if dbSearcher.IPType == int32(utils.IPV4) {
		// IPv4
		ipAddr := net.ParseIP(ip).To4()
		if ipAddr == nil {
			return "", fmt.Errorf("invalid IPv4 address: %s", ip)
		}
		copy(ipBytes, ipAddr)
	} else {
		// IPv6
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			return "", fmt.Errorf("invalid IPv6 address: %s", ip)
		}
		copy(ipBytes, ipAddr)
	}
	
	// 从内存中二分查找
	indexStart := int(dbSearcher.StartIndexPtr)
	indexEnd := int(dbSearcher.EndIndexPtr)
	indexLength := int(dbSearcher.IndexLength)
	dataStart := SuperPartLength // 索引从SuperBlock之后开始
	
	fmt.Printf("Debug: Memory search parameters - IndexStart: %d, IndexEnd: %d, IndexLength: %d\n", 
		indexStart, indexEnd, indexLength)
	
	// 二分查找
	low, high := 0, (indexEnd - indexStart) / indexLength
	found := false
	var dataPtr uint32
	var dataLen uint8
	
	for low <= high {
		mid := (low + high) / 2
		offset := indexStart + mid * indexLength
		
		if dataStart + offset + dbSearcher.IPBytesLength * 2 + 5 > len(dbSearcher.DBBin) {
			return "", fmt.Errorf("index offset out of bounds: %d", offset)
		}
		
		// 读取起始IP和结束IP
		startIP := dbSearcher.DBBin[dataStart + offset : dataStart + offset + dbSearcher.IPBytesLength]
		endIP := dbSearcher.DBBin[dataStart + offset + dbSearcher.IPBytesLength : dataStart + offset + dbSearcher.IPBytesLength * 2]
		
		// 比较IP
		if dbSearcher.IPType == int32(utils.IPV4) {
			// 对于IPv4
			startIPLong := uint32(startIP[0])<<24 | uint32(startIP[1])<<16 | uint32(startIP[2])<<8 | uint32(startIP[3])
			endIPLong := uint32(endIP[0])<<24 | uint32(endIP[1])<<16 | uint32(endIP[2])<<8 | uint32(endIP[3])
			
			if ipLong < startIPLong {
				high = mid - 1
			} else if ipLong > endIPLong {
				low = mid + 1
			} else {
				// IP在范围内
				dataPos := dataStart + offset + dbSearcher.IPBytesLength * 2
				
				// 读取数据长度和指针 - 根据白皮书规范处理
				dataLen = dbSearcher.DBBin[dataPos]
				
				// 指针是4字节小端序，但我们只使用低24位
				dataPtr = uint32(dbSearcher.DBBin[dataPos+1]) |
					uint32(dbSearcher.DBBin[dataPos+2])<<8 |
					uint32(dbSearcher.DBBin[dataPos+3])<<16
					
				fmt.Printf("Debug: Found data pointer at offset %d: len=%d, ptr=%d\n", 
					dataPos, dataLen, dataPtr)
				
				found = true
				break
			}
		} else {
			// 对于IPv6
			cmpStart := compareBytes(ipBytes, startIP, dbSearcher.IPBytesLength)
			cmpEnd := compareBytes(ipBytes, endIP, dbSearcher.IPBytesLength)
			
			if cmpStart < 0 {
				high = mid - 1
			} else if cmpEnd > 0 {
				low = mid + 1
			} else {
				// IP在范围内
				dataPos := dataStart + offset + dbSearcher.IPBytesLength * 2
				
				// 读取数据长度和指针 - 根据白皮书规范处理
				dataLen = dbSearcher.DBBin[dataPos]
				
				// 指针是4字节小端序，但我们只使用低24位
				dataPtr = uint32(dbSearcher.DBBin[dataPos+1]) |
					uint32(dbSearcher.DBBin[dataPos+2])<<8 |
					uint32(dbSearcher.DBBin[dataPos+3])<<16
				
				fmt.Printf("Debug: Found data pointer at offset %d: len=%d, ptr=%d\n", 
					dataPos, dataLen, dataPtr)
					
				found = true
				break
			}
		}
	}
	
	if !found {
		return "IP not found", nil
	}
	
	// 检查数据指针和长度
	if dataPtr == 0 || dataLen == 0 {
		return "", fmt.Errorf("invalid data pointer or length: ptr=%d, len=%d", dataPtr, dataLen)
	}
	
	// 检查指针是否有效
	if int(dataPtr) >= len(dbSearcher.GeoMapData) {
		return "", fmt.Errorf("geo pointer out of bounds: ptr=%d, len=%d, dataSize=%d",
			dataPtr, dataLen, len(dbSearcher.GeoMapData))
	}

	if int(dataPtr) + int(dataLen) > len(dbSearcher.GeoMapData) {
		return "", fmt.Errorf("geo data exceeds buffer bounds: ptr=%d, len=%d, dataSize=%d",
			dataPtr, dataLen, len(dbSearcher.GeoMapData))
	}
	
	// 从数据库二进制文件中读取数据
	data := make([]byte, dataLen)
	
	// 从内存或文件中复制数据
	if dbSearcher.SearchType == MEMORY {
		copy(data, dbSearcher.DBBin[dataPtr:dataPtr+uint32(dataLen)])
	} else {
		// 如果不是内存模式，从文件读取
		_, err := dbSearcher.File.Seek(int64(dataPtr)+dbSearcher.FileOffset, io.SeekStart)
		if err != nil {
			return "", fmt.Errorf("failed to seek to data position: %v", err)
		}
		_, err = dbSearcher.File.Read(data)
		if err != nil {
			return "", fmt.Errorf("failed to read data: %v", err)
		}
	}
	
	// 获取地理信息
	geoData, err := GetActualGeo(dbSearcher.GeoMapData, dbSearcher.ColumnSelection, int(dataPtr), int(dataLen), data, int(dataLen))
	if err != nil {
		return "", fmt.Errorf("failed to get geo data: %v", err)
	}
	
	// 清理结果字符串，移除非打印字符
	geoData = cleanString(geoData)
	
	return geoData, nil
}

// 将数据库文件加载到内存
func loadDBIntoMemory(dbSearcher *DBSearcher) error {
	// 获取文件大小
	fileInfo, err := dbSearcher.File.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}
	fileSize := fileInfo.Size()
	
	fmt.Printf("Loading database into memory (size: %d bytes)...\n", fileSize - dbSearcher.FileOffset)
	
	// 从文件偏移位置开始读取数据
	_, err = dbSearcher.File.Seek(dbSearcher.FileOffset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to file offset: %v", err)
	}
	
	// 分配内存
	dbSearcher.DBBin = make([]byte, fileSize - dbSearcher.FileOffset)
	
	// 读取数据
	bytesRead, err := dbSearcher.File.Read(dbSearcher.DBBin)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file into memory: %v", err)
	}
	
	if int64(bytesRead) < fileSize - dbSearcher.FileOffset {
		fmt.Printf("Warning: Read %d of %d bytes into memory\n", bytesRead, fileSize - dbSearcher.FileOffset)
		dbSearcher.DBBin = dbSearcher.DBBin[:bytesRead]
	}
	
	fmt.Printf("Database loaded into memory successfully (%d bytes)\n", bytesRead)
	return nil
}

// 将IP转换为uint32
func ipToUint32(ipstr string) (uint32, error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP address: %s", ipstr)
	}
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("not an IPv4 address: %s", ipstr)
	}
	
	var result uint32
	result = uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
	return result, nil
}

// B-tree模式搜索
func BTreeSearch(dbSearcher *DBSearcher, ip string) (string, error) {
	// 检查IP类型
	ipLong, err := ipToUint32(ip)
	if err != nil {
		return "", fmt.Errorf("invalid IP address format: %v", err)
	}
	
	fmt.Printf("Debug: Searching for IP: %s (Decimal: %d) in btree mode\n", ip, ipLong)
	
	// 准备IP字节
	ipBytes := make([]byte, dbSearcher.IPBytesLength)
	if dbSearcher.IPType == int32(utils.IPV4) {
		// IPv4
		ipAddr := net.ParseIP(ip).To4()
		if ipAddr == nil {
			return "", fmt.Errorf("invalid IPv4 address: %s", ip)
		}
		copy(ipBytes, ipAddr)
	} else {
		// IPv6
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			return "", fmt.Errorf("invalid IPv6 address: %s", ip)
		}
		copy(ipBytes, ipAddr)
	}
	
	// 二分查找
	param := dbSearcher.BtreeModeParam
	l, h := 0, param.HeaderLength-1
	sptr, eptr := int32(0), int32(0)
	
	for l <= h {
		m := (l + h) / 2
		
		// 比较IP
		cmp := compareBytes(ipBytes, param.HeaderSip[m], dbSearcher.IPBytesLength)
		if cmp < 0 {
			h = m - 1
		} else if cmp > 0 {
			l = m + 1
		} else {
			sptr = param.HeaderPtr[m-1]
			eptr = param.HeaderPtr[m]
			break
		}
	}
	
	if l > h {
		if l < param.HeaderLength {
			sptr = param.HeaderPtr[l-1]
			eptr = param.HeaderPtr[l]
		} else if h >= 0 && h+1 < param.HeaderLength {
			sptr = param.HeaderPtr[h]
			eptr = param.HeaderPtr[h+1]
		} else { // 搜索到最后一个头部行，可能在最后一个索引块
			sptr = param.HeaderPtr[param.HeaderLength-1]
			blockLen := int32(dbSearcher.IPBytesLength*2 + 5) // 索引块长度
			eptr = sptr + blockLen
		}
	}
	
	if sptr == 0 {
		return "IP not found", nil
	}
	
	// 准备索引缓冲区
	blockLen := eptr - sptr
	blen := dbSearcher.IndexLength
	
	// 从文件读取索引
	_, err = dbSearcher.File.Seek(int64(sptr)+dbSearcher.FileOffset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("failed to seek to index position: %v", err)
	}
	
	indexBuffer := make([]byte, blockLen)
	bytesRead, err := dbSearcher.File.Read(indexBuffer)
	if err != nil {
		return "", fmt.Errorf("failed to read index buffer: %v", err)
	}
	if bytesRead < int(blockLen) {
		return "", fmt.Errorf("incomplete index buffer read: %d of %d bytes", bytesRead, blockLen)
	}
	
	// 二分查找索引块
	l, h = 0, int(blockLen/blen)
	var dataPtr uint32
	var dataLen uint8
	found := false
	
	for l <= h {
		m := (l + h) / 2
		offset := m * int(blen)
		
		if offset+int(dbSearcher.IPBytesLength)*2+5 > len(indexBuffer) {
			break
		}
		
		// 读取起始IP和结束IP
		startIP := indexBuffer[offset:offset+dbSearcher.IPBytesLength]
		endIP := indexBuffer[offset+dbSearcher.IPBytesLength:offset+dbSearcher.IPBytesLength*2]
		
		// 使用统一的比较方法，无论是IPv4还是IPv6
		cmpStart := compareBytes(ipBytes, startIP, dbSearcher.IPBytesLength)
		cmpEnd := compareBytes(ipBytes, endIP, dbSearcher.IPBytesLength)
		
		if cmpStart >= 0 && cmpEnd <= 0 {
			// IP在这个块中
			dataPos := offset + dbSearcher.IPBytesLength*2
			
			// 获取4字节的数据指针和1字节的数据长度
			dataPtr = uint32(utils.GetIntLong(indexBuffer, dataPos))
			dataLen = uint8(utils.GetInt1(indexBuffer, dataPos+4))
			
			fmt.Printf("Debug: Found data pointer in btree mode: len=%d, ptr=%d\n", dataLen, dataPtr)
			
			found = true
			break
		} else if cmpStart < 0 {
			// IP小于此块，在左半部分搜索
			h = m - 1
		} else {
			// IP大于此块，在右半部分搜索
			l = m + 1
		}
	}
	
	if !found {
		return "IP not found", nil
	}
	
	// 检查数据指针和长度
	if dataPtr == 0 || dataLen == 0 {
		return "", fmt.Errorf("invalid data pointer or length: ptr=%d, len=%d", dataPtr, dataLen)
	}
	
	// 读取地理位置数据
	if int(dataPtr) >= len(dbSearcher.GeoMapData) {
		return "", fmt.Errorf("geo pointer out of bounds: ptr=%d, len=%d, dataSize=%d",
			dataPtr, dataLen, len(dbSearcher.GeoMapData))
	}
	
	if int(dataPtr)+int(dataLen) > len(dbSearcher.GeoMapData) {
		return "", fmt.Errorf("geo data exceeds buffer bounds: ptr=%d, len=%d, dataSize=%d",
			dataPtr, dataLen, len(dbSearcher.GeoMapData))
	}
	
	// 从文件读取数据
	data := make([]byte, dataLen)
	
	// 从文件中读取数据
	_, err = dbSearcher.File.Seek(int64(dataPtr)+dbSearcher.FileOffset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("failed to seek to data position: %v", err)
	}
	_, err = dbSearcher.File.Read(data)
	if err != nil {
		return "", fmt.Errorf("failed to read data: %v", err)
	}
	
	// 获取地理信息
	geoData, err := GetActualGeo(dbSearcher.GeoMapData, dbSearcher.ColumnSelection, int(dataPtr), int(dataLen), data, int(dataLen))
	if err != nil {
		return "", fmt.Errorf("failed to get geo data: %v", err)
	}
	
	return geoData, nil
}

// 比较字节数组
func compareBytes(bytes1, bytes2 []byte, length int) int {
	for i := 0; i < length; i++ {
		if i >= len(bytes1) || i >= len(bytes2) {
			break
		}
		
		// 直接比较无符号字节值，这符合Go的byte(uint8)性质
		// 并且适用于IP地址比较（IP地址字节通常被视为无符号值）
		if bytes1[i] < bytes2[i] {
			return -1
		} else if bytes1[i] > bytes2[i] {
			return 1
		}
	}
	return 0
}

// 清理字符串
func cleanString(s string) string {
	var result strings.Builder
	for _, c := range s {
		if c >= 32 && c <= 126 { // 可打印ASCII字符
			result.WriteRune(c)
		}
	}
	return result.String()
}

// 关闭数据库搜索器
func CloseDBSearcher(dbSearcher *DBSearcher) {
	if dbSearcher == nil {
		return
	}
	if dbSearcher.File != nil {
		dbSearcher.File.Close()
	}
}

// 打印数据库信息
func Info(dbSearcher *DBSearcher) {
	fmt.Println("\n=========== Database Information ===========")
	fmt.Printf("IP Type: %d\n", dbSearcher.IPType)
	fmt.Printf("IP Bytes Length: %d\n", dbSearcher.IPBytesLength)
	fmt.Printf("Start Index Pointer: %d\n", dbSearcher.StartIndexPtr)
	fmt.Printf("End Index Pointer: %d\n", dbSearcher.EndIndexPtr)
	fmt.Printf("Header Block Size: %d\n", dbSearcher.HeaderBlockSize)
	fmt.Printf("Search Type: %s\n", searchTypeToString(dbSearcher.SearchType))
	fmt.Printf("BTree Header Length: %d\n", dbSearcher.BtreeModeParam.HeaderLength)
	fmt.Printf("Geo Map Data Size: %d bytes\n", len(dbSearcher.GeoMapData))
	fmt.Println("===========================================")
	fmt.Println("\nEnter IP addresses to look up their geographical locations.")
	fmt.Println("Type 'q' or 'quit' to exit.")
}

// 将搜索类型转换为字符串
func searchTypeToString(searchType SearchType) string {
	switch searchType {
	case MEMORY:
		return "Memory"
	case BTREE:
		return "B-tree"
	default:
		return "Unknown"
	}
}

// 解密数据
func Decrypt(encryptedBytes []byte, key string) []byte {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		fmt.Printf("Error decoding key: %v\n", err)
		return encryptedBytes
	}
	
	result := make([]byte, len(encryptedBytes))
	for i := 0; i < len(encryptedBytes); i++ {
		result[i] = encryptedBytes[i] ^ keyBytes[i%len(keyBytes)]
	}
	return result
}

// 获取地理信息
func GetActualGeo(geoMapData []byte, columnSelection int32, geoPtr int, geoLen int, data []byte, dataLen int) (string, error) {
	if len(geoMapData) == 0 {
		return "No geo data available", nil
	}
	
	if geoPtr+geoLen > len(geoMapData) {
		return "", fmt.Errorf("Geo pointer out of bounds: ptr=%d, len=%d, dataSize=%d", geoPtr, geoLen, len(geoMapData))
	}
	
	// 使用msgpack直接解码，类似Java实现
	dec := msgpack.NewDecoder(bytes.NewReader(data))
	
	// 解包第一个值：geoPosMixSize (uint64)
	geoPosMixSize, err := dec.DecodeUint64()
	if err != nil {
		return "", fmt.Errorf("failed to decode geoPosMixSize: %v", err)
	}
	
	// 解包第二个值：otherData (string)
	otherData, err := dec.DecodeString()
	if err != nil {
		return "", fmt.Errorf("failed to decode otherData: %v", err)
	}
	
	// 如果geoPosMixSize为0，直接返回otherData
	if geoPosMixSize == 0 {
		return otherData, nil
	}
	
	// 提取地理指针和长度
	dataLen = int((geoPosMixSize >> 24) & 0xFF)
	dataPtr := int(geoPosMixSize & 0x00FFFFFF)
	
	// 检查索引是否有效
	if dataPtr < 0 || dataPtr+dataLen > len(geoMapData) {
		return otherData, nil // 索引无效时返回otherData
	}
	
	// 从geoMapData中读取地理数据
	regionData := geoMapData[dataPtr : dataPtr+dataLen]
	
	// 使用新的解码器解包地理数据
	geoDec := msgpack.NewDecoder(bytes.NewReader(regionData))
	
	// 读取数组头，获取列数
	columnNumber, err := geoDec.DecodeArrayLen()
	if err != nil {
		return otherData, fmt.Errorf("failed to decode column array: %v", err)
	}
	
	// 构建结果
	var sb strings.Builder
	
	// 遍历所有列
	for i := 0; i < columnNumber; i++ {
		// 检查列是否被选中
		columnSelected := (columnSelection >> (i + 1) & 1) == 1
		
		// 解码列值（字符串）
		value, err := geoDec.DecodeString()
		if err != nil {
			return otherData, fmt.Errorf("failed to decode column %d: %v", i, err)
		}
		
		// 处理空值
		if value == "" {
			value = "null"
		}
		
		// 如果列被选中，添加到结果中
		if columnSelected {
			sb.WriteString(value)
			sb.WriteString("\t")
		}
	}
	
	// 将地理数据和其他数据合并
	return sb.String() + otherData, nil
}

// 解包MessagePack数据
func Unpack(geoMapData []byte, columnSelection int32, data []byte) (string, error) {
	return GetActualGeo(geoMapData, columnSelection, 0, 0, data, len(data))
} 