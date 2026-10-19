## ADDED Requirements

### Requirement: Client authentication follows the LLVY-owned desktop protocol

The client SHALL follow the LLVY-owned desktop-device-authorization contract for public-client browser approval, exact loopback callbacks, PKCE S256, mode binding, scoped tokens and revocation. It SHALL bind its loopback listener before requesting approval, validate callback Host/path/state/structure, close it on completion or timeout, serialize refresh within the single instance and require login after an uncertain refresh. Local logout SHALL remove local secrets even if remote revocation fails and report that outcome. Client builds SHALL not contain Discord, Blob or server signing secrets.

#### Scenario: Forged browser callback reaches the client

- **WHEN** a callback has an unexpected Host, path, state or duplicate parameters
- **THEN** the client rejects it without exchanging a code or persisting a credential

#### Scenario: Refresh response is uncertain

- **WHEN** the client cannot establish whether its refresh succeeded
- **THEN** it pauses authorized work and requests login without replaying the old credential

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

### Requirement: Processing starts with sufficient credential lifetime

The client SHALL reserve at least two minutes of access-token validity before a process request, using serialized refresh when needed. If refresh cannot supply that lifetime because the device is expiring, it SHALL require login before starting processing. Cancellation SHALL preserve already completed local logout, record deletion and authentication state changes.

#### Scenario: Token would expire during processing

- **WHEN** access expires in 35 seconds and the user starts processing
- **THEN** the client refreshes before processing and does not begin a claim with the old credential

#### Scenario: Logout is canceled during server revocation

- **WHEN** cancellation interrupts server revocation after local logout is requested
- **THEN** credentials are removed and the UI shows signed out while retaining the revocation failure notice

### Requirement: Local connection and server snapshot are distinct states

The UI SHALL synchronize the locally retained device ID after every completed or canceled worker. A failed session lookup after token exchange SHALL display that connection information remains and allow the login action to retry lookup without another browser authorization. A stale snapshot SHALL NOT restore a cleared or different connection.

#### Scenario: Session lookup is canceled after token exchange

- **WHEN** login saves credentials but session lookup is canceled
- **THEN** the UI shows connection information awaiting verification and the next login action retries lookup on the same device
