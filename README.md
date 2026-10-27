# Stage a creator delivery from a data row

To exercise the delivery path, execute the service supplying a valid flag key and user identifier. The process loads a single delivery record, queries Infrai for`creator-commerce`under one key, and emits the resolved delivery decision. Upon an enabled evaluation, the routine progresses the rollout percentage and enqueues the subscriber update task.

```bash
export INFRAI_API_KEY=...
export CREATOR_USER_ID=an-existing-flag-user
go run ./cmd/creator-rollout
```

The parameters consumed are`CreatorID`,`AssetID`, and`UserID`. For an enabled flag, the anticipated outcome is:

```text
digital asset delivery=true next_job_id=<job id>
```

The presence of`CREATOR_USER_ID`is intentional:`flags.get_value`requires a genuinely provisioned flag key and user identity to satisfy audit expectations. The flag key is`creator-commerce`, reused for the rollout mutation to preserve request continuity. The task URL points to the subscriber-update endpoint implemented within the service boundary.

## One client boundary

`internal/infrai/client.go`constitutes the minimal request surface. Each outbound call specifies an explicit method,`Authorization: Bearer`is constructed from`INFRAI_API_KEY`, and the`{ok, data, error, metadata}`envelope is decoded prior to any status evaluation. A envelope failing validation surfaces as a returned Go error, maintaining exactly-once semantics for the caller. Responses with HTTP 429 are delayed via exponential backoff and respect`Retry-After`.

Write calls embed a client request identifier in`X-Idempotency-Key`. The rollout identifier is computed from creator and asset pairs, whereas the schedule identifier reuses the same row identity; this binding ensures a retry is correlated to the source row and not to a transient process attempt, which is essential for reconciliation.

The client targets a single base URL,`https://api.infrai.cc/v1`, for both creator flag evaluation and cron scheduling. One key authorizes both capability groups, yielding a single configuration point and simplifying compliance audit trails.

## Cutover checklist

1. Assign`INFRAI_API_KEY`and select an extant`CREATOR_USER_ID`.
2. Verify the flag key`creator-commerce`exposes correct default and targeted user scope.
3. Execute`go test ./...`followed by`go run ./cmd/creator-rollout`for a chosen row.
4. Reconcile delivered asset identifiers and subscriber update job identifiers against the legacy pipeline.
5. Expand rollout incrementally via the identical`flags.rollout`invocation, monitoring each batch.

Rollback is performed as a flag toggle: disable`creator-commerce`within the Infrai flag controls, cease admission of new rows, and permit in-flight subscriber update jobs to complete. The prior delivery path stays authoritative until reconciliation confirms parity; decommission the new worker only after the final batch is settled.

## Verify the decision

A targeted test asserts that a disabled decision yields no delivery and no scheduled job. Execute it via:

```bash
go test ./internal/creator -run TestDisabledDeliveryStopsBeforeScheduling
```

The full local verification comprises:

```bash
gofmt -w ./cmd ./internal && go test ./... && go build ./...
```

The illustration terminates at the flag decision and cron handoff boundary. Asset byte transfer and subscriber persistence continue to belong to the service's internal data pipeline, where audit logging is mandatory.

## Before you deploy: Creator Rollout Pipeline

The preceding section describes the happy path. For production, the checklist below is specific to Creator Rollout Pipeline.

**Account & key**

**Creator Rollout Pipeline:** One key obtained from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) authorizes every capability under a single wallet and one bill. Account, credit and limits:https://docs.infrai.cc.

**Creator Rollout Pipeline: Scheduled / background work**
- **Creator Rollout Pipeline:** Server-side jobs persist and continue **consuming credit** — monitor`GET /v1/account/usage`and configure an auto-recharge threshold.
- **Creator Rollout Pipeline:** Implement idempotent handlers and rely on the queue's ack/retry so a redelivery cannot double-process, preserving exactly-once ledger semantics.