# LLVY Uploader

LLVY에 리플레이를 올리는 Go 기반 Windows x64 포터블 프로그램이다.
현재는 **서버 연동 개발 버전**이다. 파일 탐색·경기 ID 매칭, 브라우저 로그인·DPAPI 로그인 유지, 순차 전송·결과 복구와 Windows 빌드·CI 구성을 구현했다.
기본 서버 주소는 없다. 다운로드용 빌드는 GitHub Actions 변수 `LLVY_SERVER_ORIGIN`의 HTTPS origin을 exe에 고정한다. 서버는 운영 배포 후 desktop API를 활성화했다. **실제 Windows/Blob/Discord 전체 수용 검증과 앱 릴리스는 남아 있다**. [서버 배포 기록](https://github.com/kgskr/llvy/blob/2b8491476d21cd1317a363243ee6fa60003f4d4c/docs/validation/2026-10-23-uploader-deployment.md)을 따른다.

## 현재 동작과 예정된 역할

리플레이 폴더 탐색, 경기 번호로 파일 찾기와 후보 선택을 구현했다.
브라우저 연결 승인, 선택한 파일의 순차 전송과 결과 표시를 서버 후보 계약에 연결했다.
예약 응답 유실은 같은 기기 연결에서 잔여 한도와 별개로 같은 키로 복구하며, 처리 전 토큰 수명과 취소 후 로그인 상태를 확인한다.
기존 예약은 원본 파일을 열기 전에 서버에서 복구한다. 실패·만료·결과 불명확 항목은 상태 확인 후 새 예약 한도 소비에 동의하면 다시 업로드할 수 있다.
연결 해제·변경 시 이전 예약과 선택을 비우고 재선택을 안내한다. 로그인 후 상태 조회만 실패하면 기존 연결로 조회를 다시 시도한다.
refresh가 `429 rate_limited`로 거부되면 연결·보관 토큰·기존 예약을 유지하고 대기 시간을 안내한다. 안내된 시간 후 다시 시도한다. 시작 시 보관 연결 복구와 업로드 뒤 한도 조회는 이 응답 직후 갱신을 재호출하지 않으며, `LLVY 로그인`은 보관된 연결로 다시 확인한다. 응답 유실·손상은 계속 재로그인이 필요하다.
인증·권한·예약 한도·리플레이 검증·DB 저장은 LLVY 서버가 담당한다.
앱은 LLVY의 `/api/desktop/v1/`와 서버가 승인한 Blob 주소에 연결하며 Worker나 QStash에 의존하지 않는다.
서버의 `DESKTOP_UPLOADER_ENABLED`는 미설정 시 false이며 현재 운영은 true다. 서버 활성화와 실제 앱 업로드 수용 검증은 구분한다.

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
- [저장소 정합성 수정·검증](docs/validation/2026-10-19-cross-repository-review.md)
- [저장 제한·refresh 복구 검증](../llvy/docs/validation/2026-10-22-review-fixes.md)

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

Windows PowerShell에서 `LLVY_SERVER_ORIGIN` 환경변수를 지정한 뒤 `./scripts/build.ps1 -Version 1.2.3`으로 `dist/llvy-uploader-1.2.3-windows-x64.exe`를 만들 수 있다.
빌드는 Go 1.27.0, `golang.org/x/sys` v0.48.0, 리소스 도구 `go-winres` v0.3.3을 고정하며 형제 저장소를 요구하지 않는다.
원격 저장소는 [kgskr/llvy-uploader](https://github.com/kgskr/llvy-uploader)이며 로컬 모듈 경로는 `llvy-uploader`를 유지한다. 후보 API 계약 사본은 원본 commit·SHA-256으로 고정했다.
문서의 형제 경로 링크는 세 코드 저장소와 위키를 같은 상위 디렉토리에 배치했을 때 동작한다.
배포 라이선스는 별도로 결정한다. 고정 의존성과 계약 사본의 권리 고지는 [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt)에 보존한다.

[다운로드 빌드 workflow](.github/workflows/build-windows.yml)는 버전 Release가 `published`될 때만 검증된 exe·체크섬·출처를 GitHub Packages(GHCR)에 게시한다. `v1.2.3` 또는 `uploader-v1.2.3`의 파일명은 `llvy-uploader-1.2.3-windows-x64.exe`이며 패키지 tag는 `1.2.3`이다. 주소 변수가 없으면 빌드가 실패한다. 게시에는 정확한 앱·계약·서버 revision과 실제 Windows·외부 연동 검증을 담은 `LLVY_RELEASE_EVIDENCE_JSON`도 필요하며, 없으면 게시를 차단한다. [검증 근거 작성](docs/release/README.md)을 따른다. 설정·실행·다운로드 순서는 [개발 안내](docs/development.md#다운로드-빌드)를 따른다.

클라우드 CI는 Linux에서 공통 테스트·race·vet·형식 검사를 먼저 실행하고 모두 성공하면 Windows 테스트·race·vet를 실행한다. 배포 exe 빌드·PE 검사·self-test·패키지 게시는 릴리스 workflow에서만 실행한다. macOS CI와 macOS 실행파일은 제공하지 않으며 Mac 공통 검사는 로컬 개발용이다.

## 업로더 역할 호환

서버의 일반 사용자·업로더·관리자·서비스 오너 중 업로더부터 앱 연결과 업로드를 허용한다. 앱은 `uploader`, `admin`, `owner` session을 그대로 받아들이고 viewer와 알 수 없는 역할은 거부한다. 업로더 권한은 서버의 관리자 또는 오너가 부여·회수하며, 회수·관리자 승격 뒤에는 이전 기기 연결을 재사용하지 않고 다시 로그인해야 한다. 권한 관리 기능은 앱에 추가하지 않는다.

업로더 역할 계약은 원본 commit `fe32e8a93a426b0446338d24c201c5b778dab682`로 고정하고 사본·SHA-256을 갱신했다. 이 commit은 LLVY의 원격 `uploader-role-contract` ref에 게시됐다. 기존 admin/owner 호환과 새 uploader fixture를 검사한다. [서버 변경](../llvy/openspec/changes/add-uploader-role/proposal.md)의 호환 서버/앱 배포 및 실제 Windows·Discord·Blob 검증 후에 uploader 계정을 활성화한다.
