# AGENTS.md - Development Guidelines for AI Coding Agents

This document provides coding standards and commands for AI agents working in the **Project Her** codebase - a Go-based AI companion backend using Google ADK and PostgreSQL with pgvector.

---

## Build, Lint, and Test Commands

### Build
```bash
# Build all packages
go build ./...

# Build main binary
go build -o bin/project-her cmd/platform/main.go

# Clean build
rm -rf bin/ && go build -o bin/project-her cmd/platform/main.go
```

### Dependency Management
```bash
# Download dependencies
go mod download

# Sync dependencies (after adding/removing imports)
go mod tidy

# Verify dependencies
go mod verify
```

### Linting and Formatting
```bash
# Format all Go files
gofmt -w .

# Check formatting (without modifying files)
gofmt -l .

# Static analysis
go vet ./...

# Advanced linting (if golangci-lint is installed)
golangci-lint run
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test
go test -v -run TestFunctionName ./path/to/package

# Run tests for a specific package
go test -v ./internal/agent

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests with race detection
go test -race ./...
```

**Note**: Currently, there are **no test files** in this project. When creating tests:
- Name test files with `_test.go` suffix (e.g., `roleplay_test.go`)
- Place test files alongside the code they test
- Use table-driven tests for comprehensive coverage

---

## Code Style Guidelines

### 1. Import Organization

Imports MUST be organized in **three distinct blocks** separated by blank lines:

```go
import (
    // Block 1: Standard library
    "context"
    "fmt"
    "log/slog"

    // Block 2: External dependencies
    "github.com/pgvector/pgvector-go"
    "google.golang.org/adk/agent"
    "gorm.io/gorm"

    // Block 3: Internal packages
    "github.com/easeaico/project-her/internal/config"
    "github.com/easeaico/project-her/internal/types"
)
```

**Rules**:
- Use explicit aliases for naming conflicts (e.g., `internalagent "github.com/.../internal/agent"`)
- Alphabetize within each block
- Run `gofmt` to auto-organize

### 2. Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| **Packages** | Short, lowercase, singular | `agent`, `config`, `memory` |
| **Exported Structs** | PascalCase | `MemoryRepo`, `Config`, `Character` |
| **Unexported Structs** | camelCase | `memoryModel`, `openaiModel` |
| **Interfaces** | PascalCase, capability-named | `CharacterRepo`, `Embedder` |
| **Constructors** | `New<Type>` pattern | `NewMemoryRepo(db)`, `NewEmbedder(...)` |
| **Variables** | Short, context-aware | `ctx`, `cfg`, `err`, `db` |
| **Constants** | PascalCase or UPPER_SNAKE | `MemoryTypeChat`, `TOP_K` |

### 3. Error Handling

**ALWAYS**:
- Check errors immediately: `if err != nil`
- Wrap errors with context: `fmt.Errorf("description: %w", err)`
- Use `%w` verb for error wrapping (enables `errors.Is` and `errors.As`)
- Return errors to callers rather than logging (except in callbacks/goroutines)

**In `main.go` startup**:
```go
if err != nil {
    log.Fatalf("failed to connect to database: %v", err)
}
```

**In internal packages**:
```go
if err != nil {
    return nil, fmt.Errorf("failed to create grok model: %w", err)
}
```

**In callbacks/background goroutines**:
```go
if err != nil {
    slog.Error("failed to generate image", "error", err.Error())
    return fallbackValue, nil
}
```

**NEVER**:
- Ignore errors with `_` unless explicitly justified in comments
- Use `panic()` except in truly unrecoverable situations
- Suppress errors with type assertions (`as any`) or ignore directives

### 4. Logging

**Use `log/slog` for structured logging**:

```go
import "log/slog"

// Error logging with context
slog.Error("failed to close stream", "error", err.Error())

// Warning with key-value pairs
slog.Warn("high memory usage", "allocated_mb", allocatedMB, "threshold_mb", thresholdMB)

// Info logging
slog.Info("agent initialized", "model", modelName, "character_id", charID)
```

**Rules**:
- Use key-value pairs for structured data
- Avoid `fmt.Printf` for logging (use it only for user-facing output)
- In `cmd/platform/main.go`, use `log.Fatalf` for fatal startup errors

### 5. Type Usage

**Context**:
```go
// ALWAYS first parameter for I/O or long-running functions
func (r *MemoryRepo) AddMemory(ctx context.Context, mem types.Memory) error
```

**Pointers vs Values**:
- **Use pointers** for:
  - Service structs (`*MemoryRepo`, `*Config`, `*gorm.DB`)
  - Large config objects
  - Types that need to be mutable or nullable
- **Use values** for:
  - Small DTOs (`types.Memory`, `types.Character`)
  - Immutable data structures

**Interfaces**:
- Define interfaces in the **consumer package**, not the implementation package
- Example: `CharacterRepo` interface is defined in `internal/agent/roleplay.go`, not in `internal/repository/`

**Database Models**:
- Separate internal DB models (with GORM tags) from domain types
- Provide converter functions: `func memoryFromModel(model memoryModel) types.Memory`

Types:
- Use any replace interface{}

### 6. Function Signatures

**Constructor Pattern**:
```go
// Returns pointer, error for structs that wrap dependencies
func NewMemoryRepo(db *gorm.DB) *MemoryRepo

// Returns struct, error for value types
func NewEmbedder(ctx context.Context, apiKey, model string) (*Embedder, error)
```

**Service Methods**:
```go
// Context first, then parameters, return (result, error)
func (r *MemoryRepo) SearchSimilar(
    ctx context.Context,
    userID, appName, memoryType string,
    embedding []float32,
    topK int,
    threshold float64,
) ([]types.RetrievedMemory, error)
```

### 7. Code Organization

**Project Structure**:
```
project-her/
├── cmd/platform/          # Application entry point (main.go)
├── internal/
│   ├── agent/            # Business logic (roleplay, memory agents)
│   ├── config/           # Configuration loading
│   ├── emotion/          # Emotion analysis & state machine
│   ├── memory/           # Memory service (embeddings, RAG)
│   ├── models/           # LLM adapters (OpenAI, xAI)
│   ├── prompt/           # Prompt building logic
│   ├── repository/       # Data access layer (GORM)
│   ├── types/            # Shared domain types
│   └── utils/            # Helper functions
├── migrations/           # SQL schema migrations
└── docs/                 # Documentation
```

**Dependency Injection**:
- Pass dependencies explicitly via constructors
- Wire dependencies in `main.go`
- Avoid global variables

### 8. Database Patterns

**GORM Usage**:
```go
// Use WithContext for all queries
r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&records)

// Use parameterized queries (NEVER string interpolation)
query := r.db.WithContext(ctx).Where("id = ?", id)  // ✓ Safe
// query := r.db.Where("id = " + id)  // ✗ SQL injection risk
```

**Transactions**:
```go
err := r.db.Transaction(func(tx *gorm.DB) error {
    // Perform operations with tx
    return nil
})
```

### 9. Comments and Documentation

**ONLY add comments when**:
- Explaining complex algorithms or business logic
- Documenting public APIs (exported functions/types)
- Clarifying non-obvious behavior or edge cases
- Warning about security or performance implications

**Avoid unnecessary comments**:
```go
// Bad: Obvious comment
// Get user by ID
func GetUserByID(id int) (*User, error)

// Good: Self-documenting code
func GetUserByID(id int) (*User, error)
```

**Package documentation**:
```go
// Package agent provides agent initialization and business logic
// for the AI companion roleplay system.
package agent
```

### 10. Testing Guidelines (for when tests are added)

**File Structure**:
```go
// roleplay_test.go
package agent

import (
    "testing"
)

func TestNewRolePlayAgent(t *testing.T) {
    // Table-driven tests
    tests := []struct {
        name    string
        config  *config.Config
        wantErr bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

**Mocking**:
- Use interfaces for dependencies (already in place)
- Create test implementations in `*_test.go` files

---

## Environment Configuration

**Required Environment Variables** (see `.env.example`):
- `GOOGLE_API_KEY`: Google AI API key
- `XAI_API_KEY`: xAI (Grok) API key  
- `DATABASE_URL`: PostgreSQL connection string

**Optional** (with defaults in `internal/config/config.go`):
- `CHAT_MODEL` (default: `grok-4-fast`)
- `MEMORY_MODEL` (default: `gemini-2.0-flash`)
- `EMBEDDING_MODEL` (default: `text-embedding-004`)
- `IMAGE_MODEL` (default: `gemini-2.0-flash-exp`)
- `TOP_K` (default: `5`)
- `SIMILARITY_THRESHOLD` (default: `0.7`)
- `HISTORY_LIMIT` (default: `10`)
- `MEMORY_TRUNK_SIZE` (default: `100`)
- `ASPECT_RATIO` (default: `9:16`)

---

## Common Tasks

### Adding a New Repository
1. Define interface in consumer package
2. Create implementation in `internal/repository/`
3. Add constructor following `NewXxxRepo(db *gorm.DB)` pattern
4. Wire in `cmd/platform/main.go`

### Adding a New Model Adapter
1. Implement `model.LLM` interface from Google ADK
2. Place in `internal/models/`
3. Follow existing patterns (see `openai.go` or `xai.go`)

### Database Migration
1. Add SQL file to `migrations/` with sequential numbering
2. Run manually: `psql -d project_her -f migrations/00X_name.sql`

---

**Last Updated**: 2026-01-25  
**Go Version**: 1.23+  
**Project**: [github.com/easeaico/project-her](https://github.com/easeaico/project-her)
