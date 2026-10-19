## ADDED Requirements

### Requirement: Portable Go application uses a responsive native Windows interface

The client SHALL be an independent Go module under `clients/windows-uploader/` and run as one Windows x64 exe without an installer, elevation, Node.js, WebView2 or a separately installed application runtime. It SHALL use native Windows controls with Korean labels and keep file and network operations off the UI thread. Windows-specific code SHALL be isolated from portable matching, queue and API logic.

Only one instance per Windows user and server origin SHALL access shared configuration or credentials. Before reading them, the process SHALL acquire an exclusive lock-file handle in user-local storage and hold it for its lifetime, across exe paths and versions. A second instance SHALL report that the app is already running and exit without reading credentials. Crash recovery SHALL depend on OS handle release rather than whether the lock file still exists.

#### Scenario: Launch as an ordinary Windows user

- **WHEN** a user launches the distributed exe from a writable or read-only directory
- **THEN** the application opens without installation or elevation, uses user-local storage for configuration, and remains responsive while discovering files or uploading

#### Scenario: Keyboard and Korean input

- **WHEN** a user navigates with a keyboard and enters Korean paths or text at supported display scaling
- **THEN** controls expose meaningful names, input remains usable, and essential controls and results remain visible

#### Scenario: A previous process crashed while holding the instance lock

- **WHEN** the user launches the app after the previous process exited abnormally and left a lock file
- **THEN** the new process can acquire the released handle lock and start normally without deleting an active process's lock

### Requirement: Startup discovers replay locations independently of game installation

The client SHALL first reuse a valid saved replay folder, then examine validated replay-path settings associated with known LoL installations and the Windows Documents Known Folder's `League of Legends/Replays` candidate. Installation candidates SHALL be bounded to known paths and verified installation metadata. An installed game directory alone SHALL NOT prove that it is the replay directory. Discovery SHALL NOT recursively scan entire drives, modify game settings or require the game process to run.

#### Scenario: Documents has been redirected

- **WHEN** Windows Documents resolves to OneDrive or a different drive
- **THEN** the client uses the Known Folder result rather than constructing a path from the username

#### Scenario: Game exists but replay path differs

- **WHEN** a known installation is found and a validated setting identifies another replay directory
- **THEN** that replay directory is considered instead of automatically selecting the installation root

#### Scenario: Multiple candidates

- **WHEN** multiple usable replay locations remain after honoring a saved user choice
- **THEN** the client displays the candidates and their discovery sources and requires a selection

### Requirement: Missing or inaccessible folders have a manual recovery path

The client SHALL use `리플레이 폴더를 찾지 못했습니다. LoL 클라이언트의 리플레이 저장 위치를 확인한 뒤 폴더를 선택하세요.` when automatic discovery fails. It SHALL provide `폴더 선택` and `다시 찾기`, validate folder accessibility, remember a confirmed selection, and accept a readable empty folder. Canceling the picker SHALL leave the application open.

#### Scenario: No installed game or saved replay path

- **WHEN** discovery finds no usable folder, including on a PC containing only copied replay files
- **THEN** the message and native folder picker are available and a user-selected replay folder works without a LoL installation

#### Scenario: Saved folder is unavailable

- **WHEN** a previously selected folder was removed or access is denied
- **THEN** the client reports the condition and offers discovery or manual selection without silently scanning unrelated locations

### Requirement: Match IDs are validated and matched as complete filename tokens

The client SHALL accept one or multiple match IDs separated by whitespace, commas or semicolons. It SHALL accept 1–20 digit IDs and optional 2–8 character alphanumeric platform prefixes separated by `-` or `_`, normalize platform case and leading numeric zeroes, deduplicate inputs, and cap input at 100 unique IDs. IDs SHALL remain exact strings. Riot IDs containing `#` SHALL receive a match-number explanation. A shorter ID SHALL NOT match a longer filename number by substring.

#### Scenario: Multiple pasted match IDs

- **WHEN** the input contains `KR-1234567890`, `1234567891` and a repeated ID across separators
- **THEN** the client searches once per normalized identity and displays validation failures separately

#### Scenario: Partial number collision

- **WHEN** the requested ID is `123` and a file is named `KR-1234.rofl`
- **THEN** that file is not selected for the requested ID

#### Scenario: Platform ambiguity

- **WHEN** a numeric-only ID matches multiple platform files or copies
- **THEN** the user must choose one candidate; a platform-qualified input excludes a conflicting platform

### Requirement: Search previews bounded regular files without changing originals

The client SHALL search only direct regular `.rofl` children of the selected folder, case-insensitively for the extension, and SHALL NOT follow symlinks, junctions or other name-surrogate reparse points. It SHALL NOT exclude Cloud Files merely because they have a non-name-surrogate reparse tag. Fully locally available cloud files SHALL be supported. Online-only or partially downloaded files SHALL display `로컬 다운로드 필요` with instructions to keep them on the device in Explorer and search again. Discovery SHALL inspect metadata without triggering hydration. Provider/offline/read-state errors SHALL be distinguished from a missing file and SHALL stop that item before reservation. It SHALL display ID, filename, size, modification time and match/error status before upload. No match and multiple matches SHALL be distinct. A file selected by multiple inputs SHALL enter the queue only once. Empty, unreadable or larger-than-64-MiB files SHALL not be uploaded.

#### Scenario: Preview includes missing and ambiguous IDs

- **WHEN** some requested IDs have no file and others have multiple candidates
- **THEN** those rows show `파일 없음` or `파일 선택 필요` and only resolved, selected valid files can enter the queue

#### Scenario: Replay changes after discovery

- **WHEN** a selected file changes or cannot be read stably when hashing or opening for upload
- **THEN** the client reports a file-specific error and requires a fresh selection rather than transmitting different bytes under the previous metadata

#### Scenario: OneDrive contains local and online-only replays

- **WHEN** a redirected Documents replay folder contains a fully downloaded cloud file and an online-only matching file
- **THEN** the downloaded file can be selected normally, the online-only row requests local download rather than reporting a missing file, and failed local availability consumes no upload reservation

### Requirement: Explicit sequential upload has observable per-file outcomes

The client SHALL begin reservations only after the user confirms `업로드`, stream one file at a time, and display waiting, reserving, transferring, processing, completed, duplicate, failed, canceled and outcome-unknown states. File errors SHALL not erase previous results. Authentication/authority failures and exhausted shared budgets SHALL pause the queue. Missing files SHALL not trigger downloads from Riot.

Before upload, the UI SHALL display selected file count beside the minimum account/global remaining reservation allowance, its observation time and available reset times. It SHALL describe the existing 24-hour counter window rather than imply a midnight reset. Selection beyond the snapshot allowance SHALL prompt a warning and an opportunity to revise selection. The UI SHALL explain that a created reservation consumes allowance even for a subsequent duplicate or failed upload, and the snapshot SHALL NOT guarantee capacity against concurrent clients.

#### Scenario: One replay fails in a batch

- **WHEN** a replay fails validation or parsing and subsequent authorized files remain
- **THEN** the failed result remains visible and the user can continue with other files or retry selected failures under the API retry rules

#### Scenario: User cancels during processing

- **WHEN** cancellation occurs after the server may already have committed a game
- **THEN** the client stops further work and queries the result without claiming rollback or deleting a stored game

#### Scenario: Server reports an existing replay

- **WHEN** content-hash deduplication returns an existing game
- **THEN** the client displays `이미 등록됨` and the canonical game link without treating it as an upload failure

#### Scenario: Selected batch exceeds the remaining allowance

- **WHEN** the user selects eleven files and the latest account/global snapshot permits ten reservations
- **THEN** the UI shows both counts and the partial-batch warning before starting, and explains that duplicates also consume a created reservation; any later 429 pauses remaining work and refreshes the displayed allowance

### Requirement: Local state is explicit and removable

The client SHALL store configuration and bounded redacted logs in origin-separated storage under `%LOCALAPPDATA%/LLVY/Uploader/`, provide local-data removal, and preserve original replay files. Removal SHALL NOT delete the held single-instance lock. Login persistence SHALL be opt-in and follow the desktop authorization specification. The initial release SHALL NOT automatically restore or run a queue after restart.

#### Scenario: Portable exe is moved

- **WHEN** the exe is moved to another directory on the same machine
- **THEN** its user-local configuration remains available without writing next to the executable
