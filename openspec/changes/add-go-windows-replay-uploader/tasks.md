기존 혼합 계획의 번호를 유지한다. 번호가 빠진 항목은 다른 저장소가 담당한다. 공동 검증은 각 저장소의 책임을 나눈다. 서버 후보 계약을 고정하고 인증·전송·결과 복구를 로컬 구현했다. 실제 Windows·Blob·Discord 검증은 미완료다. [현재 근거와 남은 작업](../../../docs/validation/2026-10-18-server-integration.md)을 함께 확인한다.

## 1. 구현 기반과 선행 기술 검증

- [x] 1.2 이 저장소 루트에 독립 Go 모듈과 OS 독립/Windows 전용 패키지 경계를 만들고 Go·Windows 바인딩 버전을 고정한다.
- [ ] 1.3 Win32 한국어 입력·폴더 선택·HTTPS·DPAPI가 포함된 최소 exe를 Windows에서 실행하고 크기/메모리/시작 시간/DLL 의존성을 기록하여 네이티브 UI 선택을 검증한다.
- [ ] 1.4 서버 담당의 실제 Blob PUT 검증에 Go 전송 harness로 참여한다. public/private 헤더·access 변조·Control API/객체 URL 구분·크기·경로·만료·overwrite 결과를 계약 버전과 함께 확인한다.
- [x] 1.5 LLVY desktop-v1 스키마·fixture 원본을 검토하고 source commit·체크섬을 기록한 고정 사본을 이 저장소에 추가한다. 단독 체크아웃 테스트와 session 한도·transfer URL/method/headers/만료 계약을 검증한다.

## 5. Windows 폴더 탐색과 게임 ID 매칭

- [x] 5.1 Known Folder, 저장된 사용자 경로, 알려진 설치 후보를 다루는 discovery adapter와 fake-filesystem 테스트를 구현한다.
- [ ] 5.2 실제 Windows에서 설정 파일 형식/파일명 사례를 확인하고 검증된 replay 경로 설정만 읽는 adapter를 추가한다. 확인하지 못한 형식은 지원한다고 표시하지 않는다.
- [x] 5.3 단일/복수 후보 선택, 빈 폴더·설치 없음·저장 경로 삭제·권한 오류에 대한 한국어 안내와 native picker 복구를 구현한다.
- [x] 5.4 경기 ID 입력 정규화·100개 제한·정확 토큰 매칭·플랫폼/복사본 모호성·동일 파일 dedup을 테스트와 함께 구현한다.
- [x] 5.5 직속 일반 `.rofl` 색인, name-surrogate 차단과 Cloud Files 구분, 로컬 다운로드 필요/오프라인 안내, 크기/mtime 재검증, 파일 잠금과 스트리밍 hash/read를 구현한다.

## 6. Go 인증과 HTTP 클라이언트

- [x] 6.1 고정 제품 origin, 개발 빌드 설정, TLS/timeout/응답 크기 제한, redirect 거부와 credential redaction을 구현한다. transfer descriptor의 endpoint/헤더 allowlist·중복 이름·CR/LF·금지 헤더 검증과 파일 크기 기반 Content-Length를 적용한다.
- [x] 6.2 loopback 선점·state/verifier 생성·브라우저 실행·callback 검증·token exchange·취소/timeout listener 정리를 구현한다.
- [x] 6.3 메모리 access 관리, single-flight refresh, uncertain refresh의 재로그인과 401/403 처리를 구현하고 자격 증명을 읽기 전 사용자/origin별 단일 인스턴스 잠금을 적용한다.
- [ ] 6.4 기본 꺼짐인 로그인 유지, user-scope DPAPI refresh의 원자적 저장, 복호화/새 토큰 저장 실패 처리, transient 종료 시 최대 2초 revoke, 로컬 로그아웃/기록 삭제와 held lock 보존을 구현한다.
- [x] 6.5 고정된 LLVY 계약 fixture를 Go에서 검사하고 원본 revision에 대한 서버 검증 결과와 대응시킨다. 인증·버전 불일치·scope/토큰 종류 혼동을 확인하며 로컬 형제 소스를 직접 읽지 않는다.
- [ ] 6.6 Windows에서 exe 경로/버전이 다른 두 프로세스의 credential 접근 차단, lock 보유 프로세스 crash 후 재실행, 기본 로그인/종료 6회 및 오프라인 잔여 연결의 명시적 교체를 검증한다.

## 7. 업로드 큐와 사용자 화면

- [ ] 7.1 단일 창에 경로·로그인 상태·ID 입력·파일 후보/결과 목록·버튼·진행률을 구현하고 UI thread 경계를 적용한다. 업로드 전 선택 수와 현재 24시간 구간의 account/global 최소 잔여 예약·관찰/갱신 시각, 초과 선택 및 중복/실패 예약 소비 안내를 표시한다.
- [x] 7.2 명시적 사용자 실행 후 한 파일씩 예약 → 서명 PUT → 처리하는 큐와 파일별 진행률을 구현한다.
- [x] 7.3 멱등 예약 재시도, PUT 응답을 전혀 받지 못한 경우 uploadId/nonce만으로 서버 확인, 처리 응답 유실 후 상태 조회와 최대 3회 safe retry를 구현한다.
- [ ] 7.4 429/reset 안내, 인증/권한 회수 시 큐 정지, 단일 파일 실패 후 계속 진행, 선택한 실패 항목 재시도를 구현한다.
- [ ] 7.5 취소/종료/결과 불명확 표시, 기존 게임 링크·duplicate 성공 처리와 원본 파일 보존을 검증한다.

## 8. 독립 빌드와 GitHub Windows CI

- [ ] 8.2 Windows `windows-2025` x64 tests/race/vet·형식·전용 연동 검증을 구성한다. ARM Mac 검사는 선택적 로컬 검증으로만 둔다.
- [ ] 8.3 고정 resource 생성 도구로 asInvoker/DPI/icon/version manifest를 포함하고 `CGO_ENABLED=0` GUI exe와 bounded self-test 실행을 검증한다.
- [ ] 8.4 DLL 의존성, checksum, license/dependency 정보, source commit provenance와 CI artifact 보존 정책을 추가한다.
- [ ] 8.5 보호된 `uploader-v*` release workflow, fork/PR secret 격리, 서명 상태 표기와 API 호환성 gate를 구성한다.

## 9. 앱 회귀와 서버 계약 검증

- [ ] 9.3 mock HTTP 및 Windows 테스트로 응답 유실·네트워크 중단·만료·파일 변경·429·취소·재시도·처리 worker 종료를 검증하고 PUT 무응답 복구 및 결과 정리 8일차 멱등 키 재사용 계약 fixture를 Go/서버에서 공유한다.
- [ ] 9.4 이 저장소의 Go tests/vet/지원 호스트 race와 Windows 검증, git diff --check를 통과시킨다. 연계 서버 변경의 npm run check·실제 replay 결과는 서버 commit과 함께 참조한다.
- [ ] 9.5 업로더에서 openspec validate --all --strict와 앱 정식/델타 명세 대조를 수행한다. 코드·Windows 실행·문서가 각 요구사항을 충족하는지 확인한다.

## 10. 외부 서비스와 실제 사용자 PC 검증

- [ ] 10.1 스테이징에서 기존 browser 업로드와 브라우저 승인 → Go token → 실제 Blob → 실제 ROFL 저장/duplicate 전체 흐름을 검증한다.
- [ ] 10.2 스테이징에서 웹·앱 공유 한도, 연결 해제/탈퇴 후 refresh·처리 거부, 서명 URL 잔여 수명과 고아 객체 정리를 확인한다.
- [ ] 10.3 실제 Windows 10/11 일반 사용자에서 무설치 실행, 한국어 IME/사용자 경로, DPI, OneDrive 문서 이동과 hydrated/pinned/online-only/오프라인 파일별 처리, 기본/변경된 LoL replay 경로를 확인한다.
- [ ] 10.4 서명 또는 미서명 산출물의 실제 다운로드·SmartScreen·실행 결과를 기록하고 CI 결과와 외부/수동 검증의 한계를 구분한다. 접근할 수 없는 환경은 이 작업을 열린 상태로 남긴다.

## 11. 문서와 릴리스 준비 및 아카이브

- [x] 11.1 사용자용 실행·폴더 탐색·경기 ID·로그인 유지·기기 해제·실패 복구·한도 안내와 앱 개발/Windows CI/릴리스 문서를 작성하고 서버 API 원본으로 연결한다.
- [ ] 11.3 실제 측정 크기·지원 OS·서명 상태·검증된 commit을 포함한 릴리스 산출물을 준비하고 승인된 배포 범위에서만 발행한다. 구매/운영 DB 적용을 계획 작성 승인으로 간주하지 않는다.
- [ ] 11.4 앱 구현과 실제 Windows·외부 연동 등 필수 검증 완료 후에만 add-go-windows-replay-uploader를 아카이브하고 strict 검증과 정식 명세 일치를 확인한다. 서버 change는 별도로 관리한다.

## 12. 복구 경계 회귀 수정

- [x] 12.1 처리 전 access 수명 확보, 미확정 예약의 한도 독립 복구, 미예약 파일 재선택과 취소 시 인증 상태 보존을 구현하고 회귀 테스트한다.
- [x] 12.2 Go tests/race/vet, Windows 교차 빌드, 문서·계약·OpenSpec strict 검사를 수행하고 외부 검증은 별도로 남긴다.

## 13. 세션 전환과 취소 상태 수정

- [x] 13.1 기기별 예약 소유권, 연결 변경 시 명시적 재선택, 로그인 후 session 재조회, 취소된 탐색 후속 동작 억제를 구현하고 회귀 테스트한다.
- [x] 13.2 Go tests/race/vet·Windows 교차 빌드·문서/strict 검사와 수정 후 서브에이전트 리뷰를 수행한다. 실제 Windows·외부 검증은 별도로 유지한다.

## 14. Windows 계약 체크섬 CI 수정

- [x] 14.1 Windows 줄바꿈 변환으로 발생한 체크섬 오류를 재현하고 계약 JSON의 LF 체크아웃을 고정한다. 원본 계약·체크섬은 유지한다.
- [x] 14.2 자동 CRLF 설정 체크아웃·기존 계약 테스트·문서·strict 검증과 수정 커밋의 Windows CI 결과를 확인한다. 실제 사용자 PC·외부 검증은 별도로 남긴다.

## 15. 빌드 주소 주입과 다운로드 아티팩트

- [x] 15.1 기본 서버 주소를 제거하고 빌드 origin 검증·주입 및 누락 시 실행/배포 차단을 구현한다. 고정 서버 계약 바이트는 유지한다.
- [x] 15.2 수동 Windows 아티팩트 워크플로와 30일 보관·체크섬·출처 정보를 추가하고 기존 CI의 주소 없는 self-test 빌드를 구분한다.
- [x] 15.3 origin 회귀 테스트·Go tests/race/vet·Windows 교차 빌드·CI와 문서/strict 검증을 수행한다. 실제 외부 업로드는 별도 완료 조건으로 유지한다.

## 16. Windows 전용 CI

- [ ] 16.1 macOS job을 제거하고 검사·다운로드 workflow의 Windows race 검사와 형식 검사를 구성한다. 실제 Windows CI와 문서/strict 검증을 확인한다.
