# Windows 업로더 저장소 분리

분리일: 2026-10-18. 상태: 계획 이전 완료, 기능 구현 미완료.

기존 `clients/windows-uploader/` 배치는 별도 `llvy-uploader` 저장소로 대체했다.
앱 Go 모듈·Windows CI·릴리스는 앱 저장소, 서버 인증·기기 관리·API·DB는 LLVY가 담당한다.
서버의 공개 계약은 클라이언트 준수 조건도 정의하며 앱은 [고정 계약 정책](../contracts/desktop-v1/README.md)을 따른다.

- [서버 계획](../../llvy/openspec/changes/add-desktop-replay-upload-api/proposal.md)
- [앱 계획](../openspec/changes/add-go-windows-replay-uploader/proposal.md)
- [분리 전 원본과 리뷰](history/2026-10-18-monorepo-plan/README.md)

## 기존 61개 작업의 목적지

아래 번호는 분리 전 작업 번호다. 공동 항목은 양쪽에 필요한 작업을 각자의 범위로 나눴다.
어떤 기능·실환경 검증도 이전했다는 이유로 완료 처리하지 않았다.

| 기존 번호 | 현재 담당            |
| --------- | -------------------- |
| 1.1       | LLVY 서버            |
| 1.2       | 업로더 앱            |
| 1.3       | 업로더 앱            |
| 1.4       | LLVY 서버, 업로더 앱 |
| 1.5       | LLVY 서버, 업로더 앱 |
| 2.1       | LLVY 서버            |
| 2.2       | LLVY 서버            |
| 2.3       | LLVY 서버            |
| 2.4       | LLVY 서버            |
| 2.5       | LLVY 서버            |
| 2.6       | LLVY 서버            |
| 2.7       | LLVY 서버            |
| 3.1       | LLVY 서버            |
| 3.2       | LLVY 서버            |
| 3.3       | LLVY 서버            |
| 3.4       | LLVY 서버            |
| 3.5       | LLVY 서버            |
| 3.6       | LLVY 서버            |
| 3.7       | LLVY 서버            |
| 4.1       | LLVY 서버            |
| 4.2       | LLVY 서버            |
| 4.3       | LLVY 서버            |
| 4.4       | LLVY 서버            |
| 4.5       | LLVY 서버            |
| 4.6       | LLVY 서버            |
| 4.7       | LLVY 서버            |
| 5.1       | 업로더 앱            |
| 5.2       | 업로더 앱            |
| 5.3       | 업로더 앱            |
| 5.4       | 업로더 앱            |
| 5.5       | 업로더 앱            |
| 6.1       | 업로더 앱            |
| 6.2       | 업로더 앱            |
| 6.3       | 업로더 앱            |
| 6.4       | 업로더 앱            |
| 6.5       | 업로더 앱            |
| 6.6       | 업로더 앱            |
| 7.1       | 업로더 앱            |
| 7.2       | 업로더 앱            |
| 7.3       | 업로더 앱            |
| 7.4       | 업로더 앱            |
| 7.5       | 업로더 앱            |
| 8.1       | LLVY 서버            |
| 8.2       | 업로더 앱            |
| 8.3       | 업로더 앱            |
| 8.4       | 업로더 앱            |
| 8.5       | 업로더 앱            |
| 9.1       | LLVY 서버            |
| 9.2       | LLVY 서버            |
| 9.3       | LLVY 서버, 업로더 앱 |
| 9.4       | LLVY 서버, 업로더 앱 |
| 9.5       | LLVY 서버, 업로더 앱 |
| 9.6       | LLVY 서버            |
| 10.1      | LLVY 서버, 업로더 앱 |
| 10.2      | LLVY 서버, 업로더 앱 |
| 10.3      | 업로더 앱            |
| 10.4      | 업로더 앱            |
| 11.1      | LLVY 서버, 업로더 앱 |
| 11.2      | LLVY 서버            |
| 11.3      | 업로더 앱            |
| 11.4      | LLVY 서버, 업로더 앱 |

## 명세와 리뷰

`windows-replay-uploader`, `windows-uploader-release`는 앱으로 이동했다.
Windows 로그인 유지·DPAPI 요구사항은 앱의 `desktop-client-authorization`로 이동했다.
인증 공개 계약·업로드 API·기존 웹의 세 델타 명세는 LLVY에 남았다.
기존 리뷰에서 사용하는 작업 번호와 모든 시나리오는 새 계획 또는 참조 계약에 보존한다.
과거 세 에이전트 리뷰와 후속 리뷰는 당시 계획 검토이며 이번 분리의 새 보안 리뷰를 의미하지 않는다.

## 완료 구분

LLVY의 `separate-windows-uploader-repository`는 저장소·계획·문서 분리만 기록한다.
서버와 앱의 기능 변경은 각각 열린 상태로 남아 있다. 실제 Go·Windows·DB·Blob·Discord 검증은 후속 구현 범위다.
