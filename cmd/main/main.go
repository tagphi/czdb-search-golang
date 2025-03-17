package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tagphi/czdb-search-golang/pkg/db"
	"github.com/tagphi/czdb-search-golang/pkg/utils"
)

func main() {
	// 定义命令行参数
	dbPath := flag.String("p", "", "Path to CZDB database file")
	key := flag.String("k", "", "Base64 encoded key for decryption")
	mode := flag.String("m", "btree", "Search mode: 'memory' or 'btree'")
	debug := flag.Bool("debug", false, "Enable debug output")
	logFile := flag.String("log", "", "Log file for debug output (default: stdout)")

	// 解析命令行参数
	flag.Parse()

	// 设置调试模式
	utils.SetDebugEnabled(*debug)

	// 如果指定了日志文件，则将调试输出重定向到文件
	if *debug && *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error opening log file: %v\n", err)
			fmt.Println("Debug output will be sent to stdout")
		} else {
			utils.SetDebugOutput(file)
			defer file.Close()
			fmt.Printf("Debug output will be written to %s\n", *logFile)
		}
	}

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
	if *debug {
		fmt.Println("Debug mode enabled")
	}
	
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