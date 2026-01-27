# Project Her - AI 伴侣后端系统

基于 Golang 和 Google ADK 构建的高情商、有记忆的 AI 角色扮演后端服务。

## 特性

- 🧠 **长期记忆**：基于 RAG（检索增强生成）技术，使用 PostgreSQL + pgvector 实现向量检索
- 💝 **情感系统**：动态好感度与情绪状态机，AI 的态度随互动变化
- 🎭 **深度拟人化**：分层 Prompt（提示词）设计，支持自定义角色人设
- 🖼️ **图片生成**：集成 Gemini 图片生成能力
- ⚡ **流式响应**：支持模型流式输出，低延迟体验

## 技术栈

- **语言**：Go 1.25+
- **AI SDK（开发工具包）**：[google/adk-go](https://github.com/google/adk-go)
- **LLM（大语言模型）**：Grok-4（可配置其他模型）
- **数据库**：PostgreSQL 17 + pgvector 扩展
- **嵌入模型**：Google text-embedding-004

## 快速开始

### 前置要求

```bash
# 安装 Go 1.25+
go version

# 安装 PostgreSQL 17
brew install postgresql@17

# 启动 PostgreSQL
brew services start postgresql@17

# 创建数据库
createdb project_her
```

## 术语表

| 术语 | 说明 |
| --- | --- |
| LLM | Large Language Model，大语言模型 |
| RAG | Retrieval-Augmented Generation，检索增强生成 |
| ADK | Agent Development Kit，代理开发工具包 |
| Prompt | 提示词 |
| Embedding | 向量化/嵌入 |
| Token | 词元（模型计费与上下文长度单位） |
| URL | 统一资源定位符 |
| URI | 统一资源标识符 |
| Base64 | 二进制到文本的编码方式 |
| pgvector | PostgreSQL 向量扩展 |

### 安装依赖

```bash
git clone https://github.com/easeaico/project-her.git
cd project-her
go mod download
```

### 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填入你的 API Key
```

必需配置项：
- `GOOGLE_API_KEY`：Google AI API 密钥
- `XAI_API_KEY`：xAI (Grok) API 密钥
- `DATABASE_URL`：PostgreSQL 连接字符串

可选配置项：
- `CHAT_MODEL`：聊天模型名称（默认：grok-4-fast）
- `MEMORY_MODEL`：记忆处理模型名称（默认：gemini-2.0-flash）
- `EMBEDDING_MODEL`：嵌入模型名称（默认：text-embedding-004）
- `TOP_K`：RAG 检索数量（默认：5）
- `SIMILARITY_THRESHOLD`：相似度阈值（默认：0.7）
- `MEMORY_TRUNK_SIZE`：记忆窗口轮次阈值（默认：100）

### 初始化数据库

```bash
psql -d project_her -f migrations/001_init.sql
```

### 运行应用

```bash
# 开发模式
go run cmd/platform/main.go

# 编译运行
go build -o bin/project-her cmd/platform/main.go
./bin/project-her
```

应用启动后，使用 ADK 自带的调试界面进行对话测试。

## 项目结构

```
project-her/
├── cmd/platform/         # 应用入口
├── internal/
│   ├── agent/           # Agent 实现（角色扮演、记忆摘要）
│   ├── config/          # 配置加载
│   ├── emotion/         # 情感分析与状态机
│   ├── memory/          # 记忆服务（嵌入、检索）
│   ├── models/          # LLM 模型适配器
│   ├── prompt/          # Prompt（提示词）构建器
│   ├── repository/      # 数据访问层
│   ├── types/           # 类型定义
│   └── utils/           # 工具函数
├── migrations/          # 数据库迁移脚本
└── docs/                # 文档
```

## 使用指南

### 对话命令

- `/image [描述]`：生成图片，例如 `/image 一个在雨中撑伞的女孩`

### 自定义角色

编辑 `migrations/001_init.sql` 中的 `INSERT INTO characters` 语句，或直接在数据库中修改：

```sql
UPDATE characters SET 
  name = '你的角色名',
  personality = '性格描述',
  scenario = '场景设定'
WHERE id = 1;
```

## 开发

### 运行测试

```bash
go test ./...
```

### 代码检查

```bash
go vet ./...
golangci-lint run  # 需先安装 golangci-lint
```

### 格式化代码

```bash
gofmt -w .
```

## 安全提示

⚠️ **重要**：`.env` 文件包含敏感信息，已添加到 `.gitignore`。请勿将实际的 API 密钥提交到版本控制系统。

## 许可证

本项目采用 Apache License 2.0 许可证。详见 [LICENSE](LICENSE) 文件。

## 致谢

- [Google ADK](https://github.com/google/adk-go) - Agent Development Kit
- [pgvector](https://github.com/pgvector/pgvector) - PostgreSQL 向量扩展
