# CZDB Search Golang

一个用于搜索CZDB格式IP数据库的Go语言实现。

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
├── go.mod              # Go模块定义
└── README.md           # 项目说明
```

## 编译和运行

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