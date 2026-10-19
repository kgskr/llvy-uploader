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
일반 검사는 read 권한과 `pull_request`를 사용하며 배포 exe를 만들지 않는다. 양쪽 OS에서 버전 파싱·파일명·리소스 생성·PowerShell 문법도 검사한다. 릴리스 workflow는 `release.published`에서만 Windows exe 빌드·PE 검사·self-test·내장 주소 검사를 거쳐 GHCR에 게시한다. 버전 Release 자체는 workflow가 생성하지 않는다.

정식 릴리스에는 서버 원본 계약의 commit·체크섬과 호환 서버 검증, Windows 10/11 일반 사용자·한국어 IME·DPI·OneDrive·실제 LoL 경로, Discord·Blob·ROFL 통합 검증이 필요하다.
릴리스 게시 권한/태그 보호, 아이콘, 의존성 라이선스 묶음과 서명 상태는 별도로 확정한다. 프로젝트 라이선스는 기존 LLVY에서 자동 상속하지 않는다.

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
2. 빌드할 commit의 버전 태그로 GitHub Release를 **Publish release**한다. `v1.2.3` 또는 `uploader-v1.2.3`, `v1.2.3-rc.1`을 지원한다. draft 저장·tag/push·수동 일반 검사는 배포 빌드를 실행하지 않는다. 버전 숫자는 Windows 리소스 범위 0–65535이며 선행 0, `+build` 메타데이터와 경로/공백은 허용하지 않는다.
3. **Publish Windows package**가 Linux 공통 검사 → Windows tests/race/vet → exe 빌드·PE·self-test·내장 주소 검사를 수행한다. `LLVY_SERVER_ORIGIN`이 없거나 검사가 실패하면 게시하지 않는다.
4. 통과하면 GitHub Packages의 `ghcr.io/<owner>/llvy-uploader:1.2.3`에 OCI 파일 패키지를 저장한다. 파일은 `llvy-uploader-1.2.3-windows-x64.exe`, `SHA256SUMS.txt`, `build-info.txt`, `provenance.txt`다. 해시는 검증/출처 정보에 보존하고 파일명에는 버전을 쓴다.
5. 게시 단계는 ORAS v1.3.5를 사용하며 `GITHUB_TOKEN`과 해당 job의 `packages: write`로 인증한다. 저장소 source annotation을 포함하고 게시한 digest를 다시 다운로드해 모든 파일의 SHA-256을 대조한다. 실행 요약에서 package reference와 검증된 digest를 확인한다.

[ORAS](https://oras.land/docs/installation/)를 설치한 뒤 다음처럼 내려받는다. `<owner>`는 해당 저장소 소유자로 바꾼다.

```powershell
oras pull ghcr.io/<owner>/llvy-uploader:1.2.3 --output ./download
Get-FileHash ./download/llvy-uploader-1.2.3-windows-x64.exe -Algorithm SHA256
Get-Content ./download/SHA256SUMS.txt
Get-Content ./download/provenance.txt
```

최초 GHCR 패키지는 GitHub의 기본 공개 범위를 따른다. private 패키지 다운로드는 `read:packages` 권한으로 ORAS 로그인이 필요하며 공개 다운로드를 원하면 패키지 설정에서 visibility를 public으로 변경해야 한다. workflow는 계정/패키지 공개 범위를 바꾸지 않는다. Actions의 30일 임시 아티팩트나 Release 첨부 파일 대신 Packages에 저장하며 삭제 전까지 버전별로 관리한다. 같은 버전을 재게시하면 태그가 갱신될 수 있으므로 고정된 다운로드에는 실행 요약의 digest를 사용한다.

현재 산출물은 미서명 개발 빌드다. 실제 버전 Release를 발행한 GHCR 게시 실행과 Windows 10/11 일반 사용자·실제 서버 업로드 검증은 별도로 확인한다. 서버 주소를 변경하려면 새 버전으로 다시 빌드한다.

- [ORAS의 GHCR 파일 게시](https://oras.land/docs/1.2/how_to_guides/remote_registries/)
- [GitHub Container registry 인증·저장소 연결·공개 범위](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
