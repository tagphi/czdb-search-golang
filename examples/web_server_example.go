package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/tagphi/czdb-search-golang/pkg/db"
)

// 响应结构
type Response struct {
	IP     string `json:"ip"`
	Region string `json:"region"`
	Error  string `json:"error,omitempty"`
}

var dbSearcher *db.DBSearcher

func main() {
	// 初始化数据库搜索器，使用内存模式以获得最佳性能
	var err error
	dbSearcher, err = db.InitDBSearcher("./data.db", "your-key-here", db.MEMORY)
	if err != nil {
		log.Fatalf("初始化数据库搜索器失败: %v\n", err)
	}
	defer db.CloseDBSearcher(dbSearcher)

	// 设置API路由
	http.HandleFunc("/api/ip/", lookupHandler)
	http.HandleFunc("/api/health", healthCheckHandler)

	// 启动服务器
	port := 8080
	fmt.Printf("IP查询服务启动在 http://localhost:%d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("服务器启动失败: %v\n", err)
	}
}

// 查询处理函数
func lookupHandler(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取IP地址
	ip := r.URL.Path[len("/api/ip/"):]
	if ip == "" {
		http.Error(w, "请提供IP地址", http.StatusBadRequest)
		return
	}

	// 查询IP地址
	region, err := db.Search(ip, dbSearcher)
	
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	
	// 准备响应
	response := Response{
		IP: ip,
	}
	
	if err != nil {
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response.Region = region
	}
	
	// 发送JSON响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// 健康检查处理函数
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "ok"}`)
} 