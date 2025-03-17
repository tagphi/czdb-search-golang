package main

import (
	"fmt"
	"log"

	"github.com/tagphi/czdb-search-golang/pkg/db"
)

func main() {
	// 初始化数据库搜索器，使用内存模式
	// 修改数据库文件路径和密钥为您的实际值
	dbSearcher, err := db.InitDBSearcher("./data.db", "your-key-here", db.MEMORY)
	if err != nil {
		log.Fatalf("初始化数据库搜索器失败: %v\n", err)
	}
	defer db.CloseDBSearcher(dbSearcher)

	// 打印数据库信息
	fmt.Println("数据库信息:")
	db.Info(dbSearcher)

	// 搜索单个IP地址
	ip := "8.8.8.8"
	region, err := db.Search(ip, dbSearcher)
	if err != nil {
		log.Printf("搜索IP %s 失败: %v\n", ip, err)
	} else {
		fmt.Printf("IP: %s, 区域: %s\n", ip, region)
	}

	// 搜索多个IP地址
	ips := []string{"114.114.114.114", "1.1.1.1", "223.5.5.5"}
	for _, ip := range ips {
		region, err := db.Search(ip, dbSearcher)
		if err != nil {
			log.Printf("搜索IP %s 失败: %v\n", ip, err)
		} else {
			fmt.Printf("IP: %s, 区域: %s\n", ip, region)
		}
	}
} 