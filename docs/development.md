# 개발·빌드·검증

상태: 서버 연동 개발 버전. 공개 릴리스와 Windows 실행 검증은 아직 없다.

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
./scripts/build.ps1
go run ./scripts/inspect ./dist/llvy-uploader.exe
./dist/llvy-uploader.exe --self-test
./dist/llvy-uploader.exe
```

`build.ps1`은 amd64 GUI exe, SHA-256, Go 빌드 정보와 소스 provenance를 `dist/`에 만든다.
manifest는 asInvoker, Per Monitor V2 DPI, 긴 경로와 Common Controls v6을 선언한다.
현재 버전은 `0.0.0-dev`이며 아이콘·코드 서명은 추가하지 않았다. 생성 `.syso`는 빌드 후 제거하며 바이너리를 커밋하지 않는다.
self-test는 임시 디렉토리에서 잠금·한글 설정·DPAPI를 검사하고 최대 5초 뒤 종료한다. UI·로그인·업로드는 실행하지 않는다.

## macOS 교차 빌드

```bash
go -C tools tool go-winres make --in ../build/windows/winres.json --out ../cmd/llvy-uploader/rsrc --arch amd64
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H windowsgui' -o dist/llvy-uploader.exe ./cmd/llvy-uploader
go run ./scripts/inspect dist/llvy-uploader.exe
```

교차 빌드는 컴파일·PE 구조만 검사한다. 실제 Windows UI, DLL 로딩, DPAPI, 파일 잠금 성공을 뜻하지 않는다.
PE 검사는 명시적 import table의 DLL을 검사한다. Win32 어댑터가 실행 중 불러오는 시스템 DLL과 실제 깨끗한 사용자 환경은 별도 Windows 실행 검증 대상이다.

## CI와 릴리스 경계

[검사 workflow](../.github/workflows/check.yml)는 고정 action SHA, Go 1.27.0, macOS와 `windows-2025`를 사용한다.
push·pull request·수동 실행에서 Windows 테스트·vet·빌드·PE 검사·self-test를 실행하도록 구성했지만 아직 실행한 CI 결과는 없다.
기본 read 권한과 일반 `pull_request`만 사용하며 비밀값을 요구하지 않는다. 현재 artifact 업로드·GitHub Release 발행은 하지 않는다.

정식 릴리스에는 서버 원본 계약의 commit·체크섬과 호환 서버 검증, Windows 10/11 일반 사용자·한국어 IME·DPI·OneDrive·실제 LoL 경로, Discord·Blob·ROFL 통합 검증이 필요하다.
그 후 보호된 `uploader-v*` workflow, 아이콘, 의존성 라이선스 묶음, artifact 보존과 서명 상태를 확정한다. 프로젝트 라이선스는 기존 LLVY에서 자동 상속하지 않는다.

## 근거

- [Windows reparse tag](https://learn.microsoft.com/en-us/windows/win32/fileio/reparse-point-tags): name-surrogate 판정 근거.
- [Windows CreateFile](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew): 읽기 공유와 reparse/no-recall 플래그.
- [고정 리소스 생성 도구](https://github.com/tc-hib/go-winres/tree/v0.3.3): manifest·버전 리소스 형식.
- [현재 검증 기록](validation/2026-10-18-server-integration.md).

## 고정 계약과 개발 서버

`contracts/desktop-v1/source.json`의 원본 commit·SHA-256을 검사하고 사본 fixture를 Go DTO·전송 검증기에 대조한다. endpoint와 허용 헤더 값은 사본의 `protocol.json`을 embed한다. Blob API 버전이나 store ID를 클라이언트 요청 생성 코드에 넣지 않는다.

일반 빌드는 `https://llvy.my`만 사용한다. 별도 로컬 개발 빌드는 다음처럼 만들며 제품 UI에서 주소를 변경하지 않는다.

```powershell
go build -tags development -ldflags "-H windowsgui -X llvy-uploader/internal/desktop.DevelopmentOrigin=http://127.0.0.1:3000" -o dist/llvy-uploader-dev.exe ./cmd/llvy-uploader
```

개발 origin은 제품과 다른 잠금·저장소를 쓴다. 서버는 자체 테스트 DB와 Blob/Discord 설정이 필요하다. 운영 환경 변수를 개발 fixture로 사용하지 않는다. 실제 Blob PUT 검증 전에는 후보 전송 헤더가 서비스에서 수락된다고 가정하지 않는다.
