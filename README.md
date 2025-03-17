# CZDB Search Golang

[![Go Reference](https://pkg.go.dev/badge/github.com/tagphi/czdb-search-golang.svg)](https://pkg.go.dev/github.com/tagphi/czdb-search-golang)
[![Go Report Card](https://goreportcard.com/badge/github.com/tagphi/czdb-search-golang)](https://goreportcard.com/report/github.com/tagphi/czdb-search-golang)
[![License](https://img.shields.io/github/license/tagphi/czdb-search-golang)](https://github.com/tagphi/czdb-search-golang/blob/main/LICENSE)

一个用于搜索CZDB格式IP数据库的Go语言实现。提供高效的IP地址查询功能，支持内存模式和B树模式两种查询方式。

## 安装

使用Go模块安装这个库:

```bash
go get github.com/tagphi/czdb-search-golang
```

## 作为库使用

在您的Go代码中引入并使用此库:

```go
package main

import (
	"fmt"
	"github.com/tagphi/czdb-search-golang/pkg/db"
)

func main() {
	// 初始化数据库搜索器，使用内存模式
	dbSearcher, err := db.InitDBSearcher("./data.db", "mykey", db.MEMORY)
	if err != nil {
		fmt.Printf("初始化数据库搜索器失败: %v\n", err)
		return
	}
	defer db.CloseDBSearcher(dbSearcher)

	// 搜索IP地址
	region, err := db.Search("8.8.8.8", dbSearcher)
	if err != nil {
		fmt.Printf("搜索失败: %v\n", err)
	} else {
		fmt.Printf("区域: %s\n", region)
	}
}
```

更多示例请参考 [examples](./examples) 目录。

## 特性

- **两种搜索模式**：支持内存模式和B树模式
- **高性能**：内存模式下性能极高，适合高并发场景
- **简单API**：提供易于使用的API接口
- **线程安全**：内存模式下完全线程安全

## 项目结构

```
czdb-search-golang/
├── cmd/
│   └── main/
│       └── main.go     # 主程序入口
├── pkg/
│   ├── db/             # 数据库核心功能
│   │   ├── db_searcher.go         # 数据库搜索器实现
│   │   ├── decrypted_block.go     # 解密块定义和解密功能
│   │   └── hyper_header_block.go  # 头部块定义和解析功能
│   └── utils/          # 工具函数
│       └── byte_utils.go          # 字节处理工具函数
├── examples/           # 使用示例
├── go.mod              # Go模块定义
└── README.md           # 项目说明
```

## 编译和运行命令行工具

编译项目：

```bash
go build -o cz88-search ./cmd/main/main.go
```

运行程序：

```bash
./cz88-search -p <CZDB数据库路径> -k <Base64编码的密钥> -m <搜索模式>
```

参数说明：
- `-p`: CZDB数据库文件路径
- `-k`: Base64编码的密钥
- `-m`: 搜索模式，可选值为 `btree` 或 `memory`，默认为 `btree`

## 使用示例

```bash
./cz88-search -p /path/to/ipv4.czdb -k 6ULQJvr05njRVczBC4omxA== -m btree
```

交互式使用：
1. 程序启动后会显示数据库基本信息
2. 输入IP地址进行查询
3. 输入 `q` 或 `quit` 退出程序

## 线程安全性

库支持两种查询方式：Memory和Btree。

- Memory模式：在此模式下，整个数据库被加载到内存中，是线程安全的。
- Btree模式：在此模式下，数据库文件会在查询时被读取，不是线程安全的。这主要是因为DBSearcher结构体中保存了文件句柄，如果多个线程同时访问会导致文件指针错乱。

建议：
- 对于需要高性能的应用，使用Memory模式
- 如果需要在多线程环境中使用Btree模式，每个线程应该创建自己的DBSearcher实例

## 测试

运行单元测试：

```bash
go test ./...
```

运行带覆盖率的测试：

```bash
go test -cover ./...
```

## CZDB格式规范

CZDB文件格式由以下几个部分组成：
1. 超级头部块 (HyperHeaderBlock)
2. 加密块 (EncryptedBlock)
3. 随机填充数据
4. 超级块 (SuperBlock)
5. 头部块 (HeaderBlock)
6. 索引数据
7. 地理数据

详细的格式规范可参考相关文档。

## 版本管理

此库遵循语义化版本控制(Semantic Versioning)规范。您可以在Go模块中指定版本:

```go
require github.com/tagphi/czdb-search-golang v1.0.0
```

## 贡献指南

欢迎提交问题和贡献代码！请确保您的代码符合Go的代码规范，并且添加了适当的测试。

## 许可证

Apache License 2.0 