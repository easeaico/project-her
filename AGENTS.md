# AGENTS.md - AI 编码代理开发指南

本文档为在 **Project Her** 代码库中工作的 AI 代理提供编码规范与常用命令。本项目是基于 Go、Google ADK 和 PostgreSQL（pgvector）的 AI 伴侣后端服务。

---

## 构建、静态检查（Lint）与测试命令

### 构建
```bash
# 构建所有包
go build ./...

# 构建主二进制
go build -o bin/project-her cmd/platform/main.go

# 清理并重建
rm -rf bin/ && go build -o bin/project-her cmd/platform/main.go
```

### 依赖管理
```bash
# 下载依赖
go mod download

# 同步依赖（添加/移除 import 后）
go mod tidy

# 校验依赖
go mod verify
```

### 静态检查（Lint）与格式化
```bash
# 格式化所有 Go 文件
gofmt -w .

# 检查格式（不修改文件）
gofmt -l .

# 静态分析
go vet ./...

# 高级 lint（需安装 golangci-lint）
golangci-lint run
```

### 测试
```bash
# 运行所有测试
go test ./...

# 运行测试并输出详细信息
go test -v ./...

# 运行单个测试
go test -v -run TestFunctionName ./path/to/package

# 运行特定包的测试
go test -v ./internal/agent

# 运行测试并生成覆盖率
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行测试并开启竞态检测
go test -race ./...
```

**注意**：当前项目**没有测试文件**。创建测试时：
- 测试文件以 `_test.go` 结尾（如 `roleplay_test.go`）
- 与被测代码放在同一目录
- 使用表驱动测试覆盖主要场景

---

## 代码风格规范

### 1. 导入（Import）组织

导入必须分为**三个块**并用空行分隔：

```go
import (
    // 块 1：标准库
    "context"
    "fmt"
    "log/slog"

    // 块 2：外部依赖
    "github.com/pgvector/pgvector-go"
    "google.golang.org/adk/agent"
    "gorm.io/gorm"

    // 块 3：内部包
    "github.com/easeaico/project-her/internal/config"
    "github.com/easeaico/project-her/internal/types"
)
```

**规则**：
- 名称冲突时使用显式别名（如 `internalagent "github.com/.../internal/agent"`）
- 每个块内按字母顺序排序
- 运行 `gofmt` 自动整理

### 2. 命名约定

| 元素 | 约定 | 示例 |
|---------|-----------|---------|
| **Packages** | 短小、小写、单数 | `agent`, `config`, `memory` |
| **Exported Structs** | PascalCase | `MemoryRepo`, `Config`, `Character` |
| **Unexported Structs** | camelCase | `memoryModel`, `openaiModel` |
| **Interfaces** | PascalCase，能力命名 | `CharacterRepo`, `Embedder` |
| **Constructors** | `New<Type>` 模式 | `NewMemoryRepo(db)`, `NewEmbedder(...)` |
| **Variables** | 简短、上下文明确 | `ctx`, `cfg`, `err`, `db` |
| **Constants** | PascalCase 或 UPPER_SNAKE | `MemoryTypeChat`, `TOP_K` |

### 3. 错误处理

**始终**：
- 立即检查错误：`if err != nil`
- 添加上下文并包装：`fmt.Errorf("description: %w", err)`
- 使用 `%w` 包装（支持 `errors.Is`/`errors.As`）
- 将错误返回给调用方而非直接日志（回调/协程除外）

**在 `main.go` 启动逻辑**：
```go
if err != nil {
    log.Fatalf("failed to connect to database: %v", err)
}
```

**在 internal 包中**：
```go
if err != nil {
    return nil, fmt.Errorf("failed to create grok model: %w", err)
}
```

**在回调/后台协程中**：
```go
if err != nil {
    slog.Error("failed to generate image", "error", err.Error())
    return fallbackValue, nil
}
```

**禁止**：
- 用 `_` 忽略错误（除非有注释说明原因）
- 使用 `panic()`（除非不可恢复）
- 使用类型断言/忽略指令压制错误（如 `as any`）

### 4. 日志

**使用 `log/slog` 进行结构化日志**：

```go
import "log/slog"

// 带上下文的错误日志
slog.Error("failed to close stream", "error", err.Error())

// 带键值对的警告日志
slog.Warn("high memory usage", "allocated_mb", allocatedMB, "threshold_mb", thresholdMB)

// Info 日志
slog.Info("agent initialized", "model", modelName, "character_id", charID)
```

**规则**：
- 使用键值对表示结构化数据
- 避免用 `fmt.Printf` 记录日志（仅用于用户输出）
- 在 `cmd/platform/main.go` 中用 `log.Fatalf` 处理启动致命错误

### 5. 类型使用

**上下文（Context）**：
```go
// I/O 或耗时函数始终将 Context 置于首参
func (r *MemoryRepo) AddMemory(ctx context.Context, mem types.Memory) error
```

**指针 vs 值**：
- **使用指针**：
  - 服务结构体（`*MemoryRepo`、`*Config`、`*gorm.DB`）
  - 较大配置对象
  - 需要可变或可空的类型
- **使用值**：
  - 小型 DTO（`types.Memory`、`types.Character`）
  - 不可变数据结构

**接口（Interfaces）**：
- 在**使用方包**定义接口，而不是实现包
- 例如：`CharacterRepo` 定义在 `internal/agent/roleplay.go`，而不是 `internal/repository/`

**数据库模型（Database Models）**：
- 数据库模型（带 GORM tags）与领域类型分离
- 提供转换函数：`func memoryFromModel(model memoryModel) types.Memory`

类型说明：
- 用 `any` 替代 `interface{}`

### 6. 函数签名

**构造函数模式**：
```go
// 依赖型结构体返回指针与 error
func NewMemoryRepo(db *gorm.DB) *MemoryRepo

// 值类型返回结构体与 error
func NewEmbedder(ctx context.Context, apiKey, model string) (*Embedder, error)
```

**服务方法**：
```go
// Context 置首，其后参数，返回 (result, error)
func (r *MemoryRepo) SearchSimilar(
    ctx context.Context,
    userID, appName, memoryType string,
    embedding []float32,
    topK int,
    threshold float64,
) ([]types.RetrievedMemory, error)
```

### 7. 代码组织

**项目结构**：
```
project-her/
├── cmd/platform/          # 应用入口（main.go）
├── internal/
│   ├── agent/            # 业务逻辑（角色扮演、记忆代理）
│   ├── config/           # 配置加载
│   ├── emotion/          # 情感分析与状态机
│   ├── memory/           # 记忆服务（嵌入、RAG）
│   ├── models/           # LLM 适配器（OpenAI、xAI）
│   ├── prompt/           # Prompt 构建逻辑
│   ├── repository/       # 数据访问层（GORM）
│   ├── types/            # 共享领域类型
│   └── utils/            # 工具函数
├── migrations/           # SQL 迁移脚本
└── docs/                 # 文档
```

**依赖注入**：
- 通过构造函数显式传递依赖
- 在 `main.go` 进行装配
- 避免全局变量

### 8. 数据库模式

**GORM 使用**：
```go
// 所有查询使用 WithContext
r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&records)

// 使用参数化查询（禁止字符串拼接）
query := r.db.WithContext(ctx).Where("id = ?", id)  // ✓ Safe
// query := r.db.Where("id = " + id)  // ✗ SQL injection risk
```

**事务**：
```go
err := r.db.Transaction(func(tx *gorm.DB) error {
    // Perform operations with tx
    return nil
})
```

### 9. 注释与文档

**仅在以下情况添加注释**：
- 解释复杂算法或业务逻辑
- 为公开 API（导出函数/类型）写说明
- 解释不明显的行为或边界条件
- 警示安全/性能影响

**避免冗余注释**：
```go
// Bad: 显而易见的注释
// Get user by ID
func GetUserByID(id int) (*User, error)

// Good: 自解释代码
func GetUserByID(id int) (*User, error)
```

**包文档**：
```go
// Package agent provides agent initialization and business logic
// for the AI companion roleplay system.
package agent
```

### 10. 测试指南（当新增测试时）

**文件结构**：
```go
// roleplay_test.go
package agent

import (
    "testing"
)

func TestNewRolePlayAgent(t *testing.T) {
    // 表驱动测试
    tests := []struct {
        name    string
        config  *config.Config
        wantErr bool
    }{
        // 测试用例
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

**Mock（Mocking）**：
- 依赖使用接口（已具备）
- 在 `*_test.go` 中创建测试实现

---

## 环境配置

**必需环境变量**（见 `.env.example`）：
- `GOOGLE_API_KEY`: Google AI API 密钥
- `XAI_API_KEY`: xAI (Grok) API 密钥  
- `DATABASE_URL`: PostgreSQL 连接字符串

**可选**（默认值见 `internal/config/config.go`）：
- `CHAT_MODEL`（默认：`grok-4-fast`）
- `MEMORY_MODEL`（默认：`gemini-2.0-flash`）
- `EMBEDDING_MODEL`（默认：`text-embedding-004`）
- `IMAGE_MODEL`（默认：`gemini-2.0-flash-exp`）
- `TOP_K`（默认：`5`）
- `SIMILARITY_THRESHOLD`（默认：`0.7`）
- `MEMORY_TRUNK_SIZE`（默认：`100`）
- `ASPECT_RATIO`（默认：`9:16`）

---

## 常见任务

### 新增仓储（Repository）
1. 在使用方包定义接口
2. 在 `internal/repository/` 创建实现
3. 按 `NewXxxRepo(db *gorm.DB)` 模式添加构造函数
4. 在 `cmd/platform/main.go` 完成装配

### 新增模型适配器
1. 实现 Google ADK 的 `model.LLM` 接口
2. 放在 `internal/models/`
3. 参考既有模式（`openai.go`/`xai.go`）

### 数据库迁移
1. 在 `migrations/` 添加顺序编号 SQL 文件
2. 手动执行：`psql -d project_her -f migrations/00X_name.sql`

---

**最后更新**：2026-01-25  
**Go 版本**：1.25+  
**项目**：[github.com/easeaico/project-her](https://github.com/easeaico/project-her)
