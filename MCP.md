# IBM MQ MCP Server
- [IBM MQ MCP Server](#ibm-mq-mcp-server)
  - [How it works](#how-it-works)
    - [Connecting IBM Bob to the MCP server](#connecting-ibm-bob-to-the-mcp-server)
    - [Multi-server support](#multi-server-support)
    - [TLS verification](#tls-verification)
    - [Credentials](#credentials)
  - [Example conversations](#example-conversations)
    - [List queue managers](#list-queue-managers)
    - [Check Native HA status](#check-native-ha-status)
    - [Check uniform cluster balance](#check-uniform-cluster-balance)
    - [Check queue depth](#check-queue-depth)
    - [Failover scenario](#failover-scenario)
  - [Connecting to IBM Bob](#connecting-to-ibm-bob)

This repository includes an [MCP (Model Context Protocol)](https://modelcontextprotocol.io/introduction) server that lets an AI assistant — such as [IBM Bob](https://www.ibm.com/products/bob) — query and administer IBM MQ queue managers using natural language.

The MCP server is located in the [`mq-mcp-server/`](mq-mcp-server/) directory. Full setup instructions are in [`mq-mcp-server/README.md`](mq-mcp-server/README.md).

---

## How it works

The server is written in Python and exposes two MCP tools that wrap the [MQ Administrative REST API](https://www.ibm.com/docs/en/ibm-mq/9.4.x?topic=administering-administration-using-rest-api):

| Tool | Description |
|---|---|
| `dspmq` | Lists all queue managers across every configured mqweb server and their running state |
| `runmqsc` | Runs any MQSC command against a named queue manager |

```
AI Assistant (IBM Bob)
        │
        │  MCP (streamable-http / SSE / stdio)
        ▼
 mqmcpserver.py  ──► MQ REST API (mqweb :9443)  ──► QM1 (Native HA)
                 ──► MQ REST API (mqweb :9444)  ──► QM1 (replica)
                 ──► MQ REST API (mqweb :9445)  ──► QM1 (replica)
                 ──► MQ REST API (mqweb :9446)  ──► QM2 (Native HA)
                 ──► MQ REST API (mqweb :9447)  ──► QM2 (replica)
                 ──► MQ REST API (mqweb :9448)  ──► QM2 (replica)
```

### Connecting IBM Bob to the MCP server

Before you can ask questions, Bob needs to know where the MCP server is. Follow these steps:

**Step 1 — Start the MQ MCP server**

```bash
cd mq-mcp-server
export MQ_USERNAME=mqreader   # or your mqweb username
export MQ_PASSWORD=mqreader   # or your mqweb password
uv run mqmcpserver.py
```

The server starts listening on `http://127.0.0.1:8000/mcp`.

**Step 2 — Create `.bob/mcp.json`** in your project root

You can do this via **Bob Settings → MCP → Edit Project MCP**, or create the file manually:

```json
{
  "mcpServers": {
    "mqmcpserver": {
      "type": "http",
      "url": "http://127.0.0.1:8000/mcp"
    }
  }
}
```

**Step 3 — Verify the connection**

In Bob's MCP settings panel, `mqmcpserver` should appear as connected with two tools: `dspmq` and `runmqsc`. You can now ask Bob natural-language questions about your MQ environment (see [Example conversations](#example-conversations) below).

### Multi-server support

`MQ_SERVERS` is a list — one entry per mqweb endpoint. This allows the server to cover an entire **Uniform Cluster with Native HA** (e.g. 2 QMs × 3 nodes = 6 entries):

- **`dspmq`** queries all servers in **parallel** using `asyncio.gather` and labels each result by URL
- **`runmqsc`** iterates servers in order with a **3-second connect timeout**, automatically skipping unreachable nodes (`ConnectError`/`ConnectTimeout`) and servers that don't host the target queue manager (HTTP 404)

### TLS verification

```python
MQ_CA_BUNDLE = os.environ.get("MQ_CA_BUNDLE")
MQ_VERIFY = ssl.create_default_context(cafile=MQ_CA_BUNDLE) if MQ_CA_BUNDLE else False
```

Set `MQ_CA_BUNDLE` to your internal CA certificate (PEM) to validate mqweb's TLS certificate. Leave unset for zero-config demo use (TLS verification disabled — not for production).

### Credentials

```python
MQ_USERNAME = os.environ.get("MQ_USERNAME", "mqreader")
MQ_PASSWORD = os.environ.get("MQ_PASSWORD", "mqreader")
```

Credentials are read from environment variables — never hardcoded. Individual `MQ_SERVERS` entries can include `"username"`/`"password"` keys to override per server.

---

## Example conversations

Once connected to IBM Bob, you can ask natural-language questions. Here are real examples:

### List queue managers

> **"List my queue managers"**

```
=== https://localhost:9443/ibmmq/rest/v3/admin/ ===
---
name = QM1, running = running
---
=== https://localhost:9446/ibmmq/rest/v3/admin/ ===
---
name = QM2, running = running
---
```

---

### Check Native HA status

> **"Verify configuration of HA group, which one is the active node?"**

Bob runs `DISPLAY QMSTATUS TYPE(NATIVEHA) NHATYPE(INSTANCE) ALL` and returns:

| Instance | Role | Status | In Sync | Backlog |
|---|---|---|---|---|
| `node-1` | 🟢 **ACTIVE** | NORMAL | — | — |
| `node-2` | REPLICA | NORMAL | ✅ Yes | 0 |
| `node-3` | REPLICA | NORMAL | ✅ Yes | 0 |

---

### Check uniform cluster balance

> **"Check the uniform cluster status. What apps are connected? Are they balanced between the 2 queue managers?"**

Bob discovers the cluster name, queries both QMs, and reports:

| Queue Manager | App | Connections |
|---|---|---|
| QM1 | `keshi` | 5 |
| QM2 | `keshi` | 5 |

✅ **Perfectly balanced — 50% / 50%**

---

### Check queue depth

> **"Show me the queue with the maximum queue depth"**

Bob runs `DISPLAY QLOCAL(*) MAXDEPTH CURDEPTH` and ranks results:

| Queue | Current Depth | Max Depth |
|---|---|---|
| `DEV.DEMO.QL.IN` | **100** | 5,000 |
| `AMQ.6A7ED3C225049202` | 13 | 5,000 |

---

### Failover scenario

> **"I downed QM1 — check jack app status and balancing"**

Bob checks both QMs and reports:

| Queue Manager | `jack` connections | Status |
|---|---|---|
| QM1 | 0 | ❌ Down |
| QM2 | **16** | ✅ All reconnected |

All 16 workload connections automatically failed over to QM2 thanks to `MQCNO_RECONNECT`.

> **"I restarted QM1 — check balance again"**

| Queue Manager | `jack` connections |
|---|---|
| QM1 | 5 |
| QM2 | 5 |

✅ **Rebalanced automatically** after QM1 rejoined the uniform cluster.

---

## Connecting to IBM Bob

1. Start the MQ MCP server:
   ```bash
   cd mq-mcp-server
   export MQ_USERNAME=mqreader
   export MQ_PASSWORD=mqreader
   uv run mqmcpserver.py
   ```

2. Create `.bob/mcp.json` in your project root:
   ```json
   {
     "mcpServers": {
       "mqmcpserver": {
         "type": "http",
         "url": "http://127.0.0.1:8000/mcp"
       }
     }
   }
   ```

3. Ask Bob anything about your MQ environment.

→ Full setup guide: **[mq-mcp-server/README.md](mq-mcp-server/README.md)**
