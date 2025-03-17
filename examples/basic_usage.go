package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tagphi/czdb-search-golang/pkg/db"
)

func main() {
	// 从环境变量中获取数据库路径和密钥
	dbPath := os.Getenv("CZDB_PATH")
	dbKey := os.Getenv("CZDB_KEY")

	// 如果环境变量未设置，使用默认值
	if dbPath == "" {
		dbPath = "./data.db"
		fmt.Println("未设置CZDB_PATH环境变量，使用默认值:", dbPath)
	}
	if dbKey == "" {
		dbKey = "your-key-here"
		fmt.Println("未设置CZDB_KEY环境变量，使用默认值:", dbKey)
	}

	// 初始化数据库搜索器，使用内存模式
	fmt.Printf("使用数据库文件: %s\n", dbPath)
	dbSearcher, err := db.InitDBSearcher(dbPath, dbKey, db.MEMORY)
	if err != nil {
		log.Fatalf("初始化数据库搜索器失败: %v\n", err)
	}
	defer db.CloseDBSearcher(dbSearcher)

	// 打印数据库信息
	fmt.Println("\n数据库信息:")
	db.Info(dbSearcher)

	// 搜索单个IP地址
	ip := "8.8.8.8"
	region, err := db.Search(ip, dbSearcher)
	if err != nil {
		log.Printf("搜索IP %s 失败: %v\n", ip, err)
	} else {
		fmt.Printf("\nIP: %s, 区域: %s\n", ip, region)
	}

	// 搜索多个IP地址
	fmt.Println("\n批量搜索示例:")
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