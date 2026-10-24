# 2026-10-19 저장소 정합성 검증

대상: 업로더, LLVY 서버, 문서 위키의 로컬 수정본. 실제 Windows 사용자 PC·Blob·Discord·운영 DB/배포·버전 릴리스 검증은 포함하지 않는다.

## 수정과 근거

- 완료/중복 경로의 변경된 파일은 새 후보로 표시하고 이전 경기 링크를 제공하지 않는다. 명시적 업로드에서 새 예약을 생성하며 미확정 예약은 원래 키로 복구한다. 메타데이터 비교 범위는 사용 안내에 명시했다.
- 업로드 전 account/global reset 시각과 429 Retry-After를 표시한다. 실패 재시도도 같은 연결의 최신 한도 snapshot을 조회한다.
- origin별 진단 로그를 2×256 KiB로 제한한다. 이벤트·결과·오류 분류·HTTP 상태 외 자유 텍스트는 기록하지 않는다. 삭제·종료·회전은 잠금 아래 처리한다. 실제 네트워크 sentinel 분류와 비밀 제거 회귀를 검사했다.
- 릴리스 게시에 정확한 앱·계약·호환 서버 revision과 9개 수용 검증 근거를 요구한다. 누락·불일치·pending·인증 query/fragment를 포함한 근거 URL은 거부한다. 실제 검증 근거는 아직 없으며 예시는 통과하지 않는다.
- 고정 Go·런타임·빌드 도구·계약 사본의 라이선스 고지를 패키지에 포함한다. 자체 라이선스 결정과 외부 고지는 별개다.
- 서버의 전체 기기 회수 함수에 `npm run desktop:revoke-all` 운영 진입점을 추가했다. 기본 동작은 무접속 계획 출력이며 apply는 명시적 설정·대상과 전체 배포 차단/drain 확인을 요구한다. 운영 회수는 실행하지 않았다.
- 서버와 앱의 origin·`protocol` 필드·구현 상태 안내를 맞췄다. 완료된 서버 OpenSpec 변경 네 개는 기존 정식 명세 대조 후 아카이브했으며 누락된 경기 아이콘 정식 명세를 반영했다.
- 고정 계약 commit `f6c218f8d9b0476623b73dc75579722c8872d479`은 서버의 `desktop-v1-contract` 원격 브랜치에서 GitHub commit 조회와 SHA가 일치함을 확인했다. 계약 파일 전용 commit이며 서버 기능의 운영 배포를 뜻하지 않는다.

## 실행한 검사

- Go 1.27.0: 전체 `go test -race ./...`, `go vet ./...`, Windows amd64 교차 build/vet 통과. 독립 검토의 후속 수정은 해당 회귀/race 검사를 다시 통과했다.
- PowerShell 버전·파일명·리소스·문법 검사 통과. `build.ps1 -Version 0.0.0-review` 로컬 교차 빌드와 PE 검사 통과: 11,590,656 bytes, 정적 import는 `kernel32.dll`, 고정 `golang.org/x/sys v0.48.0`. 실제 Windows의 동적 DLL 로드/UI 실행 결과가 아니다.
- 서버 Node 24.21.0, `VITEST_MAX_WORKERS=2 npm run check` 통과: 56 test files / 893 tests 통과, 4 files / 81 tests skipped. lint·typecheck·Next build·format 통과. skip된 실제 파일·PostgreSQL·외부 서비스 검증을 완료로 취급하지 않는다.
- 운영 CLI 21개와 기존 desktop 통합 19개 scoped 검사는 통과했다. 기본 무접속 CLI 실행을 확인했으며 운영 DB apply는 수행하지 않았다.
- OpenSpec strict: 업로더 1개, LLVY 27개 통과. 코드·문서의 독립 수정 후 재검토에서 미해결 지적은 없다.

## 남은 검증

실제 Windows 10/11 사용자 환경, Discord 로그인, public/private Blob PUT, 실제 리플레이·중복·기존 웹 회귀, 실제 PostgreSQL 동시성/마이그레이션, 운영 배포 및 GHCR 릴리스는 활성 작업 목록의 미완료 조건으로 유지한다. 서버/앱 기능 변경은 로컬 수정 상태다. 위키 원격 저장소와 업로더 자체 라이선스는 별도 결정 사항이다.
