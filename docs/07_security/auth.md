# Authentication
status: LOCKED

## Scope
Defines account identity, login credentials, access/refresh lifecycle, gameplay ticket issuance, linking boundaries, and revocation.

Decision: `../11_decisions/0009-account-session-credentials.md`.

## Canonical Identity
`account_id` is the persistent account identity.

Login provider identity is a credential/link to an account, not the account itself.

Characters, currencies, cosmetics, account storage, and other persistent ownership reference canonical account/character IDs rather than email, platform handle, or provider username.

## HTTPS Login Flow
1. Unity connects to authenticated HTTPS endpoint.
2. Client presents a configured login-provider credential.
3. Go backend/provider adapter verifies credential.
4. Backend resolves exactly one canonical `account_id`.
5. Backend issues access + rotating refresh credentials.
6. Client may request a short-lived gameplay connection ticket.
7. Gameplay ticket is consumed by the WSS handshake.

Realtime gameplay packets never carry a password or long-lived refresh credential.

## Credential Types
Launch defaults:
```text
access credential TTL     = 15 minutes
gameplay ticket TTL       = 60 seconds, single-purpose
resume credential TTL     = 10 minutes
refresh credential TTL    = 30 days maximum, rotating
```

TTL values may be shortened by security config, but widening them requires security review.

## Opaque Tokens
Public bearer credentials are high-entropy opaque values.

Server stores only the representation required for verification/revocation; refresh/resume secrets are never stored in plaintext recoverable form.

Do not place gameplay authority claims such as currency, inventory, role, map, or character stats inside client-trusted token payloads.

## Refresh Rotation
On successful refresh:
- issue a new refresh credential,
- invalidate/rotate the previous credential,
- detect reuse of an already-rotated credential,
- revoke the affected session family on suspicious reuse.

Retry safety must distinguish a lost response from true token replay.

## Login Providers
The account model supports verified external/OIDC/platform identities through explicit provider records.

A provider record is unique by:
```text
provider_id + provider_subject
```

Linking/unlinking requires reauthentication/verification and must never remove the account's last usable login method (there is no recovery path at launch).

## Password Provider (ADR-0051)
Provider id `password`. Enabled at launch beside `apple`, `google`, `steam`.

HTTPS endpoints (same TLS/HTTPS surface as federated login):
```text
POST /api/v1/auth/password/register   body: username, password, email
POST /api/v1/auth/password/login      body: username, password
```
Both return the same access + rotating refresh credentials as federated login (steps 5-7 of the HTTPS Login Flow).

Registration creates a new `accounts` row and one `account_password_credentials` row in one transaction. Adding a password to an existing federated account is not supported at launch; a password account may link federated providers under the normal linking rules.

Username:
```text
input trimmed of leading/trailing whitespace
length 4..20
charset after lowercasing: [a-z0-9_]
must start with a letter
username_key = ASCII lowercase(username)
UNIQUE(username_key)
reserved/profanity check per ../06_data/text.md on username_key
```

Email:
```text
required; trimmed; length <= 254
exactly one '@'; local part 1..64; domain has >= 1 '.', no leading/trailing '.' or '-' per label
any provider (not limited to Gmail)
email_key = ASCII lowercase(email); no dot/plus-alias folding
UNIQUE(email_key)
NOT verified; never a login identifier; not used for recovery
```

Password:
```text
length 8..128 Unicode code points after NFC (../06_data/text.md)
must not equal username_key or email_key case-insensitively
no composition rules; no silent truncation
```

Hashing:
```text
algorithm        = Argon2id (golang.org/x/crypto/argon2, pin in ../00_context/technology_versions.md)
params_version 1 = memory 19456 KiB, iterations 2, parallelism 1, salt 16 bytes crypto/rand, key 32 bytes
stored           = PHC string in password_hash + params_version
comparison       = constant time
rehash           = on successful login when params_version < current
```

Errors:
- login with unknown username or wrong password -> `AUTH_INVALID` (same shape and similar latency; hash a dummy value for unknown usernames),
- registration -> `USERNAME_INVALID`, `EMAIL_INVALID`, `PASSWORD_INVALID`, `USERNAME_TAKEN`, `EMAIL_TAKEN` (`../05_network/errors.md`),
- banned/suspended accounts -> `ACCOUNT_BANNED` / `ACCOUNT_SUSPENDED` after a correct password.

Recovery: none at launch. No password reset by email or support. An account with a linked federated provider can still log in through it. The client registration screen states that a forgotten password cannot be recovered unless a provider is linked.

Password change while logged in requires the current password and revokes all other refresh/session families of the account.

Rate limits: `../07_security/rate_limits.md` (Authentication).

## Operator (Admin/GM) Accounts
Operators are not player accounts and never share credentials with them.
```text
table              operators(operator_id, login_key, password_hash (Argon2id, same params), totp_secret_encrypted (AES-256-GCM, key from secret env OPERATOR_TOTP_KEY),
                             role, status, created_at, last_login_at)
login              password + TOTP (RFC 6238, 30 s step, 6 digits; implemented with Go crypto/hmac + crypto/sha1)
network            admin HTTPS API bound to the private operations network only, never the public listener
session            8 h absolute, 30 min idle; separate from player sessions
roles              SUPPORT     read-only lookup
                   MODERATOR   mute / suspend / BANNED status changes
                   ECONOMY     rollback or grant of items/currency
                   ADMIN       operator management, config
two-person rule    ECONOMY grants/rollbacks above 1,000,000 common or any item of tier >= T5 need a second operator's approval
audit              every call writes audit_events(actor = operator_id, reason, ticket_id, before/after, operation_id)
```
Setting `accounts.status = BANNED` or `SUSPENDED_*` is only possible through this API and revokes all player sessions.

## Revocation
Revocation scopes include:
- one access/session family,
- one device/session family,
- all sessions for account,
- one linked provider credential.

Security-sensitive account changes may revoke all refresh/gameplay/resume credentials.

Already-bound realtime sessions are notified/closed when their session authority is revoked.

## Storage
PostgreSQL may store:
- canonical accounts,
- provider links,
- hashed verifier material,
- session/refresh family metadata,
- revocation state,
- audit metadata.

Do not query PostgreSQL to revalidate the access token on every gameplay message; the accepted WSS session is bound to a current session epoch.

## Recovery
Account recovery is a privileged authentication flow and must not bypass normal ownership checks.

Recovery actions are audited and invalidate superseded credentials.

## Invariants
- canonical identity = account_id,
- provider handle/email is not persistent ownership identity,
- gameplay WSS never receives password/refresh credential,
- refresh credentials rotate,
- gameplay ticket is short-lived and single-purpose,
- credential revocation can terminate active authority,
- password plaintext storage is forbidden,
- username_key and email_key are each unique across accounts,
- email is unverified and never used for login or recovery.
