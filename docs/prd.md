# 产品需求文档 (PRD) - Project Her: AI 伴侣后端系统

| 文档版本 | 1.0 |
| --- | --- |
| **项目名称** | Project Her (AI 角色扮演智能体演示) |
| **状态** | 课程最终演示版 (Demo) |
| **核心技术** | Golang, Google ADK, PostgreSQL (pgvector) |
| **目标用户** | 希望构建高情商、有记忆的 AI 伴侣的开发者学员 |

---

## 1. 项目背景与目标 (Background & Goals)

本产品旨在展示如何使用 Golang 和 Google ADK 框架构建一个生产级、低延迟的 AI 角色扮演（Role-Play）后端服务。区别于普通的聊天机器人，本项目强调**深度拟人化**、**长期记忆**。

### 核心价值主张

1. **高沉浸感：** 通过分层 Prompt 设计，打破 AI 机械感。
2. **无限记忆：** 基于 RAG 技术，让 AI 记住用户的一言一行。
3. **动态情感：** 内置情感状态机，AI 的态度随好感度变化。

---

## 2. 功能范围 (Scope)

### ✅ 在范围内 (In Scope)

* **核心对话：** 支持流式（Streaming）文本回复，支持 Markdown 格式。
* **记忆系统：** 短期记忆（上下文窗口）+ 长期记忆（向量检索）。
* **情感系统：** 基于对话内容自动更新好感度与情绪状态。
* **演示界面：** 使用ADK-go的调试界面作为演示界面即可，不需要开发接口和Web界面。 

---

## 3. 详细功能需求 (Functional Requirements)

### 3.1 模块一：核心对话引擎 (Chat Engine)

**用户故事：** 用户发送消息，AI 根据当前人设、心情和记忆进行回复。

* **FR-2.1: 分层 Prompt 组装**
* 系统必须按以下顺序动态构建 Prompt：
1. **System:** 强制 RP 规则 (ADK SystemInstruction)。
2. **Persona:** 渲染角色姓名、性格、外貌。
3. **World/State:** 当前时间、地点、用户资料。
4. **Memory (RAG):** 检索到的 top-k 相关历史片段。
5. **Few-Shot:** 经过变量替换（{{char}}/{{user}}）的 `mes_example`。
6. **History:** 最近 N 轮对话（滑动窗口）。
7. **Anchor:** 尾部指令（如“保持简短”）。

### 3.2 模块二：记忆与检索 (Memory & RAG)

**用户故事：** 用户提到“我上次说的那个电影”，AI 能反应过来是《疯狂动物城》。

* **FR-3.1: 向量化 (Embedding)**
* 用户发送消息后，异步调用 Embedding 模型将文本转化为向量。
* 存储向量至 PostgreSQL `memories` 表（使用 `vector` 类型）。

* **FR-3.2: 语义检索**
* 在每一轮对话前，对用户输入进行 Embedding，在数据库中检索余弦相似度最高的 3-5 条记录。
* 设置相似度阈值（如 0.7），低于阈值不注入 Prompt。

* **FR-3.3: 摘要机制 (可选高级功能)**
* 每累计 20 轮对话，后台触发一次 LLM 总结任务，将对话压缩为一条“情节记忆”存入库。

### 3.3 模块四：情感状态机 (Emotion Engine)

**用户故事：** 用户辱骂 AI，AI 进入“生气”状态，回复变冷淡；用户道歉后，AI 转为“委屈”。

* **FR-4.1: 情感分析**
* 每轮对话后，调用轻量级 LLM (或 Prompt 链) 分析当前对话的情感倾向 (Positive/Negative/Neutral)。

* **FR-4.2: 数值系统**
* 维护 `affection_score` (0-100)。
* 维护 `current_mood` (Happy, Angry, Sad, Neutral)。

* **FR-4.3: 状态反馈**
* 不同的 `current_mood` 对应不同的 System Prompt 微调指令（例如：Angry -> "Reply shortly and coldly"）。

---

## 4. 数据结构设计 (Data Schema)

基于 PostgreSQL。

### 4.1 Characters 表

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| `id` | SERIAL | 主键 |
| `name` | VARCHAR | 角色名 |
| `system_prompt_raw` | TEXT | 原始设定文本 |
| `example_dialogue` | TEXT | 对话范例 |
| `avatar_url` | VARCHAR | 图片路径 |
| `affection` | INT | 当前好感度 (0-100) |

### 4.2 Chat_History 表

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| `id` | SERIAL | 主键 |
| `session_id` | UUID | 会话 ID |
| `role` | VARCHAR | user / model |
| `content` | TEXT | 内容 |
| `embedding` | VECTOR(768) | **核心：pgvector 向量数据** |
| `created_at` | TIMESTAMP | 时间 |

---

## 6. 技术栈与环境 (Tech Stack)

* **编程语言:** Go 1.25+
* **AI SDK:** `adk-go` 文档地址: https://github.com/google/adk-go
* **LLM:** grok-4-fast / grok-4
* **Database:** PostgreSQL 17
* Extension: `pgvector` (用于向量搜索)

---

## 7. 验收标准 (Acceptance Criteria)

2. **记忆测试：** 告诉 AI 用户的名字，重启程序或清空上下文后，询问“我是谁”，AI 仍能回答正确（触发 RAG）。
3. **情感测试：** 连续发送负面内容，AI 表现出抗拒或愤怒；查看数据库，`affection` 值下降。
4. **性能测试：** 首字响应时间 (TTFT) 小于 2 秒。