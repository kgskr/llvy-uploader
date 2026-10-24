# 업로더 코드 게시 검증

업로더 역할 파서와 고정 계약, 예약 복구·refresh 제한 응답 처리, 제한 크기 진단 로그, 릴리스 근거 검사를 포함한 변경을 검증했다. 서버 원본 계약 `fe32e8a93a426b0446338d24c201c5b778dab682`는 원격에 게시됐으며 앱 사본의 세 파일 해시와 일치한다. 서버는 `2b8491476d21cd1317a363243ee6fa60003f4d4c`로 운영 배포 및 desktop 기능 활성화를 완료했다.

## 로컬 검증

Go 1.27.0에서 `go mod verify`, `go test -race -timeout 120s ./...`, `go vet ./...`, Windows amd64 교차 build/vet, `openspec validate --all --strict`, `git diff --check`를 통과했다. Windows 실행·Discord·Blob 수용 검증을 의미하지 않는다.

실제 Windows 일반 사용자·외부 서비스 검증은 미완료이며 기존 OpenSpec을 열린 상태로 유지한다. 정식 패키지 게시에는 `docs/release/README.md`의 정확한 commit에 대응하는 실환경 검증 기록이 필요하다. 위키의 앱 게시 상태 갱신은 패키지 게시 결과 확정 후 별도로 진행한다.
