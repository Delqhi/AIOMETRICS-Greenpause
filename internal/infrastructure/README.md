# Infrastructure Layer Contract

## Scope
- Implements adapters for databases, caches, brokers, external APIs, and telemetry.

## AdapterCatalog
- `PostgresReminderRepositoryAdapter`
- `RedisScheduleIndexAdapter`
- `JwtVerifierAdapter`
- `MtlsTransportAdapter`
- `NotificationProviderAdapter`

## SecurityControls
- Internal RPC clients require `mTLS` with workload identity.
- JWT verification requires signature key rotation and audience checks.
- Secrets loaded only from managed secret store at runtime.

## DataPartitioningStrategy
- Primary partition key: `TenantId`.
- Logical shard formula: `ShardId = Hash(TenantId) mod N`.
- Hot-tenant mitigation: dynamic shard split and tenant rebalancing runbook.
