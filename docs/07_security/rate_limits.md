# Rate Limits
status: LOCKED

## Scope
Defines launch rate-limit classes, default budgets, burst handling, and failure behavior.

Rate limiting protects service availability and abuse surfaces. It does not replace gameplay-state validation.

## Dimensions
Limits may apply by:
- connection/session,
- account,
- character,
- source IP/network,
- operation/resource key.

No single dimension is sufficient for every abuse case.

## Realtime Defaults
Per attached gameplay session:
```text
C2S_INPUT_STATE      sustained 30/s, burst 60
C2S_MOVEMENT_EDGE    sustained 20/s, burst 30   (never coalesced; excess edges are rejected, not merged)
discrete combat/input sustained 20/s, burst 30
interaction/portal   sustained 10/s, burst 20
durable mutations    sustained 10/s, burst 20
heartbeat            only configured cadence + small tolerance
auction search       sustained 1/s, burst 5
```

HTTPS webhooks (`external_integrations.md`): Apple/Google IAP notifications are accepted only after signature/OIDC verification; unauthenticated requests are limited to 60/min per source IP and verified notifications to 600/min total, excess answered `429` so the store retries.

These limits are intentionally above legal normal gameplay frequency and can be runtime-tuned after telemetry.

A rate limit never makes an otherwise illegal skill/action legal.

## Chat
Default character/account chat submission:
```text
short burst = 5 messages / 10s
sustained   = 20 messages / 60s
```

Channel-specific stricter limits are allowed.

## Authentication
Authentication endpoints have stricter account/IP/network limits and progressive backoff.

Exact provider-sensitive values are runtime security configuration so they can react to attack patterns without a game client patch.

Login error shape must not enable account enumeration.

Password provider (ADR-0051) classes, values in runtime security config:
- `auth.password.login`: per IP and per `username_key`, progressive backoff after consecutive failures,
- `auth.password.register`: per IP; registration is the only path that reveals `USERNAME_TAKEN` / `EMAIL_TAKEN`, so this limit is the enumeration control.

## Per-Operation Sub-limits
The `durable mutations` bucket (sustained 10/s, burst 20) sets the aggregate ceiling. The following operations carry **stricter per-character sub-limits** that bind before the aggregate bucket:

```text
Operation                  Sub-limit             Rationale
---------------------------------------------------------------------------
direct trade invitation    3 / 60 s per          A griefer spamming trade dialogs during
  (outbound, per sender)   sending character     PvP or boss fights is disruptive even if
                                                 the server rejects each trade. Normal
                                                 behavior is <= 1 invite per minute.

auction listing creation   10 / 60 min per       20-listing cap makes > 10 listings/h
                           character             unusual except at session start; prevents
                                                 rapid relisting bursts.

auction purchase           20 / 60 s per         Legitimate buyers browse then buy; 20/min
  (C2S_AUCTION_BUY 732)    character             is generous. Prevents automated purchase
                                                 burst that could starve normal players.

reward claim               30 / 60 s per         Players claim rewards in short bursts;
                           character             30/min covers a full session catch-up.
                                                 Prevents bulk-claim automation loops.

party / guild / friend     10 / 60 s per         Covers both outbound and processed
  invite (any social        sending character     inbound. Normal social interaction is
  invite type)                                   well under 2/min; 10/min allows burst
                                                 at guild-formation time.
```

Sub-limit violations return `RATE_LIMITED` with the specific operation name in the retry hint. Repeated sub-limit violations (> 3x in a session) escalate to the session-level throttle and may trigger a security anomaly signal.

## Expensive Operations
Search/list endpoints such as Auction queries and broad social lookups use separate lower request budgets, pagination, and response caps.

Never allow an unbounded query just because the caller is authenticated.

## Enforcement
On limit exceed:
1. reject current operation with `RATE_LIMITED`,
2. return bounded retry hint when safe,
3. repeated abuse may throttle/close the connection,
4. security controls may temporarily restrict account/IP/device source.

Do not queue unlimited excess requests.

## Fairness
Normal latency/reconnect duplicate traffic should be tolerated within sequence/idempotency rules.

A one-off duplicate or stale packet is not itself cheating.

## Distributed Enforcement
Launch correctness must not depend on a distributed cache.

All limits run in the single world process (ADR-0052). Account/IP auth limits that must survive a restart use the PostgreSQL L2 counter (`external_integrations.md`).

## Metrics
Track:
- rejects by limit/message family,
- top sources by rejected rate,
- false-positive/support reports,
- queue depth and CPU before/after throttling,
- auth abuse patterns separately from gameplay spam.

## Invariants
- every public endpoint/message family is bounded,
- realtime input cannot create unbounded queue growth,
- rate limit does not replace gameplay validation,
- excess work is rejected/backpressured rather than buffered,
- auth enumeration through error differences is forbidden,
- trade invitation sub-limit = 3 per 60 s per sending character,
- auction listing creation sub-limit = 10 per 60 min per character,
- auction purchase sub-limit = 20 per 60 s per character,
- reward claim sub-limit = 30 per 60 s per character,
- social invite (any type) sub-limit = 10 per 60 s per sending character.
