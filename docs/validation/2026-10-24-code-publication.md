# 업로더 코드 게시 검증

업로더 역할 파서와 고정 계약, 예약 복구·refresh 제한 응답 처리, 제한 크기 진단 로그, 릴리스 근거 검사를 포함한 변경을 검증했다. 서버 원본 계약 `fe32e8a93a426b0446338d24c201c5b778dab682`는 원격에 게시됐으며 앱 사본의 세 파일 해시와 일치한다. 서버는 `2b8491476d21cd1317a363243ee6fa60003f4d4c`로 운영 배포 및 desktop 기능 활성화를 완료했다.

## 로컬 검증

Go 1.27.0에서 `go mod verify`, `go test -race -timeout 120s ./...`, `go vet ./...`, Windows amd64 교차 build/vet, `openspec validate --all --strict`, `git diff --check`를 통과했다. Windows 실행·Discord·Blob 수용 검증을 의미하지 않는다.

실제 Windows 일반 사용자·외부 서비스 검증은 미완료이며 기존 OpenSpec을 열린 상태로 유지한다. 정식 패키지 게시에는 `docs/release/README.md`의 정확한 commit에 대응하는 실환경 검증 기록이 필요하다. 위키의 앱 게시 상태 갱신은 패키지 게시 결과 확정 후 별도로 진행한다.

## 테스트용 게시 준비

앱 코드 `81b2a4a3be58017a6039da5b154e1aad5d69c29c`의 [Linux/Windows CI](https://github.com/kgskr/llvy-uploader/actions/runs/40063249264)가 통과했다. 이후 게시 채널 변경은 Go 회귀 테스트로 정식 채널의 pending 거부, 사전 채널의 서버 검사 필수 조건, 버전/channel/commit/계약 불일치 및 누락·failed·잘못된 근거 URL 거부를 검증한다. 최종 게시 workflow 자체에서 같은 commit의 전체 Linux/Windows 검사와 exe self-test를 다시 수행한다.

버전 `0.1.0-rc.1`의 실환경 검증 상태는 Windows 10/11·Discord·Blob·실제 리플레이/웹 회귀·회수 모두 pending이다. 서버 check·OpenSpec·PostgreSQL 경쟁 검증은 [서버 배포 기록](https://github.com/kgskr/llvy/blob/2b8491476d21cd1317a363243ee6fa60003f4d4c/docs/validation/2026-10-23-uploader-deployment.md)을 따른다. 테스트용 게시가 이 pending 항목을 완료하거나 정식 지원을 확정하지 않는다.
