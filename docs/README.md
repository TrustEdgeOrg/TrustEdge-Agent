# TrustEdge Agent documentation

TrustEdge Agent is the EDR-lite cross-platform endpoint agent for the [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) security observability platform. Telemetry is sent to [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API).

## Guides

| Document | Description |
|----------|-------------|
| [Architecture](architecture.md) | End-to-end telemetry flow, batching, compression, concurrency |
| [Agent](agent.md) | Installation, platform support, collectors, credentials |
| [Configuration](configuration.md) | Environment variables for the agent |
| [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md) | HTTP endpoints, event types, payloads |

## Quick links

- **Run agent locally:** `TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080 go run ./cmd/trustedge-agent`
- **Run ingest API locally:** [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) — `go run ./cmd/trustedge-agent-api`
- **Build all platforms:** `make build-all`
- **Tests:** `make test`

## Related repos

| Repo | Role |
|------|------|
| [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) | Ingest API, Redis/Kafka, ECR deploy |
| [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) | Dashboard, detection engine, policy, docker-compose stack |
| [TrustEdgeClient](https://github.com/TrustEdgeOrg/TrustEdgeClient) | Optional VPN enroll client |
