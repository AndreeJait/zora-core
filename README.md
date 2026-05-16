# Zora Core

Agent orchestration service for the [Zora](https://github.com/AndreeJait) platform. Runs a think-act-observe loop powered by an LLM, discovers and calls tools via the MCP server, retrieves knowledge context, and processes incoming WhatsApp messages through a WAHA webhook.

Part of the Zora ecosystem:

- **[go-utility](https://github.com/AndreeJait/go-utility)** — shared infrastructure wrappers (logging, HTTP, DB, Redis, auth, storage, etc.)
- **zora-core** (this repo) — agent orchestration service
- **[zora-mcp-server](https://github.com/AndreeJait/zora-mcp-server)** — tool registry and execution
- **[zora-knowledge](https://github.com/AndreeJait/zora-knowledge)** — knowledge ingestion and semantic search

## Architecture

Hexagonal (Ports & Adapters) with strict inward dependency: **adapters → ports → domain**.

```
cmd/
  http/                HTTP server entry point (wiring + DI)
  worker/              Background worker (NSQ consumer)
  migrate/             Migration runner (up, down, fresh)
domain/
  entity/              ZoraState, Conversation, Task, WAHA models
  error/               Domain errors
port/
  inbound/
    agent/             Agent use case interface + input/output DTOs
    webhook/           Webhook use case interface
    task/              Task management interface
    setting/           Runtime settings interface
    health/            Health check interface
  outbound/            ToolRegistry, KnowledgeClient, ConversationRepo, etc.
usecase/
  agent.go             Agent orchestration (think-act-observe graph)
  webhook.go           Incoming WAHA message handler
  worker.go            NSQ message handlers + task dispatcher
  health.go            Health check
  graph/               LangGraph-style stateful graph (nodes, reducer, builder)
adapter/
  inbound/echo/        HTTP handlers (Echo v5)
  outbound/
    tool_registry.go   HTTP client to zora-mcp-server
    knowledge.go       HTTP client to zora-knowledge
    conversation/      GORM conversation repository
    checkpointer.go    Redis-based graph checkpoint
config/                Configuration loading
files/
  config/              app.yaml + app.local.yaml
  migrations/          PostgreSQL migrations
```

## API

### Agent

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/agent/execute` | Execute agent synchronously |
| GET | `/api/v1/agent/stream` | Stream agent steps via SSE |
| POST | `/api/v1/agent/test` | Test agent with `!zora` message |

### Graph State

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/agent/state/:sessionId` | Get current agent state |
| GET | `/api/v1/agent/history/:sessionId` | Get checkpoint history |
| POST | `/api/v1/agent/redirect` | Redirect agent to a different step |
| POST | `/api/v1/agent/revert` | Revert agent to a previous checkpoint |

### Task Management

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tasks` | List tasks |
| GET | `/api/v1/tasks/:id` | Get task by ID |
| GET | `/api/v1/tasks/:id/graph` | Render Mermaid diagram |

### Settings

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/settings` | List all settings |
| GET | `/api/v1/settings/:key` | Get setting by key |
| PUT | `/api/v1/settings/:key` | Set a setting value |

### Webhook

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhook` | Handle incoming WAHA WhatsApp messages |

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Service health (DB + Redis connectivity) |

## Agent Think-Act-Observe Loop

```
think ──► act ──► observe ──► think ──► ... ──► END
  │         │         │
  │         │         └─ Convert tool results → LLM messages
  │         └─ Call MCP tools via zora-mcp-server
  └─ Extract tags → Embed task → Search tools + knowledge → Call LLM
```

- **think**: Extracts tags from the task, embeds the task text, searches for relevant tools (via MCP server) and knowledge (via zora-knowledge), builds a system prompt with context, and calls the LLM. If the LLM returns tool calls, routes to `act`. Includes a plan mode where tool calls are formatted as an execution plan for user approval.
- **act**: Executes each pending tool call against the MCP server or locally for built-in WAHA tools (reactions, typing indicators, seen).
- **observe**: Converts tool results into LLM messages, increments iteration count. Routes back to `think` or `END` if resolved or max iterations reached.

### Feature Toggles

Tools and knowledge retrieval can be toggled independently in the agent config:

| Config Key | Default | Description |
|------------|---------|-------------|
| `agent.tools_enabled` | `true` | Whether to search and call tools from MCP server |
| `agent.knowledge_enabled` | `false` | Whether to search knowledge from zora-knowledge (release still in progress) |

When disabled, the agent skips the corresponding search and proceeds without that context.

## Getting Started

### Prerequisites

- Go 1.26+
- PostgreSQL
- Redis
- Ollama (for LLM + embeddings)
- NSQ (for background task processing)
- [zora-mcp-server](https://github.com/AndreeJait/zora-mcp-server) running on port 8081
- [zora-knowledge](https://github.com/AndreeJait/zora-knowledge) running on port 8001 (optional, see feature toggles)
- WAHA (for WhatsApp integration, optional)

### Setup

```bash
git clone https://github.com/AndreeJait/zora-core.git
cd zora-core

# Copy local config and customize
cp files/config/app.local.yaml.example files/config/app.local.yaml

# Run migrations
make migrate-up

# Run
make run
```

### Run

```bash
make run                      # Default engine (echo)
make run-engine E=gin         # Specific engine (gin|mux)
make build                    # Build binary to bin/server
```

### Run with Docker

```bash
docker compose -f deploy/docker-compose.yaml up --build
```

See `deploy/docker-compose.yaml` for environment variables. Services included:
- **zora-core** — port 8080 (HTTP API)
- **zora-worker** — background NSQ consumer
- **PostgreSQL** — port 5434
- **Redis** — port 6382
- **MinIO** — ports 9004 (API), 9005 (console)
- **NSQ** — nsqd (4150-4151), nsqlookupd (4160-4161), nsqadmin (4171)

## CI/CD

Pushes to `master` trigger the GitHub Actions workflow (`.github/workflows/deploy.yml`):

1. Build Docker image
2. Push to `ghcr.io/andreejait/zora-core:latest`
3. SSH into server via cloudflared
4. Run `deploy/deploy.sh` — login, pull, up, migrate, cleanup

## Configuration

Config files in `files/config/`:

| File | Purpose |
|------|---------|
| `app.yaml` | Base config (committed) |
| `app.local.yaml` | Local overrides (gitignored) |

**Override priority** (highest wins): environment variables → `app.local.yaml` → `app.yaml`

```yaml
app:
  name: zora-core
  env: development
  http_port: 8080

http:
  engine: echo
  enable_swagger: true
  debug_mode: true
  api_key: ""          # Set to protect management endpoints

log:
  level: debug
  format: JSON

db:
  driver: gorm
  dialect: postgres
  dsn: "postgres://zora:zora@localhost:5432/zora_core?sslmode=disable"

redis:
  address: "localhost:6379"
  db: 0

agent:
  max_steps: 25
  tool_limit: 15
  knowledge_limit: 5
  tool_min_score: 0.3
  knowledge_min_score: 0.4
  max_tool_context_tokens: 4000
  max_knowledge_context_tokens: 2000
  tools_enabled: true           # search + call tools from MCP server
  knowledge_enabled: false      # search knowledge (release in progress)

llm:
  provider: ollama
  model: "qwen3:14b"
  embed_model: "nomic-embed-text:latest"
  base_url: "http://localhost:11434/v1"
  api_key: "ollama"
  temperature: 0.7
  max_tokens: 4096

minio:
  endpoint: "localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"

knowledge:
  base_url: "http://localhost:8001"

mcp_server:
  base_url: "http://localhost:8081"
  api_key: ""          # API key for zora-mcp-server

waha:
  base_url: "http://localhost:3000"
  api_key: ""
  session: "default"

task:
  worker_count: 5
  channel_size: 1000
  worker_timeout: 5m
  default_max_retry: 3
  default_retry_delay: 30s

nsq:
  nsqd_addr: "127.0.0.1:4150"
  lookupd_addrs:
    - "127.0.0.1:4161"
  channel: "zora-worker"

graceful:
  shutdown_timeout: 10s
```

## Database Migrations

```bash
make migrate-new name=add_column   # Create new migration
make migrate-up                    # Run pending migrations
make migrate-down                 # Roll back last migration
make migrate-fresh                # Drop all + re-run all
```

## Swagger

Swagger UI available at `http://localhost:8080/swagger/` when `http.enable_swagger: true`.

```bash
make swag    # Regenerate docs from annotations
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make run` | Run with default engine (echo) |
| `make run-engine E=gin` | Run with specific engine |
| `make build` | Build binary to `bin/server` |
| `make swag` | Generate Swagger docs |
| `make test` | Run all tests |
| `make vet` | Run static analysis |
| `make tidy` | Clean up dependencies |
| `make migrate-new name=foo` | Create new migration |
| `make migrate-up` | Run pending migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-fresh` | Drop all + re-run all |

## License

MIT
