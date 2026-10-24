# 릴리스 수용 검증 근거

현재 상태: 자동 검사 구성은 있지만 실제 Windows 10/11·Blob·Discord 수용 검증과 버전 릴리스는 미완료다. [예시](evidence.example.json)는 의도적으로 통과하지 않는 작성 양식이다.

검증 담당자는 게시할 정확한 앱 commit·버전·빌드 origin으로 후보를 만들고 아래 검사를 수행한다. 실제 결과와 재현 범위가 있는 HTTPS 문서 또는 CI 실행 링크를 각 `evidenceUrl`에 넣는다. 원본의 계약 revision·SHA-256은 `contracts/desktop-v1/source.json`과 일치해야 한다. `serverCommit`은 호환 배포에서 검증한 서버 구현 commit이며 계약 파일만 추가한 commit과 구분한다. `deploymentUrl`과 근거에는 배포 URL의 commit 대응을 기록한다. 공개 산출물에 포함되는 URL은 인증 정보·query·fragment 없이 작성한다. 실제로 통과한 검사만 `status`를 `passed`로 변경한다. 정식 릴리스에는 모든 검사의 통과가 필요하다.

| 검사 키                     | 필요한 근거                                                                     |
| --------------------------- | ------------------------------------------------------------------------------- |
| `serverCheck`               | 해당 서버 commit의 `npm run check`                                              |
| `serverOpenSpec`            | 서버 OpenSpec 전체 strict 검사                                                  |
| `postgresAuthorizationRace` | 실제 다중 연결 PostgreSQL 권한·회수·경쟁 검사                                   |
| `windows10`, `windows11`    | 일반 사용자 UI·한국어 입력/경로·DPI·OneDrive·실제 LoL 경로·다운로드/미서명 실행 |
| `discordLogin`              | 브라우저 승인부터 앱 연결까지 실제 Discord 흐름                                 |
| `blobPublicPrivate`         | 실제 public/private Blob 전송·헤더·만료·overwrite 검사                          |
| `replayDuplicateBrowser`    | 실제 리플레이 저장·중복 처리와 기존 웹 업로드 회귀                              |
| `revocation`                | 실제 연결 해제·권한 회수·공유 한도·잔여 URL/객체 정리                           |

완성 JSON을 저장소 Actions 변수 `LLVY_RELEASE_EVIDENCE_JSON`에 등록한다. 앱 commit·버전·origin이 달라지면 해당 후보를 다시 검증하고 기록을 갱신한다. 정식 릴리스 workflow는 예시 주소·pending·누락·revision 불일치를 거부한다. 이 검사는 검토된 기록의 일관성을 확인하며 링크 내용이나 원격 테스트 실행의 진실성을 자동으로 증명하지 않는다. Release 게시 권한과 변수 편집 권한을 제한하고 태그 보호 정책으로 검토된 commit을 게시한다.

일반 로컬 빌드는 근거 없이 가능하다. 게시 workflow와 게시 스크립트는 모두 근거를 검사한다. 배포 패키지는 `release-evidence.json`과 [의존성 라이선스 고지](../../THIRD_PARTY_NOTICES.txt)를 포함하며 게시 후 digest로 다시 내려받아 모든 파일을 대조한다. 외부 고지는 업로더 자체 코드의 라이선스 결정을 대신하지 않는다.

로컬 검사는 네트워크 요청 없이 수행한다.

```powershell
$env:RELEASE_VERSION = "1.2.3"
$env:SOURCE_COMMIT = git rev-parse HEAD
$env:LLVY_SERVER_ORIGIN = "https://server.example.invalid" # 검증한 실제 origin
go run ./scripts/validate-release ./reviewed-evidence.json
```

라이선스/의존성 버전 변경 시 원문 출처와 고정 버전을 다시 대조하여 `THIRD_PARTY_NOTICES.txt`를 갱신한다. 빌드·검증 명령은 [개발 안내](../development.md)를 따른다.

## 테스트용 사전 릴리스

GitHub Release의 **Set as a pre-release**를 켜고 `v0.1.0-rc.1`처럼 SemVer 접미사가 있는 버전을 사용한다. workflow는 GitHub prerelease 표시에서 `RELEASE_CHANNEL`을 도출하며 버전 접미사와 맞지 않으면 실패한다. 기록의 `channel`도 `prerelease`여야 한다. 정식 릴리스의 기본값은 stable이며 모든 검증 passed 조건을 유지한다.

사전 릴리스도 정확한 client/version/origin/contract/server 대응과 `serverCheck`, `serverOpenSpec`, `postgresAuthorizationRace`의 passed 근거가 필요하다. `windows10`, `windows11`, `discordLogin`, `blobPublicPrivate`, `replayDuplicateBrowser`, `revocation`은 실제 검증 전까지 pending으로 남길 수 있으며, 각 `evidenceUrl`은 미검증 범위를 설명한 HTTPS 문서를 가리켜야 한다. failed·누락·예시 URL은 허용하지 않는다.

Linux/Windows tests·race·vet, PE 검사, 실행 파일 self-test와 내장 서버 주소 검사, 게시 후 digest 다운로드와 파일 해시 대조는 사전 릴리스에서도 필수다. provenance·패키지 설명·Release 안내는 미서명 테스트용이며 실제 PC와 외부 서비스 수용 검증이 남아 있음을 명시한다. GitHub-hosted Windows 검사 성공은 Windows 10/11 일반 사용자 UI·IME·DPI·OneDrive·실제 Discord/Blob 사용을 대체하지 않는다. 이 항목들은 OpenSpec에서 열린 상태를 유지한다.

사전 릴리스 근거를 로컬 확인할 때만 `RELEASE_CHANNEL=prerelease`를 지정한다. 게시 workflow에서는 이 값을 수동 입력으로 받지 않고 Release metadata로 결정한다.
