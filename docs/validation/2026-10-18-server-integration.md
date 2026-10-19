# 서버 연동 개발 구현·검증

확인 기간: 2026-10-18~19. 상태: 로컬 구현·자동 검사. 실제 Windows 사용자 PC·Blob/Discord 스테이징·운영 배포 검증은 미수행이다. Windows CI 자동 실행 결과는 아래 별도 항목에 기록한다.

## 구현

- 제품 origin `https://llvy.my`, 사용자/origin별 단일 인스턴스 잠금 후 자격 증명 접근.
- literal loopback 선점, state·PKCE S256, 제한된 callback, 기존 브라우저 승인과 취소/timeout 정리.
- access 메모리 관리, 직렬화 refresh, 갱신 응답 유실 시 재로그인, 선택한 로그인 유지의 DPAPI refresh 저장과 로컬 해제.
- 단일 파일 순차 예약·스트리밍 PUT·처리·상태 조회, 안정된 파일 핸들 및 SHA-256, 같은 키 재시도와 PUT 무응답 확인, 처리 결과 복구, 한도/권한 오류 시 큐 정지.
- Win32 상태/진행률·실패 행 재시도·완료 경기 링크, 취소와 원본 보존. 실제 컨트롤 배치·IME·DPI 접근성은 Windows 검증 대상이다.
- 서버 소유 후보 계약 `f6c218f8d9b0476623b73dc75579722c8872d479` 고정 사본. 계약 commit은 서버 기능 배포 commit이 아니다.

## 자동 검사 범위

Go 1.27.0 macOS ARM64에서 공통 테스트·vet·race를 실행한다. HTTP 테스트는 합성 localhost 서버를 사용하며 redirect·응답 크기·비밀 비노출·동시 refresh·갱신 유실·callback state/중복·취소·헤더 allowlist·PUT/처리 무응답·같은 예약 키·429·고정 계약 체크섬을 검사한다.

Windows amd64 vet·교차 빌드와 PE 검사를 수행한다. manifest의 GUI subsystem·리소스, Go 빌드 정보와 명시적 import table을 검사한다. 개발 exe는 약 11 MiB이며 최종 릴리스 크기는 아니다. 바이너리는 저장소에 넣지 않는다. Windows 전용 테스트·self-test·실제 DLL 로딩·DPAPI·파일 잠금은 교차 빌드로 실행되지 않는다.

서버의 PGlite 및 실제 localhost PostgreSQL 검증은 [서버 기록](../../../llvy/docs/validation/2026-10-18-desktop-uploader.md)을 참고한다. 해당 결과는 Windows/Blob 실행 증거가 아니다. 서버·위키가 없는 별도 디렉토리에서 앱 tests/race와 Windows 교차 빌드도 통과했다.

## 남은 완료 조건

- 실제 public/private Blob PUT과 access/경로/크기/만료/overwrite 변조 거부. 현재 descriptor는 SDK 소스·타입에 맞춘 후보다.
- staging의 Discord 승인부터 실제 ROFL 저장·duplicate·회수·공유 한도·cleanup·owner 변경/rollback까지 전체 흐름.
- Windows 10/11 일반 사용자, IME·DPI·OneDrive·실제 LoL 설정/파일명, 서로 다른 exe 경로·버전의 잠금 및 crash 복구, 반복 임시 연결·로그인 유지 저장 실패.
- Windows CI의 일반 자동 검사는 아래 기록대로 통과했다. 아이콘·라이선스 묶음·서명/SmartScreen, 보호된 release workflow와 라이선스 결정은 남아 있다.

두 기능 change를 열린 상태로 유지한다. 로컬 구현과 검사 통과를 운영 활성화나 릴리스로 표시하지 않는다. [이전 미리보기 기록](2026-10-18-local-preview.md)은 연동 전 상태의 기록이다.

## 2026-10-19 복구 경계 회귀 검사

처리 전 access 유효기간 2분 확보와 기기 만료 임박 시 요청 차단, 예약 응답 3회 유실 후 잔여 0에서 동일 key·metadata 복구, 신규 항목 초과와 기존 예약 복구의 혼합 선택, 미예약 파일 재검색, 취소된 로그아웃·기록 삭제 상태 보존을 회귀 테스트로 추가했다. UI 결과 처리는 공통 Go 함수에서 검사하며 실제 Win32 화면 실행을 대신하지 않는다.

Go 전체 tests·race·vet, Windows amd64 vet·GUI 교차 빌드, 고정 계약 체크섬, 문서 링크·OpenSpec strict 검사를 수행했다. 서버의 활성 기기 우선 목록과 요청 생성 시 만료 정리는 [서버 기록](../../../llvy/docs/validation/2026-10-18-desktop-uploader.md#2026-10-19-인증-목록과-만료-요청-회귀-검사)을 따른다. 계약 JSON과 원본 revision은 변경하지 않았다.

## 2026-10-19 기기 연결과 탐색 취소 회귀 검사

예약에 생성 당시 기기 세션을 보관하고 다른 연결의 기존 예약·응답 유실 예약을 요청 전에 차단한다. 연결 해제·변경 시 UI의 이전 작업과 선택을 비우며 새 예약이 한도를 사용함을 안내한다. 같은 기기의 access 갱신은 예약 복구를 유지한다.

로그인 토큰 교환 후 상태 조회 실패·취소를 합성 HTTP 서버로 재현하고, 다음 로그인 동작이 브라우저 승인 없이 기존 연결의 상태 조회만 재시도함을 검사했다. 완료된 로그아웃 뒤 오래된 session 응답을 화면에 적용하지 않는 경계도 검사했다. 취소된 자동 폴더 탐색·파일 검색은 기존 선택을 보존하고 폴더 선택창을 열지 않으며, 완료된 직접 폴더 선택·로그아웃·기록 삭제는 유지한다. 실제 Win32 화면과 외부 인증 실행은 미검증이다.

Go 전체 tests·race, macOS/Windows vet와 Windows amd64 GUI 교차 빌드를 통과했다. 기기 소유권·UI 상태·통합 회귀를 나눠 독립 검토한 결과 추가 수정이 필요한 결함은 발견되지 않았다. OpenSpec strict와 현재 문서 60개의 상대 링크·앵커 262개, Markdown 형식·공백 검사를 통과했다. 서버 계약과 기능 플래그는 변경하지 않았고 실제 Windows·Blob·Discord 검증 및 운영 배포는 남아 있다.

## 2026-10-19 Windows CI 줄바꿈 수정

[최초 CI](https://github.com/kgskr/llvy-uploader/actions/runs/39184546181)에서 macOS 검사는 통과했으나 Windows는 `TestPinnedServerContract`의 체크섬 검사에 실패했다. Windows의 자동 CRLF 체크아웃 설정으로 계약 JSON 세 파일의 바이트가 바뀌는 현상을 별도 체크아웃에서 재현했다. `.gitattributes`로 해당 JSON의 LF를 고정한 뒤 같은 설정에서 원본 체크섬 세 개가 모두 일치했다. 계약 JSON·원본 revision·체크섬 및 검증 함수는 변경하지 않았다.

수정 커밋 `706afa925555f81ea1d3997b8bd3f6b064baf84a`의 [CI 실행](https://github.com/kgskr/llvy-uploader/actions/runs/39185539718)에서 macOS tests/race/vet·형식 검사와 Windows 전체 테스트·vet·exe 빌드·PE 검사·bounded self-test가 모두 통과했다. Windows 작업은 1분 49초, macOS 작업은 32초였다. GitHub-hosted Windows Server runner의 자동 검증이며 실제 Windows 10/11 일반 사용자 PC 검증을 대신하지 않는다. 일반 사용자 PC·IME·DPI·실제 Blob·Discord 검증은 별도 완료 조건이다.

## 빌드 origin과 다운로드 구성

고정 제품 주소 기본값을 제거하고 `LLVY_SERVER_ORIGIN`으로 검증한 HTTPS origin을 linker에 전달한다. 누락·HTTP·경로·query·fragment·사용자 정보·범위 밖 port와 linker 구문을 섞은 입력을 거부하는 테스트를 추가했다. 기존 pinned JSON의 체크섬을 유지하며 원본의 `productOrigin` 메타데이터는 런타임 기본값으로 사용하지 않는다.

일반 CI의 명시적 미설정 exe는 오프라인 self-test와 설정 오류 종료만 검사한다. 수동 다운로드 workflow는 설정된 exe의 테스트·검증 후 exe·체크섬·출처 정보를 30일 보관한다. 실제 서버 주소 설정과 해당 다운로드 workflow 실행 결과는 별도 확인 대상이다.
