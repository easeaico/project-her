-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Semantic Memory: Project Rules
-- Stores static, globally-effective knowledge like code style and architecture constraints
CREATE TABLE project_rules (
    id SERIAL PRIMARY KEY,
    category VARCHAR(50) NOT NULL,  -- e.g., "STYLE", "SECURITY", "ARCHITECTURE"
    rule_content TEXT NOT NULL,     -- e.g., "禁止在循环中使用 defer"
    priority INT DEFAULT 1,         -- Rule weight
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for category-based queries
CREATE INDEX idx_rules_category ON project_rules(category);

-- Episodic Memory: Issue History
-- Stores dynamically accumulated experience, supports vector semantic search
CREATE TABLE issue_history (
    id SERIAL PRIMARY KEY,
    task_signature VARCHAR(255),    -- Short fingerprint of task/error
    error_pattern TEXT,             -- Original error message or phenomenon description
    root_cause TEXT,                -- Root cause analysis
    solution_summary TEXT,          -- Solution/code change summary
    embedding vector(768),          -- Core: Embedding vector of problem description
    occurred_at TIMESTAMP DEFAULT NOW()
);

-- Vector index using IVFFlat for similarity search
CREATE INDEX ON issue_history USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Insert some sample rules for testing
INSERT INTO project_rules (category, rule_content, priority) VALUES
    ('STYLE', '禁止在循环中使用 defer', 1),
    ('STYLE', '所有导出的函数必须有文档注释', 1),
    ('SECURITY', '禁止在代码中硬编码密钥或密码', 2),
    ('ARCHITECTURE', '数据库操作必须通过 Repository 层', 1),
    ('ARCHITECTURE', 'HTTP Handler 不得直接调用数据库', 1);
