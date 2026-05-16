# Zora Core API — Test Requests

Base URL: `http://localhost:8080`

Replace `YOUR_API_KEY` with the value from `http.api_key` in your config.

---

## Health

```bash
curl -s http://localhost:8080/health | jq .
```

---

## Agent

### Execute agent synchronously

```bash
curl -s -X POST http://localhost:8080/api/v1/agent/execute \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "task": "!zora what is the capital of Indonesia?",
    "source": "api"
  }' | jq .
```

### Stream agent (SSE)

```bash
curl -N "http://localhost:8080/api/v1/agent/stream?task=!zora%20hello&source=api" \
  -H "X-API-Key: YOUR_API_KEY"
```

### Test agent (simple message)

```bash
curl -s -X POST http://localhost:8080/api/v1/agent/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{"message": "!zora what is 2+2?"}' | jq .
```

---

## Tasks

### List tasks (with optional filters)

```bash
# All tasks
curl -s "http://localhost:8080/api/v1/tasks?page=1&per_page=10" \
  -H "X-API-Key: YOUR_API_KEY" | jq .

# Filter by status
curl -s "http://localhost:8080/api/v1/tasks?status=completed&page=1&per_page=10" \
  -H "X-API-Key: YOUR_API_KEY" | jq .

# Filter by chat ID
curl -s "http://localhost:8080/api/v1/tasks?chat_id=62812xxx@c.us&page=1&per_page=10" \
  -H "X-API-Key: YOUR_API_KEY" | jq .
```

### Get task by ID

```bash
curl -s "http://localhost:8080/api/v1/tasks/TASK_ID" \
  -H "X-API-Key: YOUR_API_KEY" | jq .
```

### Get task graph (Mermaid text)

```bash
curl -s "http://localhost:8080/api/v1/tasks/TASK_ID/graph?format=mmd" \
  -H "X-API-Key: YOUR_API_KEY" | jq .
```

### Get task graph (MinIO presigned URL)

```bash
curl -s "http://localhost:8080/api/v1/tasks/TASK_ID/graph?format=presigned" \
  -H "X-API-Key: YOUR_API_KEY" | jq .
```

---

## Settings

### List all settings

```bash
curl -s "http://localhost:8080/api/v1/settings" \
  -H "X-API-Key: YOUR_API_KEY" | jq .
```

### Get a setting

```bash
curl -s "http://localhost:8080/api/v1/settings/task.max_retry" \
  -H "X-API-Key: YOUR_API_KEY" | jq .
```

### Update a setting

```bash
curl -s -X PUT "http://localhost:8080/api/v1/settings/task.max_retry" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{"value": "5", "description": "Maximum retry attempts for tasks"}' | jq .
```

---

## Upload

```bash
curl -s -X POST http://localhost:8080/api/v1/upload \
  -H "X-API-Key: YOUR_API_KEY" \
  -F "file=@/path/to/file.pdf" \
  -F "bucket=zora-files" \
  -F "prefix=uploads" | jq .
```

---

## Webhook (WAHA)

```bash
curl -s -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "event": "message",
    "session": "default",
    "payload": {
      "id": "test_msg_001",
      "from": "62812xxx@c.us",
      "to": "62812yyy@c.us",
      "body": "!zora what is the capital of Indonesia?",
      "fromMe": false,
      "hasMedia": false,
      "mentionedIds": []
    }
  }' | jq .
```

---

## Testing chit-chat tool via agent

The chit-chat tool must be registered in zora-mcp-server first. After registering it, you can test it through zora-core's agent:

```bash
curl -s -X POST http://localhost:8080/api/v1/agent/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{"message": "!zora hey how are you?"}' | jq .
```

This will trigger the agent which will:
1. Embed the task
2. Search for relevant tools (should find `waha-chit-chat` if registered)
3. Call `waha-chit-chat` via MCP with `{message: "hey how are you?"}`
4. The chit-chat tool returns the LLM response (no longer sends to WAHA directly)
5. The agent uses `waha-send-text` (built-in tool) to deliver the final answer

### Register chit-chat tool in zora-mcp-server

First, make sure the script is uploaded to MinIO, then register:

```bash
curl -s -X POST http://localhost:8081/api/v1/tools \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_MCP_API_KEY" \
  -d '{
    "name": "waha-chit-chat",
    "description": "Handle casual conversations using an LLM. Processes incoming messages and generates natural responses. Returns the response text; use waha-send-text to deliver it to WhatsApp.",
    "language": "python",
    "object_key": "waha-chit-chat.py",
    "bucket": "zora-scripts",
    "parameters": {
      "type": "object",
      "required": ["message"],
      "properties": {
        "message": {"type": "string", "description": "The incoming message text"},
        "sender": {"type": "string", "description": "Sender name for context"},
        "sessionId": {"type": "string", "description": "Conversation session ID for chat history"}
      }
    },
    "env": {
      "MCP_LLM_BASE_URL": "http://localhost:11434/v1",
      "MCP_LLM_API_KEY": "ollama",
      "MCP_LLM_MODEL": "qwen3:14b"
    }
  }' | jq .
```

### Verify tool is registered

```bash
curl -s "http://localhost:8081/api/v1/tools?per_page=50" \
  -H "X-API-Key: YOUR_MCP_API_KEY" | jq .
```

### Test chit-chat tool directly via MCP

```bash
curl -s -X POST http://localhost:8081/api/v1/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "waha-chit-chat",
    "arguments": {
      "message": "hey how are you?",
      "sender": "Test",
      "sessionId": "test-session"
    }
  }' | jq .
```