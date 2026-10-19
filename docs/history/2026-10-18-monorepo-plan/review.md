# Go Windows 리플레이 업로더 계획 리뷰

검토일: 2026-10-16. 대상은 이 change의 계획 산출물이며 앱/서버 구현 리뷰가 아니다. 세 서브에이전트가 인증·권한, 업로드 계약, Go·Windows·검증 계획을 나눠 읽고 기존 코드와 대조했다. 주 에이전트가 지적을 확인하여 설계·명세·작업 목록을 수정했고 같은 담당자에게 재검토를 요청했다.

## 1차 서브에이전트 리뷰와 반영

P2 6건, P3 1건이다. 우선순위는 구현 전에 고쳐야 할 계획의 영향도를 뜻하며 운영 중인 취약점 등급을 뜻하지 않는다.

| ID     | 우선순위 | 발견한 문제                                                                                                     | 계획에 반영한 해결책                                                                                                                                                                              | 연결된 작업        |
| ------ | -------- | --------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| AUTH-1 | P2       | 기본 로그인은 종료 시 비밀만 지워 서버에 세션이 남고, 5회 반복 후 새 연결이 막힐 수 있음                        | transient 12시간/persistent 30일 절대 수명, 정상 종료 최대 2초 revoke, crash 잔여 연결은 승인 화면에서 명시적으로 선택해 원자적 교체. 모드·교체 대상은 승인 요청에 고정                           | 3.2, 3.4, 6.4, 6.6 |
| AUTH-2 | P2       | 소비한 refresh 해시를 정리하면 과거 토큰의 family를 식별할 수 없어 재사용 시 전체 회수 약속을 지킬 수 없음      | hash/family tombstone을 family 절대 만료까지 유지. 영구 회수로 모든 credential이 계속 거부될 때만 조기 정리. maintenance 후 과거 세대 재사용 검증                                                 | 2.1, 3.5, 3.7      |
| AUTH-3 | P2       | 환경변수 owner가 A→B→A로 바뀌는 동안 앱 요청이 없으면 단순 현재 값 비교로 옛 권한이 살아날 수 있음              | 모든 접근 가능한 배포의 desktop 차단·발급 drain, 전체 기기/미완료 연결 요청 영구 회수, owner 변경, 이전 token 거부 검증을 변경/rollback 필수 절차로 지정. 자동 환경변수 이력 감지는 주장하지 않음 | 2.6, 9.1, 11.2     |
| API-1  | P2       | PUT 응답을 전혀 받지 못하면 process 입력으로 요구한 Blob URL을 알 수 없어 복구할 수 없음                        | desktop process 입력은 uploadId/nonce로 한정하고 서버가 예약의 configured store/path로 canonical URL 도출. 웹 URL 입력 계약은 유지                                                                | 4.5, 7.3, 9.3      |
| API-2  | P3       | 결과 보존 7일 후 멱등 키도 삭제하면 아직 유효한 30일 기기가 같은 키로 새 예약·예산 소비를 할 수 있음            | 상세 결과는 7일, key/fingerprint tombstone은 기기 절대 만료까지 보존. 8일차 재사용은 409이며 예산 미소비                                                                                          | 2.2, 4.7, 9.3      |
| WIN-1  | P2       | exe 복수 실행이 같은 DPAPI refresh를 읽으면 프로세스 내부 single-flight만으로 중복 갱신·family 회수를 막지 못함 | 사용자/origin별 exclusive lock handle을 credential read 전에 확보. 경로·버전이 다른 두 exe도 하나만 실행. crash 시 OS handle 해제, refresh 파일 원자 교체                                         | 6.3, 6.4, 6.6      |
| WIN-2  | P2       | 모든 reparse point를 제외하면 OneDrive 파일 지원 정책과 충돌할 수 있음                                          | name-surrogate와 cloud placeholder를 구분. 전체 바이트가 로컬에 있으면 지원하고 online-only는 `로컬 다운로드 필요`로 안내. 실제 hydrated/pinned/online-only/오프라인 검증 추가                    | 5.5, 10.3          |

## 검토 근거와 범위

- [인증 명세](specs/desktop-device-authorization/spec.md), [업로드 API 명세](specs/desktop-replay-upload-api/spec.md), [Windows 명세](specs/windows-replay-uploader/spec.md)에 실패 시나리오와 기대 결과를 반영했다. [설계](design.md)와 [작업 목록](tasks.md)도 함께 갱신했다.
- 현재 `src/lib/auth.ts`의 `getAuthorityConfig()`는 환경변수 owner만 읽는다. owner 변경 이력을 자동으로 갖는 것으로 가정하지 않았다. 기존 `src/lib/auth-maintenance.ts`의 세션 정리도 새 child record 보존 기간을 고려하도록 작업 3.7에 포함했다.
- 설치된 `@vercel/blob` 2.5.0의 공개 `issueSignedToken`/`presignUrl` 및 PUT URL 생성 구현을 확인했다. 서명 PUT이 불가능하다는 지적은 하지 않았으며 실제 store 제한 enforcement는 선행 검증 1.4로 유지했다.
- [Microsoft Cloud Files 문서](https://learn.microsoft.com/en-us/windows/win32/cfapi/build-a-cloud-file-sync-engine)와 [reparse tag 설명](https://learn.microsoft.com/en-us/windows/win32/fileio/reparse-point-tags)은 placeholder와 symlink를 구분한다. 일부 API의 호환성 처리가 tag를 숨기므로 모든 OneDrive 파일이 반드시 실패한다고 주장하지 않는다.
- 기존 `src/lib/expired-upload-cleanup.ts`에는 예약 만료 후 24시간 유예가 있다. 5분 전송 권한과 즉시 삭제의 경합을 현재 결함으로 보고하지 않았다. canonical Blob 보존, 기존 웹 DTO, 계정 예산 공유, 최종 저장 권한 잠금은 기존 계획을 유지했다.

## 후속 리뷰 반영

2026-10-16 후속 리뷰의 구현 주의점 6개를 현재 저장소와 대조했다. 아래는 기존 발견 7건의 추가 재심사가 아닌, 구현 계약을 구체화한 후속 보완이다. 이 후속 수정본을 새 서브에이전트가 재검토했다고 주장하지 않는다.

| 항목                 | 코드 대조 결과                                                                                                                                                                                | 반영 결정과 검증 작업                                                                                                                                                                                    |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ① 서명 PUT 헤더/URL  | `@vercel/blob/dist/chunk-UVSKRCEW.js`의 `buildPresignedPutUrl`은 Control API URL을 반환하며 `requestApi`/`createPutHeaders`가 access/version/store 등의 헤더를 별도로 공급함                  | transfer를 URL/method/headers/만료 descriptor로 정의. 서버 adapter가 값을 공급하고 Go는 allowlist를 검증. 전송 endpoint와 객체 URL 검증 분리. 실제 필수 헤더/access 변조 테스트는 1.4/1.5/4.4/6.1에 반영 |
| ② 예산 transaction   | `consumeRequestBudgets`는 자체 transaction을 끝내며 `createPendingUpload`는 actor transaction 전에 호출함. `lockActorAuthority`의 SHARE 잠금만으로 같은 기기의 최초 요청 둘을 직렬화하지 못함 | caller tx helper, 정렬된 예산 키, actor→기기 행 FOR UPDATE→기존 키 조회→신규 예산 차감→예약/job/감사 순서를 명시. 예산 부족 및 후속 실패는 outer tx 전체 rollback. 4.3/9.2                               |
| ③ ingest 원자성      | `ingestReplay`는 사전 duplicate와 신규/경합 duplicate의 transaction을 내부 완결하고 웹 handler는 나중에 pending 상태를 갱신함                                                                 | desktop transaction finalizer를 세 성공 분기 모두에 연결. 최신 claim 재검사와 결과/hash/감사/pending/job을 함께 커밋. 반환 후 별도 갱신을 desktop 완료 근거로 사용하지 않음. 4.6/9.6                     |
| ④ 명시적 관리자 회수 | `applyMembership`는 pending을 실패 처리하지만 `revokeAdministrator`는 grant/auth session만 회수함                                                                                             | 두 진입점 모두 대상 desktop credential 및 미완료 pending/job/claim 무효화를 같은 감사 transaction에 포함. 완료 결과/Blob/retention 보존. 2.6/9.2                                                         |
| ⑤ 감사 필터          | `AuditChanges`와 `safeChanges`의 고정 allowlist는 기기 필드를 지원하지 않음                                                                                                                   | UUID/enum으로 검증한 deviceSessionId/channel/mode/replacedSessionId를 기존 before/after JSON allowlist에 추가. 새 컬럼 대신 실제 DB readback과 비밀 제외 테스트로 검증. 2.7/9.6                          |
| ⑥ 한도 UX            | 계정 10회·전체 50회는 자정 기준 일일 할당이 아닌 기존 24시간 fixed window이며, 예약 후 중복/실패도 소비됨                                                                                     | session에 account/global 잔여량·resetAt·asOf snapshot, UI에 선택 수와 최소 잔여량 및 초과 선택 안내. 동시 요청 후 429 가능성과 중복 소비를 명시. 1.5/7.1/9.6                                             |

선택 사항인 clientSha256 기반 사전 중복 조회/예약 생략은 도입하지 않는다. 실제 바이트 검증과 현재 한도를 유지하고, 필요하면 별도 변경으로 검토한다. 코드와 서버·클라이언트 동작을 바꾸지 않고 설계/명세/미완료 작업만 보완했다.

## 검증과 남은 경계

1차 수정본은 세 담당 서브에이전트가 다시 검토하여 기존 7건이 계획 수준에서 해결되었음을 확인했고 추가 확정 모순은 보고하지 않았다. 후속 리뷰는 주 에이전트가 코드 대조 및 문서 일관성 검증을 수행했다.

- `openspec validate --all --strict`: 후속 수정 후 28개 통과, 실패 0개. 1차 검증의 27개에서 별도 변경의 `web-analytics` 정식 명세가 추가됐다.
- Prettier 문서 검사 통과. OpenSpec이 기본 ignore 대상이므로 이 change에 `--ignore-path /dev/null`을 명시했다.
- MODIFIED requirement의 기존 시나리오 22개 본문 보존 확인.
- 해당 change 범위의 `git diff --check` 통과 및 아직 untracked인 파일 12개의 후행 공백 별도 확인.
- 계획 산출물 4/4 완료, apply 상태 `ready`. 감사 저장 검증과 ingestion finalizer 검증을 명시하여 구현 작업은 61개이며 모두 미완료로, change를 열린 상태로 유지한다.

코드·Windows exe·실제 Discord/Blob·일반 사용자 PC 검증은 이번 리뷰에서 실행하지 않았다. Win32 최소 앱과 서명 PUT 검증, 실제 PostgreSQL 경쟁 검증, Windows 10/11 및 OneDrive 검증은 [작업 목록](tasks.md)의 완료 조건으로 남는다.
