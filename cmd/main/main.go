package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cz88/czdb-search-golang/pkg/db"
)

func main() {
	// 定义命令行参数
	dbPath := flag.String("p", "", "Path to CZDB database file")
	key := flag.String("k", "", "Base64 encoded key for decryption")
	mode := flag.String("m", "btree", "Search mode: 'memory' or 'btree'")

	// 解析命令行参数
	flag.Parse()

	// 检查必要参数
	if *dbPath == "" || *key == "" {
		fmt.Println("Error: Database path and key are required")
		flag.Usage()
		os.Exit(1)
	}

	// 确定搜索模式
	var searchType db.SearchType
	if strings.ToLower(*mode) == "memory" {
		searchType = db.MEMORY
		fmt.Println("Using Memory search mode")
	} else {
		searchType = db.BTREE
		fmt.Println("Using B-tree search mode")
	}

	// 初始化数据库搜索器
	fmt.Printf("Initializing database searcher with file: %s\n", *dbPath)
	dbSearcher, err := db.InitDBSearcher(*dbPath, *key, searchType)
	if err != nil {
		fmt.Printf("Error initializing database searcher: %v\n", err)
		os.Exit(1)
	}
	defer db.CloseDBSearcher(dbSearcher)

	// 打印数据库信息
	db.Info(dbSearcher)

	// 启动交互式查询
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nEnter IP address (or 'q' to quit): ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "q" || input == "quit" {
			break
		}

		if input == "" {
			continue
		}

		// 查询IP地址
		result, err := db.Search(input, dbSearcher)
		if err != nil {
			fmt.Printf("Error searching for IP %s: %v\n", input, err)
			continue
		}

		fmt.Printf("Result for %s: %s\n", input, result)
	}

	fmt.Println("Exiting...")
} 