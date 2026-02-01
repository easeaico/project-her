-- pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- characters: companion profiles
CREATE TABLE characters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    personality TEXT,
    scenario TEXT,
    first_mes TEXT,
    mes_example TEXT,
    system_prompt TEXT,
    avatar VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- memories: retrieval-oriented summaries + embeddings
CREATE TABLE memories (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(64),
    app_name VARCHAR(255),
    -- type: chat/persona/facts/events
    type VARCHAR(20) NOT NULL,
    -- summary: summarized memory body
    summary TEXT,
    -- facts: durable facts or user preferences
    facts JSONB,
    -- commitments: promises or plans
    commitments JSONB,
    -- emotions: relationship or emotional shifts
    emotions JSONB,
    -- time_range: covered period of the summarized window
    time_range JSONB,
    -- salience_score: importance score in [0,1]
    salience_score FLOAT DEFAULT 0,
    -- embedding: vector for similarity search
    embedding VECTOR(768),
    -- created_at: memory creation time
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- chat histories: raw message storage
CREATE TABLE chat_histories (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(64),
    app_name VARCHAR(255),
    content TEXT NOT NULL,
    turn_count INT DEFAULT 0,
    summarized BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- indexes
CREATE INDEX idx_memories_embedding ON memories
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- recent lookups per user
CREATE INDEX idx_memories_user ON memories (user_id, created_at);

-- filter by memory type
CREATE INDEX idx_memories_type ON memories (type);

-- salience helps ranking
CREATE INDEX idx_memories_salience ON memories (salience_score);
-- GIN indexes enable containment filtering
CREATE INDEX idx_memories_facts ON memories USING gin (facts);
CREATE INDEX idx_memories_commitments ON memories USING gin (commitments);

-- chat histories lookups
CREATE INDEX idx_chat_histories_user ON chat_histories (user_id, app_name, created_at);
CREATE INDEX idx_chat_histories_summarized ON chat_histories (summarized, created_at);
