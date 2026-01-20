-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Characters table: stores AI character profiles
CREATE TABLE characters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    personality TEXT,
    scenario TEXT,
    first_message TEXT,
    example_dialogue TEXT,          -- 对应 mes_example
    system_prompt TEXT,             -- 自定义系统提示词
    avatar_path VARCHAR(255),       -- 存储上传后的图片路径
    affection INT DEFAULT 50,       -- 好感度 (0-100)
    current_mood VARCHAR(50) DEFAULT 'Neutral', -- 当前心情: Happy, Angry, Sad, Neutral
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Chat history table: stores conversation messages with embeddings for RAG
CREATE TABLE chat_history (
    id SERIAL PRIMARY KEY,
    session_id UUID NOT NULL,
    character_id INT REFERENCES characters(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,      -- user / model
    content TEXT NOT NULL,
    embedding VECTOR(768),          -- pgvector embedding for semantic search
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for vector similarity search using cosine distance
CREATE INDEX idx_chat_history_embedding ON chat_history 
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Create index for session queries
CREATE INDEX idx_chat_history_session ON chat_history (session_id, created_at);

-- Create index for character queries
CREATE INDEX idx_chat_history_character ON chat_history (character_id);