# CZDB Go语言库示例

本目录包含了如何使用CZDB Go语言库的示例代码。

## 示例列表

1. **basic_usage.go**: 基本用法示例，演示如何初始化数据库搜索器并查询IP地址。
2. **web_server_example.go**: Web服务器示例，演示如何在Web应用中集成IP查询功能。

## 运行示例

要运行这些示例，首先确保您有CZDB数据库文件和对应的密钥。

### 1. 准备数据库文件

将CZDB数据库文件放在适当的位置，并记下其路径。

### 2. 修改示例代码

在运行示例前，请修改示例代码中的数据库文件路径和密钥：

```go
dbSearcher, err := db.InitDBSearcher("./data.db", "your-key-here", db.MEMORY)
```

将 `"./data.db"` 替换为您的数据库文件路径，将 `"your-key-here"` 替换为您的密钥。

### 3. 运行示例

使用以下命令运行示例：

```bash
# 运行基本用法示例
go run basic_usage.go

# 运行Web服务器示例
go run web_server_example.go
```

## Web服务器示例说明

Web服务器示例会启动一个HTTP服务器，提供IP地址查询API。启动后，可以通过以下URL查询IP地址：

```
http://localhost:8080/api/ip/{ip地址}
```

例如：`http://localhost:8080/api/ip/8.8.8.8`

服务器会返回JSON格式的结果：

```json
{
  "ip": "8.8.8.8",
  "region": "美国"
}
```

健康检查接口：

```
http://localhost:8080/api/health
``` 