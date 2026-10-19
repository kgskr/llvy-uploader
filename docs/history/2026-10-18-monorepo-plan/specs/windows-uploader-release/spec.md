## ADDED Requirements

### Requirement: Windows client has independent reproducible build inputs

The repository SHALL build and version the Go client separately from Next.js using pinned Go/dependency/action versions, a committed module checksum file and an independent `uploader-vX.Y.Z` release tag. Production builds SHALL target `windows/amd64` with CGo disabled, GUI subsystem and embedded version/commit metadata. Client-only changes SHALL not require a web deployment; server or shared contract changes SHALL run all affected checks.

#### Scenario: Client-only pull request

- **WHEN** a change touches only the Go client and its release files
- **THEN** client verification runs without building or publishing a new web deployment solely for that change

### Requirement: ARM Mac checks and Windows execution evidence are distinguished

Portable logic SHALL be testable on ARM Mac using deterministic fixtures and mock HTTP. Windows-specific tests and release builds SHALL run on the GitHub-hosted `windows-2025` x64 runner. Cross-compilation success SHALL not be reported as Windows runtime success. The Windows job SHALL exercise Known Folder/discovery adapters, Unicode paths, file sharing/locks, loopback callbacks and DPAPI with isolated test data, and execute the built exe's bounded self-test mode.

#### Scenario: Mac cross-build passes but Windows test fails

- **WHEN** the application compiles on Mac but a Windows API test fails on the runner
- **THEN** release verification fails and no successful Windows compatibility claim is made

### Requirement: Contract and server regressions gate executable release

The release workflow SHALL require Go tests/static checks, shared API fixture compatibility, relevant Windows integration tests and successful server gates for coupled server changes. Required server checks SHALL include `npm run check`, strict OpenSpec validation and actual multi-connection PostgreSQL authorization/race tests when authority or locking changes. Synthetic fixtures and local mocks SHALL not substitute for real external-service acceptance.

#### Scenario: Old web upload is broken by a shared-service refactor

- **WHEN** the existing browser upload contract fails after desktop support is introduced
- **THEN** the coupled change cannot pass its release gate even if the Go tests succeed

### Requirement: Releases distribute one exe with traceable verification

A protected release workflow SHALL publish the verified exe, SHA-256 checksum, dependency/license information and source commit provenance through GitHub Releases. Application binaries SHALL not be committed to Git. The workflow SHALL inspect non-system DLL/runtime dependencies and SHALL not rely on Node.js/WebView2 or developer toolchains installed on the runner. Fork/PR jobs SHALL not receive production or signing credentials. Signing status SHALL be accurately disclosed and SmartScreen suppression SHALL not be guaranteed.

#### Scenario: Release artifact cannot run without a developer dependency

- **WHEN** dependency inspection or clean-user execution reveals a required extra application runtime or DLL
- **THEN** the artifact fails the single-exe acceptance gate until the dependency is eliminated or the design is explicitly revised

### Requirement: Real Windows and staging acceptance remain explicit completion gates

Before the change is marked implemented and archived, verification SHALL include actual Windows 10/11 ordinary-user execution with Korean input/paths, DPI, redirected Documents and real LoL replay locations, plus staging browser-to-device login, Go-to-Blob transfer, real replay ingestion, duplicate handling and revocation. Runner administrator execution, fake LoL directories and mocked Blob/Discord responses SHALL be labeled as such and SHALL not complete these gates. A missing environment SHALL leave its task open.

#### Scenario: Automated tests pass without real PC access

- **WHEN** CI succeeds but the actual Windows/LoL acceptance matrix has not been exercised
- **THEN** the automated result is recorded while the real-PC task and change remain incomplete
