# LLVY Uploader

LLVY에 리플레이를 올리는 Go 기반 Windows x64 포터블 프로그램이다.
현재는 **서버 연동 개발 버전**이다. 파일 탐색·경기 ID 매칭, 브라우저 로그인·DPAPI 로그인 유지, 순차 전송·결과 복구와 Windows 빌드·CI 구성을 구현했다.
기본 서버 주소는 없다. 다운로드용 빌드는 GitHub Actions 변수 `LLVY_SERVER_ORIGIN`의 HTTPS origin을 exe에 고정한다. 서버 API는 로컬 구현·기본 비활성화 상태이며 **운영 배포·실제 Windows/Blob/Discord 검증과 릴리스는 남아 있다**.

## 현재 동작과 예정된 역할

리플레이 폴더 탐색, 경기 번호로 파일 찾기와 후보 선택을 구현했다.
브라우저 연결 승인, 선택한 파일의 순차 전송과 결과 표시를 서버 후보 계약에 연결했다.
예약 응답 유실은 같은 기기 연결에서 잔여 한도와 별개로 같은 키로 복구하며, 처리 전 토큰 수명과 취소 후 로그인 상태를 확인한다.
연결 해제·변경 시 이전 예약과 선택을 비우고 재선택을 안내한다. 로그인 후 상태 조회만 실패하면 기존 연결로 조회를 다시 시도한다.
인증·권한·예약 한도·리플레이 검증·DB 저장은 LLVY 서버가 담당한다.
앱은 LLVY의 `/api/desktop/v1/`와 서버가 승인한 Blob 주소에 연결하며 Worker나 QStash에 의존하지 않는다.
서버의 `DESKTOP_UPLOADER_ENABLED`는 기본 false이며 실제 연동 검증 후 활성화해야 한다.

## 계획 읽기

- [제안](openspec/changes/add-go-windows-replay-uploader/proposal.md)
- [설계](openspec/changes/add-go-windows-replay-uploader/design.md)
- [미완료 작업](openspec/changes/add-go-windows-replay-uploader/tasks.md)
- [계약 관리](contracts/desktop-v1/README.md)
- [분리 이력과 작업 이동표](docs/repository-split.md)
- [LLM과 기여자 지침](AGENTS.md)
- [업로더 위키](../llvy-wiki/wiki/uploader.md)
- [현재 사용 범위](docs/usage.md)
- [개발·Windows 빌드](docs/development.md)
- [구현·검증 기록](docs/validation/2026-10-18-server-integration.md)

OpenSpec CLI가 설치된 환경에서 다음으로 문서를 확인한다.

```bash
openspec list --json
openspec validate --all --strict
```

Go 1.27.0으로 공통 검사를 실행한다.

```bash
go test ./...
go vet ./...
go test -race ./...
```

Windows PowerShell에서 `LLVY_SERVER_ORIGIN` 환경변수를 지정한 뒤 `./scripts/build.ps1`로 `dist/llvy-uploader.exe`를 만들 수 있다.
빌드는 Go 1.27.0, `golang.org/x/sys` v0.48.0, 리소스 도구 `go-winres` v0.3.3을 고정하며 형제 저장소를 요구하지 않는다.
원격 저장소는 [kgskr/llvy-uploader](https://github.com/kgskr/llvy-uploader)이며 로컬 모듈 경로는 `llvy-uploader`를 유지한다. 후보 API 계약 사본은 원본 commit·SHA-256으로 고정했다.
문서의 형제 경로 링크는 세 코드 저장소와 위키를 같은 상위 디렉토리에 배치했을 때 동작한다.
배포 라이선스는 별도로 결정한다.

[다운로드 빌드 workflow](.github/workflows/build-windows.yml)는 수동 실행 후 검증된 exe·체크섬·출처 정보를 Actions 아티팩트에 30일 보관한다. 주소 변수가 없으면 빌드가 실패한다. 설정·실행·다운로드 순서는 [개발 안내](docs/development.md#다운로드-빌드)를 따른다.

클라우드 CI는 Linux에서 공통 테스트·race·vet·형식 검사를 먼저 실행하고 모두 성공하면 Windows 테스트·race·vet·exe 검사를 실행한다. macOS CI와 macOS 실행파일은 제공하지 않으며 Mac 공통 검사는 로컬 개발용이다.
