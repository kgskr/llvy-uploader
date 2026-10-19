## 1. 구현 기반과 선행 기술 검증

- [ ] 1.1 현재 웹 로그인·업로드 공개 계약과 관련 활성 changes를 재확인하고 기존 회귀 테스트 baseline을 기록한다.
- [ ] 1.2 `clients/windows-uploader/` 독립 Go 모듈과 OS 독립/Windows 전용 패키지 경계를 만들고 Go·Windows 바인딩 버전을 고정한다.
- [ ] 1.3 Win32 한국어 입력·폴더 선택·HTTPS·DPAPI가 포함된 최소 exe를 Windows에서 실행하고 크기/메모리/시작 시간/DLL 의존성을 기록하여 네이티브 UI 선택을 검증한다.
- [ ] 1.4 현재 Blob SDK 서명 PUT을 Go로 실제 테스트 store에 전송하고 public/private 필수 헤더, access 누락/변조, Control API 요청 URL과 객체 응답 URL 구분, 다른 경로·초과 크기·만료·덮어쓰기 거부를 확인한다. 미검증이면 후속 전송 구현 완료를 주장하지 않는다.
- [ ] 1.5 `/api/desktop/v1/` 요청·응답·오류·버전 정책과 공유 JSON 계약 fixture를 작성한다. transfer의 URL/method/headers/만료 및 헤더 allowlist, session의 account/global 한도 snapshot을 포함하고 실제 관찰에 맞춰 design/specs를 조정한다.

## 2. 추가 데이터 모델과 권한 기반

- [ ] 2.1 `auth_sessions.kind`의 browser 기본값과 desktop 세션/연결 요청/credential 테이블을 추가하는 마이그레이션을 작성하고 모드·교체 대상·절대 만료·refresh tombstone 보존을 반영한다.
- [ ] 2.2 desktop upload job의 기기/예약 바인딩, 멱등 키·fingerprint unique 제약, 서버 hash·terminal 결과 7일 보존 및 기기 절대 만료까지의 멱등 tombstone 구조를 추가한다.
- [ ] 2.3 disposable DB에서 기존 browser 세션/예약/게임 보존, 중간 migration 실패 rollback과 제약을 검증한다.
- [ ] 2.4 device actor·scope 및 kind-aware 인증 어댑터를 추가하고 기존 cookie 경로에서 desktop credential을 거부한다.
- [ ] 2.5 기존 계정 → 세션 → grant 잠금 순서에 desktop/credential/job 잠금을 일관되게 연결하고 최종 저장까지 scope·원래 grant·활성 claim을 재검증한다.
- [ ] 2.6 `revokeAdministrator`와 `applyMembership` 양쪽에서 대상 grant/계정의 desktop session·credential·미완료 pending/job·claim 무효화 및 감사를 공통 잠금 순서의 같은 transaction에 연결한다. 완료 이력을 보존하고 owner 변경/secret rotation을 포함한 전체 기기/미완료 인증 요청 영구 회수 절차를 준비한다.
- [ ] 2.7 `AuditChanges`/`safeChanges`에 검증된 deviceSessionId/channel/mode/replacedSessionId를 명시적으로 허용하고 저장 행 조회로 메타데이터 보존·비밀/임의 필드 제거를 검증한다.

## 3. 브라우저 승인과 기기 토큰 서버

- [ ] 3.1 연결 시작 API에 고정 client ID, literal loopback URI, PKCE S256/state 검증과 trusted IP/global 예산 및 짧은 수명 요청 저장을 구현한다.
- [ ] 3.2 `/desktop/connect`에 기존 Discord 로그인 복귀, 계정/권한·모드/수명 표시, 명시적 승인/거절, same-origin/CSRF 검증과 요청에 바인딩한 임시 연결 교체 선택을 추가한다.
- [ ] 3.3 승인 시 60초 일회용 코드 발급, 교환 시 승인 browser session/현재 권한 재검증 및 원자적 소비를 구현한다.
- [ ] 3.4 최대 15분 access·transient 12시간/persistent 30일 절대 수명, domain-separated hash 저장, 계정 잠금 아래 활성 기기 합계 5개 제한과 승인된 transient 회수/신규 발급의 원자적 교체를 구현한다.
- [ ] 3.5 refresh rotation·소비 token 재사용 시 family 회수·bounded 갱신 예산을 구현하고 tombstone 유지, cleanup 후 재사용 감지 및 응답 유실/동시 갱신 정책을 검증한다.
- [ ] 3.6 본인 기기 목록/해제 화면과 앱 자체 revoke API를 추가하고 browser/device 로그아웃의 독립성을 검증한다.
- [ ] 3.7 만료 요청·code·credential의 bounded cleanup, 비밀 없는 감사/진단, provider 장애와 무관한 maintenance를 구현하고 기존 auth-session cleanup이 refresh/멱등 tombstone 및 7일 결과를 조기 삭제하지 않게 한다.

## 4. 기존 웹을 보존하는 데스크톱 업로드 서버

- [ ] 4.1 기존 업로드 handler에서 재사용할 검증·저장·정리 서비스를 추출하고 웹 URL/DTO/권한/오류 회귀 테스트를 유지한다.
- [ ] 4.2 새 namespace의 Proxy 분기, 각 handler/DAL의 bearer·scope 검사, `DESKTOP_UPLOADER_ENABLED` 기본 false 및 비활성화 동작을 구현한다.
- [ ] 4.3 `consumeRequestBudgetsInTx`를 분리하고 actor → 기기 행 FOR UPDATE → 멱등 키 조회 → 정렬된 budget 차감 → pending/job/감사 삽입을 같은 transaction으로 묶는다. 기존 wrapper도 같은 예산 키 정렬을 사용하고 예산 부족/후속 실패 시 전체 rollback, 기존 키의 미차감 및 nonce 재생성을 구현한다.
- [ ] 4.4 정확한 store/path, PUT only, 5분 이하 만료, content type/크기/overwrite 제한을 갖는 전송 승인 API와 서버 소유의 URL/method/headers descriptor를 구현하고 전송 endpoint와 객체 URL 검증을 분리한다.
- [ ] 4.5 desktop process가 uploadId/nonce만 받아 예약 store/path에서 canonical URL을 도출하도록 하고 기기 바인딩·서버 hash/size 검증·기존 파서 budget·duplicate·mtime 정책을 연결한다.
- [ ] 4.6 `ingestReplay`의 사전 duplicate·신규 삽입·경합 duplicate 세 분기에 transaction finalizer를 연결해 최신 claim 검사, 서버 hash, pending/job terminal·감사·신규 게임/참가자/cache revision을 함께 커밋한다. 반환 후 별도 상태 갱신에 의존하지 않고 동일 요청의 202/저장된 결과 및 상태 조회를 구현한다.
- [ ] 4.7 처리 claim 만료/worker 종료/취소의 unknown 결과와 정리 정책을 구현하고 pending cleanup이 7일 job 결과나 canonical Blob을 삭제하지 않게 한다. 결과 정리 후에도 기기 수명 동안 같은 키 요청을 tombstone으로 거부하고 추가 예산을 소비하지 않게 한다.

## 5. Windows 폴더 탐색과 게임 ID 매칭

- [ ] 5.1 Known Folder, 저장된 사용자 경로, 알려진 설치 후보를 다루는 discovery adapter와 fake-filesystem 테스트를 구현한다.
- [ ] 5.2 실제 Windows에서 설정 파일 형식/파일명 사례를 확인하고 검증된 replay 경로 설정만 읽는 adapter를 추가한다. 확인하지 못한 형식은 지원한다고 표시하지 않는다.
- [ ] 5.3 단일/복수 후보 선택, 빈 폴더·설치 없음·저장 경로 삭제·권한 오류에 대한 한국어 안내와 native picker 복구를 구현한다.
- [ ] 5.4 경기 ID 입력 정규화·100개 제한·정확 토큰 매칭·플랫폼/복사본 모호성·동일 파일 dedup을 테스트와 함께 구현한다.
- [ ] 5.5 직속 일반 `.rofl` 색인, name-surrogate 차단과 Cloud Files 구분, 로컬 다운로드 필요/오프라인 안내, 크기/mtime 재검증, 파일 잠금과 스트리밍 hash/read를 구현한다.

## 6. Go 인증과 HTTP 클라이언트

- [ ] 6.1 고정 제품 origin, 개발 빌드 설정, TLS/timeout/응답 크기 제한, redirect 거부와 credential redaction을 구현한다. transfer descriptor의 endpoint/헤더 allowlist·중복 이름·CR/LF·금지 헤더 검증과 파일 크기 기반 Content-Length를 적용한다.
- [ ] 6.2 loopback 선점·state/verifier 생성·브라우저 실행·callback 검증·token exchange·취소/timeout listener 정리를 구현한다.
- [ ] 6.3 메모리 access 관리, single-flight refresh, uncertain refresh의 재로그인과 401/403 처리를 구현하고 자격 증명을 읽기 전 사용자/origin별 단일 인스턴스 잠금을 적용한다.
- [ ] 6.4 기본 꺼짐인 로그인 유지, user-scope DPAPI refresh의 원자적 저장, 복호화/새 토큰 저장 실패 처리, transient 종료 시 최대 2초 revoke, 로컬 로그아웃/기록 삭제와 held lock 보존을 구현한다.
- [ ] 6.5 Go와 TypeScript에서 동일 계약 fixture를 읽고 인증·버전 불일치·금지된 API 접근과 토큰 종류 혼동을 검증한다.
- [ ] 6.6 Windows에서 exe 경로/버전이 다른 두 프로세스의 credential 접근 차단, lock 보유 프로세스 crash 후 재실행, 기본 로그인/종료 6회 및 오프라인 잔여 연결의 명시적 교체를 검증한다.

## 7. 업로드 큐와 사용자 화면

- [ ] 7.1 단일 창에 경로·로그인 상태·ID 입력·파일 후보/결과 목록·버튼·진행률을 구현하고 UI thread 경계를 적용한다. 업로드 전 선택 수와 현재 24시간 구간의 account/global 최소 잔여 예약·관찰/갱신 시각, 초과 선택 및 중복/실패 예약 소비 안내를 표시한다.
- [ ] 7.2 명시적 사용자 실행 후 한 파일씩 예약 → 서명 PUT → 처리하는 큐와 파일별 진행률을 구현한다.
- [ ] 7.3 멱등 예약 재시도, PUT 응답을 전혀 받지 못한 경우 uploadId/nonce만으로 서버 확인, 처리 응답 유실 후 상태 조회와 최대 3회 safe retry를 구현한다.
- [ ] 7.4 429/reset 안내, 인증/권한 회수 시 큐 정지, 단일 파일 실패 후 계속 진행, 선택한 실패 항목 재시도를 구현한다.
- [ ] 7.5 취소/종료/결과 불명확 표시, 기존 게임 링크·duplicate 성공 처리와 원본 파일 보존을 검증한다.

## 8. 독립 빌드와 GitHub Windows CI

- [ ] 8.1 root lint/typecheck/format 및 Vercel 입력/빌드 생략 경계를 조정하고 client-only·server/shared-contract 변경을 구분하는 workflow filter를 검증한다.
- [ ] 8.2 ARM Mac 공통 Go tests/vet/race 검사와 Windows `windows-2025` x64 tests/vet/전용 연동 검증을 구성한다.
- [ ] 8.3 고정 resource 생성 도구로 asInvoker/DPI/icon/version manifest를 포함하고 `CGO_ENABLED=0` GUI exe와 bounded self-test 실행을 검증한다.
- [ ] 8.4 DLL 의존성, checksum, license/dependency 정보, source commit provenance와 CI artifact 보존 정책을 추가한다.
- [ ] 8.5 보호된 `uploader-v*` release workflow, fork/PR secret 격리, 서명 상태 표기와 API 호환성 gate를 구성한다.

## 9. 자동 회귀 및 실제 PostgreSQL 검증

- [ ] 9.1 기존 웹 로그인/업로드 contract, viewer·다른 계정·다른 기기·재부여 grant·owner A→B→A 및 중간 요청 없음·scope 우회·정확 Blob 경로 회귀를 실행한다.
- [ ] 9.2 실제 disposable localhost PostgreSQL 다중 연결로 code 소비, refresh 회전, 마지막 예산 경쟁, 동일 기기/키 최초 동시 예약, 반대 순서로 입력한 budget 키, 삽입/감사 실패 후 예산 rollback, 명시적 grant 회수 및 탈퇴/device 회수와 최종 저장의 lock 경쟁을 재현해 검증한다.
- [ ] 9.3 mock HTTP 및 Windows 테스트로 응답 유실·네트워크 중단·만료·파일 변경·429·취소·재시도·처리 worker 종료를 검증하고 PUT 무응답 복구 및 결과 정리 8일차 멱등 키 재사용 계약 fixture를 Go/서버에서 공유한다.
- [ ] 9.4 관련 Go/Windows 검증, `LLVY_REPLAY_FILE`을 설정한 실제 replay 테스트, 코드 변경에 대한 `npm run check`와 `git diff --check`를 통과시킨다.
- [ ] 9.5 `openspec validate --all --strict`와 정식/델타 명세 대조를 수행하고 소스·테스트·문서가 요구사항을 구현했는지 확인한다.
- [ ] 9.6 사전 duplicate·신규 game·unique 경합 duplicate 각각에서 finalizer 실패/claim 만료의 전체 rollback, 성공 후 응답 유실의 결과 복구, canonical Blob 보존과 웹 회귀를 검증한다. 감사 메타데이터 실제 저장 및 한도 snapshot 이후 다른 클라이언트의 소진 UI도 확인한다.

## 10. 외부 서비스와 실제 사용자 PC 검증

- [ ] 10.1 스테이징에서 기존 browser 업로드와 브라우저 승인 → Go token → 실제 Blob → 실제 ROFL 저장/duplicate 전체 흐름을 검증한다.
- [ ] 10.2 스테이징에서 웹·앱 공유 한도, 연결 해제/탈퇴 후 refresh·처리 거부, 서명 URL 잔여 수명과 고아 객체 정리를 확인한다.
- [ ] 10.3 실제 Windows 10/11 일반 사용자에서 무설치 실행, 한국어 IME/사용자 경로, DPI, OneDrive 문서 이동과 hydrated/pinned/online-only/오프라인 파일별 처리, 기본/변경된 LoL replay 경로를 확인한다.
- [ ] 10.4 서명 또는 미서명 산출물의 실제 다운로드·SmartScreen·실행 결과를 기록하고 CI 결과와 외부/수동 검증의 한계를 구분한다. 접근할 수 없는 환경은 이 작업을 열린 상태로 남긴다.

## 11. 문서와 릴리스 준비 및 아카이브

- [ ] 11.1 사용자용 실행·폴더 탐색·게임 ID·로그인 유지·기기 해제·실패 복구·업로드 한도 설명과 개발/CI/인증·API 문서를 작성한다.
- [ ] 11.2 schema 선행 적용, 기능 flag 활성화, server-first 호환 배포, device-only 비활성화/회수 rollback 절차를 정리하고 staging에서 검증한다. owner 변경/rollback은 과거 URL을 포함한 모든 serving version 차단·발급 drain·기기/미완료 인증 요청 영구 회수·기존 token 거부 검증을 필수 gate로 둔다.
- [ ] 11.3 실제 측정 크기·지원 OS·서명 상태·검증된 commit을 포함한 릴리스 산출물을 준비하고 승인된 배포 범위에서만 발행한다. 구매/운영 DB 적용을 계획 작성 승인으로 간주하지 않는다.
- [ ] 11.4 모든 구현/필수 검증 완료 후 정식 명세와 델타 차이를 검토하고 `openspec archive add-go-windows-replay-uploader --yes`를 실행한 뒤 전체 strict 검증과 활성 목록 제외를 확인한다.
