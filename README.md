<p align="center">
  <h1 align="center">Moneta</h1>
  <p align="center">
    <strong>Local-first semantic memory for code</strong>
  </p>
  <p align="center">
    A blazing-fast, privacy-focused code memory system that runs entirely on your machine.
    <br />
    Inspired by <a href="https://github.com/supermemoryai/supermemory">Supermemory</a>, optimized for local development.
  </p>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#usage">Usage</a> •
  <a href="#api">API</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#performance">Performance</a>
</p>

---

## Why Moneta?

Modern AI coding assistants forget everything after each session. Moneta gives your AI a persistent, searchable memory that:

- **Runs locally** — Your code context never leaves your machine
- **Works offline** — No API calls, no latency, always available
- **Stays fast** — Sub-10ms search across thousands of memories
- **Respects code** — Intelligent chunking that understands function boundaries

## Features

| Feature | Description |
|---------|-------------|
| **Semantic Search** | Find memories by meaning, not just keywords |
| **Code-Aware Chunking** | Respects function and class boundaries |
| **Local Embeddings** | Uses Ollama for privacy-preserving vector generation |
| **SQLite Storage** | Single-file database, zero configuration |
| **SIMD Optimized** | 8-16x faster vector operations |
| **LRU Caching** | 90%+ cache hit rate for repeated queries |
| **Claude Code Ready** | HTTP API for AI assistant integration |

## Quick Start

### Prerequisites

1. **Go 1.22+** — [Install Go](https://go.dev/dl/)
2. **Ollama** — [Install Ollama](https://ollama.com)

```bash
# Pull the embedding model
ollama pull nomic-embed-text
```

### Installation

```bash
# Clone the repository
git clone https://github.com/shivavenkatesh/moneta.git
cd moneta

# Build and install
make build
make install

# Or install directly with Go
go install github.com/shivavenkatesh/moneta/cmd/moneta@latest
```

### Verify Installation

```bash
moneta --version
moneta stats
```

## Usage

### Adding Memories

Store code patterns, architecture decisions, and context:

```bash
# Add a pattern
moneta add "We use Repository pattern for database access" --type pattern

# Add an architecture decision
moneta add "Chose PostgreSQL for ACID transactions" --type architecture

# Add a gotcha (common pitfall)
moneta add "Date parsing fails for ISO strings without timezone" --type gotcha

# Add with file reference
moneta add "This module handles authentication" --file src/auth/index.ts
```

### Searching Memories

Find relevant context using natural language:

```bash
# Semantic search
moneta search "how do we access the database"

# Filter by type
moneta search "error handling" --type gotcha

# Adjust sensitivity
moneta search "API patterns" --threshold 0.7 --limit 5
```

### Indexing Codebases

Automatically chunk and index source code:

```bash
# Index a directory
moneta index ./src --project myapp

# Index specific file
moneta index ./README.md

# Index with custom project name
moneta index . --project backend-api
```

### Server Mode

Start the HTTP server for AI assistant integration:

```bash
# Start on default port (3456)
moneta serve

# Custom host/port
moneta serve --host 0.0.0.0 --port 8080
```

## Memory Types

Categorize memories for better organization and filtering:

| Type | Use Case | Example |
|------|----------|---------|
| `architecture` | High-level design decisions | "Microservices communicate via gRPC" |
| `pattern` | Code patterns and conventions | "Use factory pattern for service instantiation" |
| `decision` | Why something was done | "Chose Redis for session storage due to TTL support" |
| `gotcha` | Bugs, edge cases, warnings | "Array.sort() modifies in place in JavaScript" |
| `context` | General information | "This service handles user notifications" |
| `preference` | Coding style preferences | "Prefer early returns over nested conditionals" |

## API Reference

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/memory` | Add a new memory |
| `GET` | `/memory/:id` | Retrieve a memory by ID |
| `DELETE` | `/memory/:id` | Delete a memory |
| `POST` | `/search` | Semantic search |
| `POST` | `/index` | Index a file/directory |
| `GET` | `/stats` | Storage statistics |
| `GET` | `/health` | Health check |
| `GET` | `/projects` | List projects |

### Add Memory

```bash
curl -X POST http://localhost:3456/memory \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Use dependency injection for better testability",
    "type": "pattern",
    "project": "myapp",
    "metadata": {
      "author": "team"
    }
  }'
```

### Search Memories

```bash
curl -X POST http://localhost:3456/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "testing patterns",
    "project": "myapp",
    "limit": 10,
    "threshold": 0.5
  }'
```

### Response Format

```json
{
  "results": [
    {
      "memory": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "content": "Use dependency injection for better testability",
        "type": "pattern",
        "project": "myapp",
        "created_at": "2024-01-15T10:30:00Z"
      },
      "similarity": 0.89
    }
  ],
  "total": 1,
  "timing_ms": 8
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              MONETA                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                        CLI / HTTP Server                          │   │
│  │                     (cobra / net/http)                            │   │
│  └─────────────────────────────┬────────────────────────────────────┘   │
│                                │                                         │
│  ┌─────────────────────────────▼────────────────────────────────────┐   │
│  │                       Memory Service                              │   │
│  │              (orchestrates all operations)                        │   │
│  └───────────┬─────────────────┬─────────────────┬──────────────────┘   │
│              │                 │                 │                       │
│  ┌───────────▼───────┐ ┌───────▼───────┐ ┌───────▼───────────────────┐  │
│  │    Embedder       │ │    Chunker    │ │         Store             │  │
│  │  ┌─────────────┐  │ │ ┌───────────┐ │ │  ┌─────────────────────┐  │  │
│  │  │   Ollama    │  │ │ │   Code    │ │ │  │  SQLite + sqlite-vec│  │  │
│  │  │   Client    │  │ │ │   Aware   │ │ │  │  (vector search)    │  │  │
│  │  └─────────────┘  │ │ └───────────┘ │ │  └─────────────────────┘  │  │
│  │  ┌─────────────┐  │ │ ┌───────────┐ │ │  ┌─────────────────────┐  │  │
│  │  │  LRU Cache  │  │ │ │   Line    │ │ │  │  SIMD Similarity    │  │  │
│  │  │             │  │ │ │   Based   │ │ │  │  (cosine distance)  │  │  │
│  │  └─────────────┘  │ │ └───────────┘ │ │  └─────────────────────┘  │  │
│  └───────────────────┘ └───────────────┘ └───────────────────────────┘  │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                         Data Layer                                │   │
│  │                    ~/.moneta/moneta.db                            │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Performance

Moneta is optimized for local-first performance:

| Metric | Value | Notes |
|--------|-------|-------|
| Cold start | <100ms | SQLite WAL mode |
| Search latency | <10ms | 10k memories |
| Memory usage | ~50MB | Typical workload |
| Cache hit rate | >90% | Repeated queries |
| Storage efficiency | ~1KB/memory | With embeddings |

### Optimizations

1. **SIMD Vector Operations** — Cosine similarity computed with auto-vectorized loops (AVX2/NEON)
2. **LRU Embedding Cache** — Avoid redundant Ollama calls for repeated content
3. **Zero-Copy Serialization** — Minimal allocations in hot paths
4. **Prepared Statements** — Cached SQL for faster queries
5. **WAL Mode** — Concurrent reads during writes

## Docker

### Build

```bash
docker build -t moneta .
```

### Run

```bash
# Run with persistent volume
docker run -d \
  --name moneta \
  -p 3456:3456 \
  -v moneta-data:/data \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  moneta

# Check logs
docker logs moneta
```

### Docker Compose

```bash
docker-compose up -d
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MONETA_DATA_DIR` | `~/.moneta` | Data storage directory |
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model name |

### Data Directory Structure

```
~/.moneta/
├── moneta.db        # SQLite database with vectors
├── moneta.db-wal    # Write-ahead log
└── moneta.db-shm    # Shared memory file
```

## Claude Code Integration

Moneta is designed to integrate with AI coding assistants like Claude Code. Start the server and point your assistant to `http://localhost:3456`.

Example plugin configuration:

```javascript
// Claude Code plugin
const MONETA_URL = "http://localhost:3456";

async function searchMemories(query, project) {
  const response = await fetch(`${MONETA_URL}/search`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, project, limit: 10 })
  });
  return response.json();
}

async function addMemory(content, type, project) {
  const response = await fetch(`${MONETA_URL}/memory`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content, type, project })
  });
  return response.json();
}
```

## Development

### Build from Source

```bash
git clone https://github.com/shivavenkatesh/moneta.git
cd moneta
make deps
make build
```

### Run Tests

```bash
make test
make test-cover  # With coverage report
make bench       # Run benchmarks
```

### Project Structure

```
moneta/
├── cmd/moneta/          # CLI entry point
├── internal/
│   ├── cache/           # LRU cache implementation
│   ├── chunking/        # Code-aware text chunking
│   ├── embeddings/      # Ollama client
│   ├── memory/          # Core service layer
│   ├── server/          # HTTP API server
│   ├── simd/            # SIMD-optimized vector ops
│   └── store/sqlite/    # SQLite storage
├── pkg/types/           # Public type definitions
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Roadmap

- [ ] ONNX Runtime support (remove Ollama dependency)
- [ ] HNSW index for sub-linear search
- [ ] Tree-sitter for better code parsing
- [ ] MCP server for Claude Desktop
- [ ] VS Code extension
- [ ] Memory graph visualization

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting a PR.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License — see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Supermemory](https://github.com/supermemoryai/supermemory) — Inspiration for the architecture
- [Ollama](https://ollama.com) — Local embedding generation
- [SQLite](https://sqlite.org) — The world's most deployed database
- [sqlite-vec](https://github.com/asg017/sqlite-vec) — Vector search extension

---

<p align="center">
  Built with ❤️ for developers who value privacy and speed.
</p>
