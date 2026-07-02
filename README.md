# Payments Engineering Challenge — Build a Payments Ledger (Go)

Welcome, and thanks for taking the time.

**Time box:** ~60 minutes. We care far more about your **design decisions and reasoning**
than about how much code you write. You'll walk us through it live, and you're welcome to
use AI tools.

All code lives in **`challenge/`** : a `go.mod`, the provided wiring, and where you add your
own code. Run everything from there (see [Running](#running)).

## The problem

A platform sits between customers and merchants. A customer pays; the platform keeps a
**fee**; the remainder goes to the **merchant**. Some payments are later **refunded**, in
full or in part. The same instruction may arrive **more than once** (e.g. a payment
provider redelivers a webhook), and instructions may arrive **concurrently**.

Your job: build the **ledger** that records where money is at all times and supports two
operations: **capture** (record an incoming payment) and **refund** (return some or all
of it), such that money is never lost, created, or double-counted.

## What's provided (so you can focus on the ledger)

The plumbing is done for you — you only write ledger logic:

- **A running PostgreSQL** via `challenge/docker-compose.yml`.
- **A ready, verified connection pool**:  `connect()` (in `challenge/database.go`) opens and
  pings a `*pgxpool.Pool`, safe for concurrent use.
- **A `Ledger` type wired to the database**: `NewLedger(db)` in `challenge/ledger.go`,
  already initialized in `main`. Your `Ledger` holds the pool (`l.db`); build your operations
  as methods on it and run queries on `l.db`.
- **A starting test:  `challenge/ledger_test.go` has a `setup(t)` helper that opens a
  connection and returns a `Ledger`, plus a first passing `TestNewLedger`. Extend from there.
- **A `Makefile`**: `make start`, `make test`, `make stop` (below).

## Build on the seed: your design

The wiring above is the only starter code. How you model the ledger is entirely yours:

- How you model and run it is upto you. It just has to work and stay correct.
- Your database schema and the constraints/locking you rely on for integrity.
- The API/signatures of your operations, and whether you expose them via an HTTP API, a CLI,
  or drive them straight from tests, your call.
- Your concurrency strategy.

We only require the **behavior** and the **tests** below.

## Hard requirements

1. **Double-entry / conservation.** Every recorded money movement must balance: money
   leaving one place equals money arriving elsewhere. Across the whole ledger, money is
   conserved: nothing is created or lost.
2. **Capture** records a payment: customer pays `gross`, platform keeps `fee`
   (`0 ≤ fee ≤ gross`), merchant receives `gross − fee`.
3. **Refund** returns all or part of a capture. The **fee is returned proportionally** to
   the refunded amount. Cumulative refunds for a payment must never exceed its captured `gross`.
4. **Idempotency.** Ensure duplicate transactions are not double processed and there is no double spending.
5. **Concurrency-safe.** Correct under parallel use; `go test -race` must be clean.
6. **Persist in the provided PostgreSQL** through the `Ledger`'s pool. Beyond the provided `pgx` driver, prefer the standard
   library; add a dependency only if you can justify it. Go 1.26 (`go.mod` is at the
   challenge root, put your code alongside it).

## Deliberately your call: decide, and be ready to defend

Not oversights. Make a decision and justify it live:

- **Rounding.** When the proportional fee doesn't divide evenly (e.g. refunding 1/3 of a
  payment), where does the leftover cent go? What property does your choice preserve across
  many partial refunds?
- **Concurrency model.** One lock? Per-account locks? Optimistic/versioned? Trade-offs under
  load, and where the race windows are.
- **Idempotency scope.** What is uniqueness scoped to, and how do you identity when to enforece it vs not?
- **Error handling.** What's an error vs a silently-ignored duplicate?

## Tests you must write

**You write the tests.** Building on the provided `setup(t)` helper and `TestNewLedger` in
`challenge/ledger_test.go`, implement (at least) the test functions named below, reproducing
each scenario against your own ledger. Run them with `make test`, which starts the database
for you. You may add more. All must pass, and the concurrency test must pass under `-race`.

| Test function                               | Scenario                                                     | Expected / acceptance                                        |
| ------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `TestUnbalancedMovementRejected`            | Attempt to record a movement where money out ≠ money in (or total is 0). | Rejected with an error; **no** balances change.              |
| `TestCaptureRecordsPayment`                 | Capture `gross=1000, fee=100` for a merchant.                | Merchant +900, platform fee +100; the movement balances; total money in the system unchanged. |
| `TestCaptureIsIdempotent`                   | Same capture delivered twice.                                | Applied exactly once (merchant +900, one transaction recorded); second call returns the original result. |
| `TestFullRefund`                            | Capture `1000/fee 100`, then fully refund `1000`.            | Merchant net 0, platform-fee net 0, full `1000` returned to the payment source; ledger balanced. |
| `TestPartialRefundReturnsFeeProportionally` | Capture `1000/fee 100`, then refund `500`.                   | Fee returned proportionally (`50`); merchant net `450`; `500` returned to source; ledger balanced. |
| `TestRefundIsIdempotent`                    | Same refund request delivered twice.                         | Applied exactly once; no double refund; second call returns the original result. |
| `TestRefundCannotExceedCapture`             | Refund more than the captured amount (in one call, or via cumulative partials). | Rejected with an error; cumulative refunds never exceed captured `gross`; no balances corrupted. |
| `TestRoundingConservesMoney`                | Capture where the fee doesn't divide evenly (e.g. `gross=1000, fee=100`), then several partial refunds that sum to the full `gross`. | After all refunds: merchant net 0, platform-fee net 0, source net 0 — **not a single cent created or lost**, whatever your rounding policy. |
| `TestConcurrentOperations`                  | Many captures (and refunds) issued from parallel goroutines, including duplicate requests concurrently. | Final balances exactly correct; money conserved; no lost updates or double-applies. **Must pass `go test -race`.** |
| `TestMoneyConserved` *(recommended)*        | After an arbitrary sequence of captures and refunds.         | The entire ledger balances. (every debit has a matching credit). |

## Running

Requires Docker + Docker Compose and Go 1.26. From the `challenge/` directory:

```bash
make start   # starts PostgreSQL (background) and runs the app; prints "application started"
make test    # starts PostgreSQL (background) and runs all tests with the race detector
make stop    # stops the app, then shuts the database down
```

`make test` brings the database up for you, so your tests run against real PostgreSQL. If
you prefer to run tests directly, start the DB first (`make start`) and use
`go test -race ./...`. The connection string is `DATABASE_URL` (defaults to the local
Compose database).

## How we evaluate

This drives a live discussion: expect us to ask *why* on every meaningful choice and to
add new constraints mid-session.

- **Data integrity**: money conserved; movements always balance; no lost updates.
- **Idempotency**: duplicates are genuine no-ops, including under concurrency.
- **Money math**: proportional split and rounding are correct and conservative.
- **Concurrency**: correct under `-race`, with a defensible strategy and clear race analysis.
- **Schema & durability**: sensible tables/constraints; taking advantage of database solutions.
- **Judgment**: how you reason about the open decisions, edge cases, and how this would grow
  into a production system (reconciliation, failure recovery, scale).
- **Clarity**: code and tests a teammate could pick up.

## Using AI

You may use AI assistants, we do. The exercise is built so AI can produce code quickly
while the **decisions and their justification** stay yours. Understand everything you
submit; we'll probe it. You should explain your decisions before you execute
them rather than after the AI has built it.