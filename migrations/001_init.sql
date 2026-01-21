-- pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- characters
CREATE TABLE characters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    appearance TEXT,
    personality TEXT,
    scenario TEXT,
    first_message TEXT,
    example_dialogue TEXT,
    system_prompt TEXT,
    avatar_path VARCHAR(255),
    affection INT DEFAULT 50,
    current_mood VARCHAR(50) DEFAULT 'Neutral',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- chat_history
CREATE TABLE chat_history (
    id SERIAL PRIMARY KEY,
    session_id UUID NOT NULL,
    character_id INT REFERENCES characters(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    embedding VECTOR(768),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- indexes
CREATE INDEX idx_chat_history_embedding ON chat_history 
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE INDEX idx_chat_history_session ON chat_history (session_id, created_at);

CREATE INDEX idx_chat_history_character ON chat_history (character_id);