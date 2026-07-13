# TrustEdge Agent documentation

TrustEdge Agent is the EDR-lite endpoint agent and ingest API for the [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) security observability platform.

## Guides

| Document | Description |
|----------|-------------|
| [Architecture](architecture.md) | End-to-end telemetry flow, batching, compression, concurrency |
| [Agent](agent.md) | Installation, platform support, collectors, credentials |
| [Configuration](configuration.md) | Environment variables for agent and API |
| [API reference](api.md) | HTTP endpoints, event types, payloads |
| [AWS deploy](../aws/README.md) | ECR build, EC2 deploy, GitHub Actions secrets |

## Quick links

- **Run agent locally:** `TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080 go run ./cmd/trustedge-agent`
- **Run API locally:** `go run ./cmd/trustedge-agent-api`
- **Build all platforms:** `make build-all`
- **Tests:** `make test`

## Related repos

| Repo | Role |
|------|------|
| [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) | Dashboard, detection engine, policy, docker-compose stack |
| [TrustEdgeClient](https://github.com/TrustEdgeOrg/TrustEdgeClient) | Optional VPN enroll client |
