## ADDED Requirements

### Requirement: Windows client has independent reproducible build inputs

The repository SHALL build and version the Go client separately from Next.js using pinned Go/dependency/action versions, a committed module checksum file and an independent `vX.Y.Z` or `uploader-vX.Y.Z` release tag (optionally with a SemVer prerelease suffix). Production builds SHALL target `windows/amd64` with CGo disabled, GUI subsystem and embedded version/commit metadata. Client-only changes SHALL not require a web deployment; server or contract changes SHALL run affected checks in the owning repositories. Ordinary client builds SHALL NOT require a sibling LLVY checkout or Next.js toolchain.

#### Scenario: Client-only pull request

- **WHEN** a change touches only the Go client and its release files
- **THEN** client verification runs without building or publishing a new web deployment solely for that change

### Requirement: ARM Mac checks and Windows execution evidence are distinguished

Portable logic MAY be tested locally on ARM Mac using deterministic fixtures and mock HTTP. Cloud CI SHALL first run all Linux-compatible Go tests with the race detector, vet and formatting checks on `ubuntu-24.04`. In both check and download workflows, the Windows job SHALL depend on successful completion of the Linux job and SHALL be skipped if Linux fails or is canceled. Windows SHALL then run the full test suite with native adapters, race detection and vet. Distribution builds and executable checks SHALL run only in the release-published workflow; ordinary push/PR/manual checks SHALL NOT build a distribution exe. CGo SHALL be enabled for race tests only; distributed executables SHALL remain CGo-free. Windows-specific tests and release builds SHALL run on the GitHub-hosted `windows-2025` x64 runner. Cross-compilation success SHALL not be reported as Windows runtime success. The Windows job SHALL exercise Known Folder/discovery adapters, Unicode paths, file sharing/locks, loopback callbacks and DPAPI with isolated test data, and the release job SHALL execute the built exe's bounded self-test mode.

#### Scenario: Mac cross-build passes but Windows test fails

- **WHEN** the application compiles on Mac but a Windows API test fails on the runner
- **THEN** release verification fails and no successful Windows compatibility claim is made

### Requirement: Contract and server regressions gate executable release

The release workflow SHALL require Go tests/static checks, shared API fixture compatibility, relevant Windows integration tests and successful server gates for coupled server changes. Required server checks SHALL include `npm run check`, strict OpenSpec validation and actual multi-connection PostgreSQL authorization/race tests when authority or locking changes. Synthetic fixtures and local mocks SHALL not substitute for real external-service acceptance.

#### Scenario: Old web upload is broken by a shared-service refactor

- **WHEN** the existing browser upload contract fails after desktop support is introduced
- **THEN** the coupled change cannot pass its release gate even if the Go tests succeed

### Requirement: Releases distribute one exe with traceable verification

A protected release workflow SHALL publish the verified exe, SHA-256 checksum, dependency/license information and source commit provenance as directly downloadable GitHub Release assets after a GitHub Release is published. The distribution workflow SHALL listen only to `release.published`, validate the release tag and build that tag. Executable filenames, embedded application/resource versions SHALL use the normalized release version, not the commit hash; provenance SHALL retain the full source commit. Contents write permission SHALL be limited to the release build/publish job. It SHALL download the published assets and verify all downloaded checksums before declaring publication successful. Dependency/license notices SHALL be bundled. Real-PC acceptance SHALL remain pending until actual evidence is available and SHALL block stable publication. An explicitly marked test prerelease MAY follow the bounded prerelease evidence requirement below. Application binaries SHALL not be committed to Git. The workflow SHALL inspect non-system DLL/runtime dependencies and SHALL not rely on Node.js/WebView2 or developer toolchains installed on the runner. Fork/PR jobs SHALL not receive production or signing credentials. Signing status SHALL be accurately disclosed and SmartScreen suppression SHALL not be guaranteed.

#### Scenario: Release artifact cannot run without a developer dependency

- **WHEN** dependency inspection or clean-user execution reveals a required extra application runtime or DLL
- **THEN** the artifact fails the single-exe acceptance gate until the dependency is eliminated or the design is explicitly revised

### Requirement: Real Windows and staging acceptance remain explicit completion gates

Before the change is marked implemented and archived, verification SHALL include actual Windows 10/11 ordinary-user execution with Korean input/paths, DPI, redirected Documents and real LoL replay locations, plus staging browser-to-device login, Go-to-Blob transfer, real replay ingestion, duplicate handling and revocation. Runner administrator execution, fake LoL directories and mocked Blob/Discord responses SHALL be labeled as such and SHALL not complete these gates. A missing environment SHALL leave its task open.

#### Scenario: Automated tests pass without real PC access

- **WHEN** CI succeeds but the actual Windows/LoL acceptance matrix has not been exercised
- **THEN** the automated result is recorded while the real-PC task and change remain incomplete

### Requirement: Client releases pin a server-owned protocol contract

The client SHALL consume a reviewed copy of LLVY-owned desktop-v1 schemas and fixtures with the source commit and content checksums recorded. Contract updates SHALL be explicit reviewed changes, not floating downloads during ordinary builds. Release evidence SHALL identify the tested server contract revision and preserve compatibility with supported deployed servers. Coupled server changes SHALL pass their own gates and deploy compatibly before the executable is released.

#### Scenario: Build from only the uploader checkout

- **WHEN** a developer runs client checks from a clean checkout without a sibling LLVY repository
- **THEN** local pinned fixtures suffice for client contract checks and no web build is required

#### Scenario: Server and client change together

- **WHEN** a client release needs an updated server protocol
- **THEN** the release records the compatible server revision and passing server gates before publishing the executable

#### Scenario: Windows checkout enables automatic CRLF conversion

- **WHEN** Git checks out the pinned contract JSON with `core.autocrlf=true`
- **THEN** repository attributes preserve LF bytes and the original SHA-256 verification succeeds without normalizing or changing the pinned contract

### Requirement: Download builds require an explicit server origin

Production builds SHALL have no default server address and SHALL embed an explicitly validated HTTPS origin from `LLVY_SERVER_ORIGIN` at build time. They SHALL NOT accept runtime URL overrides or follow API redirects. The pinned contract snapshot metadata SHALL NOT act as a default origin. Missing or invalid configuration SHALL fail downloadable builds before artifact publication. A separate explicit CI-only unconfigured build MAY execute offline self-tests, but SHALL reject normal startup and SHALL NOT be published.

#### Scenario: Build without a configured address

- **WHEN** the download workflow has no `LLVY_SERVER_ORIGIN` repository variable
- **THEN** it fails with a configuration error and publishes no exe

#### Scenario: Publish a configured version

- **WHEN** a release such as `v1.2.3` is published and Linux then Windows checks, origin validation, PE inspection and bounded self-test pass
- **THEN** the workflow publishes `llvy-uploader-1.2.3-windows-x64.exe`, SHA-256 and build provenance as assets of that GitHub Release, labeled unsigned with verified acceptance evidence

#### Scenario: Push code or a tag without publishing a release

- **WHEN** code or a version tag is pushed, a draft release is saved, or ordinary checks are manually dispatched
- **THEN** no distribution exe build or package publication runs

#### Scenario: Invalid release version

- **WHEN** a published tag is not a supported version, exceeds Windows resource version limits, or contains unsafe path or shell characters
- **THEN** the release workflow fails before building or publishing an executable

#### Scenario: Linux common checks fail

- **WHEN** the Linux test, race, vet or formatting gate fails or is canceled
- **THEN** the dependent Windows test/build job does not run and no downloadable artifact is published

### Requirement: Publication requires commit-bound acceptance evidence

Publication SHALL validate a reviewed evidence record with the exact client commit, version, configured origin, pinned contract source commit and checksums, compatible deployed server commit, and HTTPS evidence references for passing server checks, strict specs, PostgreSQL authorization races, Windows 10/11 ordinary-user execution, Discord login, Blob transfer, real replay/duplicate/browser regression and revocation. Missing, failed or mismatched evidence SHALL stop publication. Only an explicitly marked test prerelease MAY retain pending external checks as described below. Build and publication SHALL include the validated record and dependency/license notices in the downloaded package. Ordinary builds SHALL remain possible without publication evidence.

#### Scenario: Release has no external acceptance record

- **WHEN** a stable release is published without complete matching passing evidence
- **THEN** its workflow fails before uploading any executable to the Release

### Requirement: Test prereleases preserve incomplete acceptance honestly

A test package MAY be published only when the GitHub Release is marked prerelease, its validated version has a SemVer prerelease suffix, and its commit-bound evidence explicitly sets channel to prerelease. Server check, strict specs and PostgreSQL authorization races SHALL remain passed requirements. Each real-PC and external-service check SHALL be present with passed or pending status and an HTTPS evidence or limitation reference; failed and omitted checks SHALL be rejected. The build and publisher SHALL both validate the exact client commit, origin, version, contract hashes and server deployment. All Linux/Windows automatic checks, PE inspection, bounded self-test and release asset readback SHALL remain mandatory. Provenance, workflow summary and release notes SHALL explicitly identify the unsigned test prerelease and pending acceptance. Stable releases SHALL still require all checks passed. Prerelease publication SHALL NOT complete real-PC or external-service OpenSpec tasks.

#### Scenario: Explicit test prerelease retains external gaps

- **WHEN** a release marked prerelease has a matching suffixed version and evidence with passed server checks and documented pending Windows/Discord/Blob acceptance
- **THEN** automatic release checks may publish the test package with the pending record and unsigned prerelease label

#### Scenario: Pending evidence is presented as a stable release

- **WHEN** the release flag, version suffix or evidence channel does not match the prerelease channel, or a stable release contains a pending check
- **THEN** publication fails before uploading package files

### Requirement: Release downloads need no registry tooling

Publication SHALL use GitHub Release assets and SHALL NOT publish to GHCR or require Docker or ORAS. The publisher SHALL validate the release tag version, actual Git tag commit and prerelease channel before uploading. A rerun SHALL preserve existing assets, accept only byte-identical files and fail on conflicting content. It SHALL download all required assets and compare their SHA-256 hashes before reporting success.

#### Scenario: User downloads a Windows executable

- **WHEN** release publication succeeds
- **THEN** the Release Assets section contains the versioned exe, checksum, build information, provenance, dependency notices and evidence record downloadable through a browser

#### Scenario: Publication is retried

- **WHEN** some required assets already exist
- **THEN** identical files are reused and missing files are uploaded, while a differing existing file stops publication without replacement
