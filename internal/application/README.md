# Application Layer Contract

## Scope
- Hosts `UseCases`, `Commands`, `Queries`, and transaction boundaries.
- Orchestrates domain objects through `Ports` interfaces.

## UseCaseCatalog
- `ScheduleReminderUseCase`
- `CancelReminderUseCase`
- `AcknowledgeReminderUseCase`
- `GetReminderViewQuery`

## PortTypes
- `ReminderRepositoryPort`
- `ReminderDispatchPort`
- `ClockPort`
- `IdempotencyPort`
- `AuditLogPort`

## NonFunctionalConstraints
- Command execution budget: `p95 <= 60ms` (excluding external provider latency).
- Query execution budget: `p95 <= 40ms`.
- All command handlers are idempotent by `(TenantId, IdempotencyKey)`.
