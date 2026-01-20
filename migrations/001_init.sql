-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE characters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    personality TEXT,
    scenario TEXT,
    first_message TEXT,
    example_dialogue TEXT, -- 对应 mes_example
    avatar_path VARCHAR(255), -- 存储上传后的图片路径
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);