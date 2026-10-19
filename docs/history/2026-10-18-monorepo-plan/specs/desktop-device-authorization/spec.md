## ADDED Requirements

### Requirement: Desktop connection delegates existing browser identity with explicit approval

The system SHALL expose a desktop connection page using the existing LLVY browser login and Discord callback. A valid browser session SHALL be sufficient to review the request without another Discord exchange. The page SHALL display the account, application, restricted upload scope and transient/persistent mode with its lifetime and require an explicit, same-origin CSRF-protected approval POST. The mode SHALL be bound to the connection request and SHALL NOT change during code exchange. Current admin/owner authority SHALL be required both at approval and token exchange. GET requests and viewer sessions SHALL NOT authorize a device.

#### Scenario: Existing authenticated browser

- **WHEN** an admin opens an unexpired desktop connection request while already logged into LLVY
- **THEN** the browser displays the approval page and creates no device until the user approves

#### Scenario: Login is required

- **WHEN** the browser has no valid session
- **THEN** the existing Discord login returns to a sanitized internal connection page and retains its normal guild admission checks

#### Scenario: Permission is revoked before exchange

- **WHEN** authority or the approving browser session is revoked after approval but before the code is exchanged
- **THEN** the exchange fails without creating a device session or issuing tokens

### Requirement: Authorization codes are bound to PKCE and a literal loopback callback

The client SHALL bind `127.0.0.1` on an ephemeral port before login and use cryptographically random state and PKCE S256. The server SHALL accept only the registered application ID and `http://127.0.0.1:<port>/callback`, preserving the exact redirect URI and challenge through exchange. Connection requests SHALL expire after ten minutes; authorization codes SHALL expire after sixty seconds and be consumed atomically at most once. Only code and state SHALL return through the browser. The client SHALL validate callback structure, Host, path and state and close the listener on completion, cancellation or timeout.

#### Scenario: Concurrent code exchange

- **WHEN** two requests attempt to exchange one valid authorization code
- **THEN** at most one receives tokens and the other receives a stable authentication error

#### Scenario: Forged callback or redirect

- **WHEN** the request uses a non-loopback host, different callback path, malformed or duplicate parameters, mismatched state, wrong verifier or expired code
- **THEN** no device credentials are issued and no caller-selected external redirect is followed

### Requirement: Device credentials are restricted and independently revocable

The system SHALL issue opaque access and rotating refresh credentials with at least 256 bits of randomness, store only domain-separated keyed hashes, and associate them with a distinct desktop-kind DB session. Access lifetime SHALL be at most fifteen minutes and SHALL NOT exceed device expiry. A transient device session without login persistence SHALL have a maximum absolute lifetime of twelve hours; a persistent session SHALL have a maximum absolute lifetime of thirty days. Neither lifetime SHALL slide on refresh. `AUTH_SECRET` rotation SHALL invalidate old device credentials. Each account SHALL have at most five active, unexpired device sessions across both modes, enforced atomically under the account lock. Credentials SHALL permit only replay upload, own-job status and own-device management; browser cookie authentication SHALL not accept them.

#### Scenario: Device token targets a management API

- **WHEN** an owner-connected app sends its bearer credential to a member-management, grant-management or audit endpoint
- **THEN** the credential grants no access to that endpoint despite the account's owner role

#### Scenario: Wrong credential type

- **WHEN** a refresh token is used as an access token or a device token is placed in the web session cookie
- **THEN** it is rejected without changing existing browser authentication behavior

#### Scenario: Device limit is reached

- **WHEN** approval would create a sixth active device for an account
- **THEN** no new device is issued without a slot, and the browser offers explicit replacement of an existing transient connection or separate device disconnection rather than automatic eviction

#### Scenario: Abandoned transient connections occupy all slots

- **WHEN** the user selects an existing transient connection in the same account for replacement during browser approval
- **THEN** its session ID is bound to that approval, and code exchange revalidates the binding and atomically revokes it, issues the new session and records audit under the account lock; exchange failure preserves the old connection

### Requirement: Refresh rotation rejects replay and does not extend authority

Refresh SHALL atomically consume one credential generation and issue the next only while its account, device session, guild and original role/grant remain valid. Reuse of a consumed refresh token SHALL revoke that device credential family. Its consumed keyed hash and family mapping SHALL remain until family absolute expiry, unless the family has been permanently revoked and all of its credentials remain rejected. The client SHALL serialize refresh operations within its sole per-user/origin instance and require reauthentication after an uncertain refresh response rather than blindly replaying it. Refresh SHALL not extend the device's absolute expiry or restore a revoked grant.

#### Scenario: Concurrent or replayed refresh

- **WHEN** the same refresh credential is exchanged again after its successful consumption
- **THEN** the token family is revoked and no independent second active family is created

#### Scenario: Grant is removed and reissued

- **WHEN** an admin receives a new grant after the previous one was revoked
- **THEN** old device credentials remain invalid and a fresh browser approval is required

#### Scenario: Consumed token is replayed after maintenance

- **WHEN** maintenance has run after a refresh rotation and an earlier generation is presented while the family remains unexpired and active
- **THEN** its retained hash/family mapping identifies the replay and the current credential family is revoked

### Requirement: Current database authority governs all device operations

Every device API SHALL check active account/session, current guild and owner/grant, original delegated authority and required scope. Shared mutation transactions SHALL revalidate these under the common authority lock order through commit. Discord availability or old membership evidence alone SHALL not invalidate an otherwise active delegated session. Confirmed account departure, grant revocation or explicit device disconnection SHALL prevent subsequent authorized processing and persistence.

Both explicit administrator revocation and membership-based suspension SHALL invalidate affected desktop sessions/credentials and unfinished desktop pending uploads/jobs, including processing claims, atomically with the authority change and audit. Grant revocation SHALL target that grant's delegated bindings; account suspension SHALL cover the account's bindings. These paths SHALL share sorted account/session/grant/desktop/pending-job lock ordering with mutations. Completed results, canonical game objects and historical audits SHALL be retained; credential rejection SHALL be immediate while physical record deletion follows retention rules.

Because owner authority is environment-configured, every supported owner change and rollback SHALL first disable desktop operations on all reachable serving deployments, drain issuance, and permanently revoke all desktop sessions and unfinished connection requests in an audited database transaction. Only after changing the configuration and verifying old credentials are rejected MAY desktop operations be reenabled. This procedure SHALL preserve browser sessions. An owner change SHALL remain blocked if all-serving-deployment enforcement cannot be established; arbitrary environment edits SHALL NOT be described as automatically detected.

#### Scenario: Revocation races replay persistence

- **WHEN** device revocation and replay persistence overlap
- **THEN** persistence commits before the revocation completes or is rejected after revocation wins, with no newly authorized commit after completed revocation

#### Scenario: Owner explicitly revokes an administrator with unfinished device uploads

- **WHEN** administrator revocation wins before the old grant's desktop work commits
- **THEN** its sessions, credentials and pending/processing desktop bindings are invalidated in the same audited transaction, and refresh, transfer and finalization remain denied after a later regrant

#### Scenario: Discord is unavailable after connection

- **WHEN** the account and delegated authority remain locally valid during a Discord outage
- **THEN** ordinary device upload APIs use current DB authority without making a Discord call

#### Scenario: Owner changes away and back without intermediate device traffic

- **WHEN** the supported owner-change procedure changes owner A to B and then back to A while A's old device sends no intermediate request
- **THEN** that device remains permanently revoked and fresh browser approval is required even if its original absolute lifetime has not elapsed

### Requirement: Browser and device logout have distinct explicit effects

The system SHALL allow an authenticated browser to list and revoke only its account's device connections and allow an app to revoke only its own session. Browser logout SHALL not silently revoke all independent devices, and device logout SHALL not clear browser sessions. Local device logout SHALL always remove locally retained secrets even when server revocation fails, while reporting that server revocation was not confirmed. Account-wide suspension SHALL still revoke both session kinds.

#### Scenario: Another account requests a device connection

- **WHEN** an authenticated user attempts to list or revoke another account's device
- **THEN** the request is denied without revealing its connection metadata

### Requirement: Login persistence protects only LLVY credentials in user storage

The client SHALL default `로그인 유지` to off. Without it, credentials SHALL remain in memory, and normal exit SHALL attempt own-session revocation for at most two seconds before clearing them. Failure SHALL be reported without blocking exit indefinitely; crash/offline orphans SHALL be bounded by the transient lifetime and recoverable through explicit browser-approved transient replacement. With persistence, only the refresh credential SHALL be stored using non-interactive Windows user-scope DPAPI in origin-separated user-local storage; the access token and PKCE verifier SHALL remain in memory. Credential reads and atomic file replacement SHALL occur only while holding the per-user/origin single-instance lock. Failure to decrypt or save a newly rotated credential SHALL require login rather than plaintext fallback or reuse of an older generation. No Discord tokens, bot tokens, OAuth client secret or Blob store credentials SHALL be delivered to the exe.

#### Scenario: Saved credential cannot be decrypted

- **WHEN** saved credentials are unavailable, corrupted or unusable under the current Windows identity
- **THEN** the application clears the unusable saved credential and requests login without exposing its contents

#### Scenario: Default login and normal exit repeat six times

- **WHEN** the user repeats login, upload and normal exit six times with persistence off and server revocation available
- **THEN** each exit revokes the transient connection, so the sixth login is not blocked by five abandoned sessions

#### Scenario: Two exe copies attempt to use saved credentials

- **WHEN** two processes under the same Windows user and server origin start from different exe paths or versions
- **THEN** only the lock holder reads or rotates credentials, and the other process reports that the app is already running without replaying a refresh token

### Requirement: Public authentication endpoints and cleanup remain bounded

Connection start and code exchange SHALL share atomic trusted-IP/global budgets, initially ten per IP and one hundred globally per minute. Refresh SHALL have separate bounded device/IP/global budgets. Production requests without trusted ingress identity SHALL fail closed. Expired attempts and eligible credentials SHALL be cleaned in bounded batches without deleting active authorization state, required refresh/idempotency tombstones or retained replay results. Consumed refresh records SHALL follow the family retention requirement rather than access-token cleanup. Existing auth-session cleanup SHALL respect these child-record lifetimes and permanent credential rejection. Logs, audits and caches SHALL exclude raw credentials, codes, state, verifier and signed URLs.

#### Scenario: Caller spoofs forwarding headers

- **WHEN** a caller rotates raw forwarded headers or starts requests across serving instances
- **THEN** the verified client identity and shared database counters enforce the same budgets

#### Scenario: Provider outage during cleanup

- **WHEN** Discord is unavailable but authenticated maintenance can access the database
- **THEN** local desktop authorization cleanup proceeds independently without revoking unverified memberships
