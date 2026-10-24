# 개발·빌드·검증

상태: 서버 연동 개발 버전. Windows CI 자동 검사는 통과했으며 공개 릴리스와 실제 Windows 10/11 사용자 PC 검증은 남아 있다.

## 독립 도구 체인

Go 1.27.0, 런타임 바인딩 `golang.org/x/sys` v0.48.0, 리소스 생성 도구 `go-winres` v0.3.3을 고정한다.
루트와 `tools/`의 `go.mod`·`go.sum`이 입력이다. 최초 의존성 다운로드에는 Go 모듈 서비스 접근이 필요하다.
형제 LLVY·Node.js·WebView2·CGo는 앱 빌드에 필요하지 않다. 원격 저장소는 [kgskr/llvy-uploader](https://github.com/kgskr/llvy-uploader)이며 모듈 경로는 `llvy-uploader`를 유지한다. 프로젝트 라이선스는 미정이다.

```bash
go mod verify
go test ./...
go vet ./...
go test -race ./...
```

race 검사는 이를 지원하는 호스트와 C 도구 체인이 필요하다. Windows 배포 빌드는 CGo를 사용하지 않는다.
Windows 전용 테스트는 DPAPI, Known Folder와 읽기 핸들의 수정·삭제 차단을 검사한다. 공통 테스트는 가상 파일과 합성 데이터를 사용한다.

## Windows PowerShell

```powershell
$env:LLVY_SERVER_ORIGIN = "https://uploader.example.invalid" # 실제 서버의 HTTPS origin으로 지정
./scripts/build.ps1 -Version 1.2.3
./scripts/verify-build.ps1 -Version 1.2.3
./dist/llvy-uploader-1.2.3-windows-x64.exe
```

주소는 경로·query·fragment·사용자 정보를 포함하지 않은 HTTPS origin이어야 한다. 누락·잘못된 값은 빌드를 실패시킨다. `-AllowUnconfigured`는 CI의 주소 없는 self-test exe에만 사용하며 이 exe는 일반 실행을 거부한다.

`build.ps1`은 amd64 GUI exe, SHA-256, Go 빌드 정보와 소스 provenance를 `dist/`에 만든다.
manifest는 asInvoker, Per Monitor V2 DPI, 긴 경로와 Common Controls v6을 선언한다.
버전은 `-Version`으로 파일명·앱 버전·PE FileVersion/ProductVersion에 함께 주입한다. 생략 시 로컬 개발 버전 `0.0.0-dev`를 사용한다. 아이콘·코드 서명은 추가하지 않았다. 생성 `.syso`는 빌드 후 제거하며 바이너리를 커밋하지 않는다.
self-test는 임시 디렉토리에서 잠금·한글 설정·DPAPI를 검사하고 최대 5초 뒤 종료한다. UI·로그인·업로드는 실행하지 않는다.

## macOS 교차 빌드

```bash
go -C tools tool go-winres make --in ../build/windows/winres.json --out ../cmd/llvy-uploader/rsrc --arch amd64
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H windowsgui -X llvy-uploader/internal/desktop.ServerOrigin=https://uploader.example.invalid' -o dist/llvy-uploader.exe ./cmd/llvy-uploader
go run ./scripts/inspect dist/llvy-uploader.exe
```

교차 빌드는 컴파일·PE 구조만 검사한다. 실제 Windows UI, DLL 로딩, DPAPI, 파일 잠금 성공을 뜻하지 않는다.
PE 검사는 명시적 import table의 DLL을 검사한다. Win32 어댑터가 실행 중 불러오는 시스템 DLL과 실제 깨끗한 사용자 환경은 별도 Windows 실행 검증 대상이다.

## CI와 릴리스 경계

[검사 workflow](../.github/workflows/check.yml)는 고정 action SHA, Go 1.27.0, `ubuntu-24.04`와 `windows-2025`를 순서대로 사용한다. Mac 검사는 로컬 개발 시 선택적으로 실행한다.
2026-10-19에 확인한 공식 안정 버전 checkout v7.0.1·setup-go v7.0.0를 commit SHA로 고정한다. 두 액션은 Node.js 24로 실행되며 앱 exe에 Node.js 의존성을 추가하지 않는다.
push·pull request·수동 실행에서 Linux 공통 테스트·race·vet·gofmt를 먼저 실행한다. Linux job이 성공해야 `needs: linux`로 연결한 Windows 전체 테스트·race·vet를 시작하며 Linux 실패·취소 시 Windows는 건너뛴다. Go 소스는 `.gitattributes`로 LF 체크아웃을 고정해 Windows에서도 gofmt 결과를 비교한다. race 검사 단계에서만 `CGO_ENABLED=1`, `CC=gcc`를 지정하며 exe 빌드는 `CGO_ENABLED=0`을 유지한다. 최초 실행에서 발견한 계약 JSON의 CRLF 체크아웃 문제는 `.gitattributes`의 LF 고정으로 수정했다. 실행별 결과와 남은 범위는 [검증 기록](validation/2026-10-18-server-integration.md#2026-10-19-windows-ci-줄바꿈-수정)을 따른다.
일반 검사는 read 권한과 `pull_request`를 사용하며 배포 exe를 만들지 않는다. 양쪽 OS에서 버전 파싱·파일명·리소스 생성·PowerShell 문법도 검사한다. 릴리스 workflow는 `release.published`에서만 Windows exe 빌드·PE 검사·self-test·내장 주소 검사를 거쳐 Release에 직접 첨부한다. 버전 Release 자체는 workflow가 생성하지 않는다.

정식 릴리스에는 서버 원본 계약의 commit·체크섬과 호환 서버 검증, Windows 10/11 일반 사용자·한국어 IME·DPI·OneDrive·실제 LoL 경로, Discord·Blob·ROFL 통합 검증이 필요하다.
고정 의존성 라이선스 묶음은 [THIRD_PARTY_NOTICES.txt](../THIRD_PARTY_NOTICES.txt)에 있다. 릴리스 게시 권한/태그 보호, 아이콘과 서명 상태는 별도로 확정한다. 프로젝트 라이선스는 기존 LLVY에서 자동 상속하지 않는다.

## 근거

- [Windows reparse tag](https://learn.microsoft.com/en-us/windows/win32/fileio/reparse-point-tags): name-surrogate 판정 근거.
- [Windows CreateFile](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew): 읽기 공유와 reparse/no-recall 플래그.
- [고정 리소스 생성 도구](https://github.com/tc-hib/go-winres/tree/v0.3.3): manifest·버전 리소스 형식.
- [현재 검증 기록](validation/2026-10-18-server-integration.md).

## 고정 계약과 개발 서버

`contracts/desktop-v1/source.json`의 원본 commit·SHA-256을 검사하고 사본 fixture를 Go DTO·전송 검증기에 대조한다. endpoint와 허용 헤더 값은 사본의 `protocol.json`을 embed한다. Blob API 버전이나 store ID를 클라이언트 요청 생성 코드에 넣지 않는다.

일반 빌드는 `LLVY_SERVER_ORIGIN`에서 주입한 HTTPS origin만 사용한다. 기본값과 실행 시 환경변수 fallback은 없다. 리다이렉트되는 별칭 대신 서버의 최종 origin을 설정해야 한다. 별도 로컬 개발 빌드는 다음처럼 명시적 loopback HTTP 주소로 만들며 제품 UI에서 주소를 변경하지 않는다.

```powershell
go build -tags development -ldflags "-H windowsgui -X llvy-uploader/internal/desktop.ServerOrigin=http://127.0.0.1:3000" -o dist/llvy-uploader-dev.exe ./cmd/llvy-uploader
```

개발 origin은 제품과 다른 잠금·저장소를 쓴다. 서버는 자체 테스트 DB와 Blob/Discord 설정이 필요하다. 운영 환경 변수를 개발 fixture로 사용하지 않는다. 실제 Blob PUT 검증 전에는 후보 전송 헤더가 서비스에서 수락된다고 가정하지 않는다.

## 다운로드 빌드

1. 저장소 Settings → Secrets and variables → Actions → Variables에 `LLVY_SERVER_ORIGIN`을 등록한다. 값은 실제 서버의 HTTPS origin이며 exe와 출처 정보에 포함된다.
2. [검증 근거](release/README.md)를 실제 검증으로 채우고 `LLVY_RELEASE_EVIDENCE_JSON` 저장소 변수에 등록한다. 근거는 정확한 앱 commit·버전·origin에 묶인다. 예시·누락은 통과하지 않는다. 정식 릴리스는 pending도 거부하며, 명시적인 테스트용 prerelease만 [채널 조건](release/README.md#테스트용-사전-릴리스)에 따라 외부 검증 pending을 허용한다. 빌드할 commit의 버전 태그로 GitHub Release를 **Publish release**한다. `v1.2.3` 또는 `uploader-v1.2.3`, `v1.2.3-rc.1`을 지원한다. draft 저장·tag/push·수동 일반 검사는 배포 빌드를 실행하지 않는다. 버전 숫자는 Windows 리소스 범위 0–65535이며 선행 0, `+build` 메타데이터와 경로/공백은 허용하지 않는다.
3. **Publish Windows executable**가 Linux 공통 검사 → Windows tests/race/vet → exe 빌드·PE·self-test·내장 주소 검사를 수행한다. `LLVY_SERVER_ORIGIN` 또는 일치하는 수용 검증 근거가 없거나 검사가 실패하면 게시하지 않는다.
4. 통과하면 해당 GitHub Release의 **Assets**에 `llvy-uploader-1.2.3-windows-x64.exe`, `SHA256SUMS.txt`, `build-info.txt`, `provenance.txt`, `THIRD_PARTY_NOTICES.txt`, `release-evidence.json`을 첨부한다. 실행 파일명에는 버전을 쓰고 source commit은 provenance에 보존한다.
5. 게시 단계만 `GH_TOKEN`을 받으며 배포 job은 `contents: write`를 사용한다. 태그 버전·실제 Git commit·Release 채널을 검증한 뒤 첨부하고, 모든 파일을 다시 내려받아 SHA-256을 대조한다. 기존 첨부는 덮어쓰지 않는다. 재실행 시 같은 파일은 재사용하고 누락 파일만 올리며 다른 바이트가 있으면 실패한다.

사용자는 [릴리스 목록](https://github.com/kgskr/llvy-uploader/releases)에서 원하는 버전의 **Assets → `.exe`**를 클릭해 다운로드한다. Docker·ORAS·GitHub CLI 설치는 필요하지 않다. 다운로드한 파일과 같은 릴리스의 `SHA256SUMS.txt`를 다음처럼 비교할 수 있다.

```powershell
Get-FileHash ./llvy-uploader-1.2.3-windows-x64.exe -Algorithm SHA256
Get-Content ./SHA256SUMS.txt
```

배포 workflow는 GHCR 패키지를 만들지 않는다. `0.1.0-rc.1`의 이전 GHCR 게시 이력은 그대로 남아 있으며 같은 검증된 exe를 Release에도 첨부했다. 기존 버전 태그를 옮기거나 파일을 교체하지 않는다. 변경된 workflow는 이 수정이 포함된 commit에서 만드는 다음 릴리스부터 적용한다.

현재 산출물은 미서명 테스트용 사전 릴리스다. Windows 10/11 일반 사용자·실제 서버 전체 업로드 검증은 별도로 남아 있다. 서버 주소를 변경하려면 새 버전으로 다시 빌드한다.

## 업로더 역할 검사

`go test ./internal/desktop -run TestSessionRoles`는 uploader/admin/owner session 허용과 viewer/unknown 거부를 확인한다. Go 전체 test/vet/race와 Windows amd64 교차 빌드를 확인했다. 새 uploader 서버 계약은 원격 commit과 사본/hash를 고정했다. 실제 Windows·외부 검증은 [호환 상태](../README.md#업로더-역할-호환)의 미완료 조건이다.
