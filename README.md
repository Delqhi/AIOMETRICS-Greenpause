# Greenpause Blueprint Repository

## PhaseStatus
- CurrentPhase: `Implementation`
- ImplementationGate: `UNLOCKED`
- GateCondition: Greenpause approved on `2026-03-01`; implementation active.

## RepositoryIntent
- Objective: Deliver a ScreamingArchitecture-compliant and HexagonalDDD-compliant reminder platform blueprint.
- PrimaryDomain: `ReminderManagement`
- SecondaryDomains: `NotificationDelivery`, `IdentityAndAccess`, `Auditability`

## ArchitectureDocuments
- SystemOverview (arc42 + C4 L1/L2): [`docs/architecture/system-overview.md`](docs/architecture/system-overview.md)
- ADR-0001 BaseArchitecture: [`docs/adr/0001-base-architecture.md`](docs/adr/0001-base-architecture.md)
- RFC-0001 CoreLogic: [`docs/rfc/0001-core-logic.md`](docs/rfc/0001-core-logic.md)

## DirectoryLayout
```text
09-Greenpause/
├── api/
│   └── openapi/
├── cmd/
│   ├── server/
│   ├── cli/
│   └── worker/
├── internal/
│   ├── domain/
│   ├── application/
│   └── infrastructure/
├── pkg/
│   ├── logging/
│   ├── auth/
│   └── timeutil/
├── docs/
│   ├── architecture/
│   ├── adr/
│   └── rfc/
├── deployments/
│   ├── terraform/
│   ├── kubernetes/
│   └── helm/
└── configs/
```

## DomainInvariants
- `ReminderId` uses UUIDv7 for monotonic index locality.
- `TenantId` is mandatory for every write/read path.
- `DueAtUtc` must be at least 30 seconds in the future at creation time.
- `ReminderStatus` transitions are restricted to:
  - `Scheduled -> Triggered`
  - `Scheduled -> Canceled`
  - `Triggered -> Acknowledged`
- `IdempotencyKey` is unique per `TenantId` for 24 hours.

## APIObjectNamingStandard
- Object naming convention: `UpperCamelCase`.
- Canonical objects: `ReminderCommandRequest`, `ReminderView`, `ReminderSchedule`, `ReminderDispatchEvent`, `AuthTokenClaims`.

## DevEnvironmentSetup
1. Toolchain requirements:
   - `git >= 2.45`
   - `docker >= 26`
   - `terraform >= 1.8`
   - `helm >= 3.15`
   - `kubectl >= 1.31`
   - `node >= 22` (for docs/tooling linters)
2. Environment bootstrap:
   - `cp configs/.env.development.template .env.local`
   - `cp configs/.env.staging.template .env.staging.local`
   - `cp configs/.env.production.template .env.production.local`
3. Documentation-first workflow:
   - Draft or update RFC in `docs/rfc/`
   - Approve ADR in `docs/adr/`
   - Update C4 diagrams in `docs/architecture/`
   - Unblock implementation gate only after review approval
4. Runtime bootstrap:
   - `go test ./...`
   - `APP_PORT=8080 STORAGE_BACKEND=memory SCHEDULE_BACKEND=memory go run ./cmd/server`
   - `curl -sS -X POST http://localhost:8080/v1/reminders -H 'content-type: application/json' -d '{\"TenantId\":\"tenant-a\",\"UserId\":\"user-1\",\"DueAtUtc\":\"2026-03-01T12:05:00Z\",\"Message\":\"Check email\",\"IdempotencyKey\":\"idem-12345678\"}'`
   - `curl -sS 'http://localhost:8080/v1/reminders/<ReminderId>?TenantId=tenant-a'`
5. Optional infrastructure backends:
   - Postgres adapter requires a registered SQL driver at runtime (`DATABASE_DRIVER`, `DATABASE_DSN`).
   - Redis schedule adapter requires `REDIS_ADDR` for `SCHEDULE_BACKEND=redis`.

## BuildPipelineCommands
- `markdownlint "**/*.md"`
- `npx @stoplight/spectral lint api/openapi/reminder-v1.yaml`
- `yamllint api configs deployments`
- `terraform fmt -check -recursive deployments/terraform`
- `helm lint deployments/helm`
- `kubeconform -strict deployments/kubernetes/*.yaml`

## NFRBaseline
- API latency budget: `p95 <= 120ms`, `p99 <= 250ms` under 2k RPS steady load.
- Scheduling precision budget: `p99 trigger jitter <= 3s`.
- Availability target: `99.95%` monthly for command/query API.
- Durability target: `RPO <= 5m`, `RTO <= 30m`.
- Security baseline:
  - East-West traffic: mandatory `mTLS`.
  - North-South auth: `JWT` access tokens (`RS256`, max lifetime `15m`).
  - Secrets: external secret manager only; no plaintext secrets in repository.

## RelatedContracts
- OpenAPI draft: [`api/openapi/reminder-v1.yaml`](api/openapi/reminder-v1.yaml)
