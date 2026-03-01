# Domain Layer Contract

## Scope
- Contains `Entities`, `ValueObjects`, and `DomainServices`.
- Must not import transport, persistence, or framework packages.

## DependencyRule
- Allowed dependencies: language standard library only.
- Forbidden dependencies: SQL drivers, HTTP clients, message brokers, cache clients.

## CoreDomainObjects
- `ReminderEntity`
- `ReminderSchedule`
- `ReminderStatus`
- `ReminderPolicy`
- `DispatchWindow`

## DomainInvariants
- `ReminderSchedule.DueAtUtc` must be normalized to UTC.
- `ReminderPolicy.MaxAttempts` range: 1..10.
- `ReminderEntity.Message` size: 1..1024 bytes UTF-8.
- `ReminderStatus` transition rules are immutable and centrally validated.
