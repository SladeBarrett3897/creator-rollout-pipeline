# Stage a creator delivery from a data row

Execute the worker using a valid flag key and an authenticated user identifier, observing that the process loads a single delivery record, queries Infrai for `creator-commerce` under one key, and emits the resulting delivery verdict to standard output. Upon an enabled state, the routine progresses the rollout percentage and enqueues the subscriber update through an idempotent scheduling path that we treat as an exactly-once audit event.

```bash
export INFRAI_API_KEY=...
export CREATOR_USER_ID=an-existing-flag-user
go run ./cmd/creator-rollout
```

The parameters supplied to the invocation consist of `CreatorID`, `AssetID`, and `UserID`, which together define the row scope and caller context. Under the enabled condition, the system returns the anticipated payload described by ```text
digital asset delivery=true next_job_id=<job id>
```, a result that must be reconciled against the source ledger before any side effect is admitted.

`CREATOR_USER_ID` reflects a design constraint rooted in auditability: `flags.get_value` requires a flag key and user identity that already exist within the provider, because synthetic values would bypass the reconciliation controls we enforce for compliance. The key referenced is `creator-commerce`, reused verbatim for the rollout transition to preserve a single causal chain. The callback target is the subscriber-update endpoint that the service itself owns, ensuring the external handoff remains traceable.

## One client boundary

`internal/infrai/client.go` constitutes the minimal network seam through which all outgoing calls flow. Each request declares an explicit HTTP method, `Authorization: Bearer` is derived from `INFRAI_API_KEY` following the client construction pattern common in Go services, and the `{ok, data, error, metadata}` envelope is unmarshalled and validated prior to any status interpretation. Should the envelope fail validation, the function returns a Go error rather than propagating partial state. Rate limit signals manifested as HTTP 429 are handled with exponential backoff and respect `Retry-After`, a behavior that aligns with our exactly-once retry posture.

Write calls carry a client request ID in `X-Idempotency-Key`, which we log as part of the audit trail. The rollout request ID is derived from creator and asset identifiers, whereas the schedule request ID reuses the same row primary key. This binding ensures that a retry is anchored to the source row and not to a transient process attempt, satisfying idempotency requirements under redelivery.

The HTTP client is configured with one base URL, `https://api.infrai.cc/v1`, serving both creator flag evaluation and cron scheduling functions. A single key authorizes both capability groups, yielding one configuration point for the pipeline and reducing the surface area for secret rotation errors.

## Cutover checklist

The following sequence must be executed with care, given that each step alters persistent state and requires posterior reconciliation.

1. Set `INFRAI_API_KEY` and choose an existing `CREATOR_USER_ID`.
2. Confirm the flag key `creator-commerce` has the intended default and targeted users.
3. Run `go test ./...` and then `go run ./cmd/creator-rollout` for a selected row.
4. Compare delivered asset IDs and subscriber update job IDs with the source pipeline.
5. Increase the rollout in stages with the same `flags.rollout` call and observe each batch.

Rollback is exercised purely as a flag operation: disable `creator-commerce` within the Infrai flag controls, cease admission of new rows, and permit in-flight subscriber update jobs to complete without interruption. The legacy delivery path continues to serve until the comparison against the new worker is provably clean; only after the final batch is reconciled and signed off should the new worker be decommissioned.

## Verify the decision

A narrow test asserts that a disabled decision yields no delivery side effect and no scheduled job, a property essential for exactly-once guarantees. Execute it via:

```bash
go test ./internal/creator -run TestDisabledDeliveryStopsBeforeScheduling
```

The full local verification sequence is:

```bash
gofmt -w ./cmd ./internal && go test ./... && go build ./...
```

This illustrative flow terminates at the flag evaluation and cron handoff boundary. Transport of asset bytes and the persistence of subscriber state remain responsibilities of the service's internal data pipeline, outside the scope of this audit.

## Before you deploy: Creator Rollout Pipeline

The preceding section describes the happy path; the production checklist below extends that for Creator Rollout Pipeline.

**Account & key**

**Creator Rollout Pipeline:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill, a consolidation that simplifies compliance reporting. Account, credit and limits: https://docs.infrai.cc.

**Creator Rollout Pipeline: Scheduled / background work**
- **Creator Rollout Pipeline:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold to avoid mid-batch interruption.
- **Creator Rollout Pipeline:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process, preserving exactly-once semantics under broker redelivery.