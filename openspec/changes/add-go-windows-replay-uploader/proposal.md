## Why

사용자가 리플레이를 경기 번호로 찾아 일괄 업로드할 수 있도록 Windows 포터블 앱을 제공한다. 웹과 다른 Go·Windows 빌드·릴리스 주기를 별도 저장소에서 관리한다.

## What Changes

- 독립 Go 모듈, Win32 기본 UI와 Windows x64 단일 exe를 구현한다.
- 파일 탐색·경기 ID 매칭·선택·순차 업로드와 상태 복구를 구현한다.
- LLVY 서버의 브라우저 승인·PKCE 계약을 소비하며 로컬 기기 자격 증명은 DPAPI로 보호한다.
- refresh가 토큰 교체 전에 `429 rate_limited`로 거부되면 자격 증명과 예약을 유지하고 대기 정보를 전달한다. 시작 시 보관 연결 복구도 같은 규칙을 따른다.
- 기존 예약은 원본 파일 접근 전에 서버에서 복구하고, 확정 실패·만료·결과 불명확의 새 시도는 한도 안내와 사용자 확인 후 시작한다.
- LLVY 계약 원본의 고정 사본으로 검사하고 독립 Windows CI·릴리스를 구성한다.
- 교체 파일·한도 안내·제한 크기 진단 로그를 보완하고 실제 검증 근거 및 라이선스 묶음이 없는 릴리스 게시를 차단한다.
- 서버 주소의 기본값 없이 빌드 변수로 HTTPS origin을 주입하고 게시된 버전 릴리스에서만 배포 exe를 만들고 버전 파일명·체크섬·출처를 GitHub Release의 다운로드 첨부 파일로 게시한다.

- session 역할 `uploader`를 관리자/오너와 함께 허용하고 권한 회수 뒤 재연결을 안내한다. 새 계약 revision 고정과 실제 검증은 별도로 완료한다.

## Capabilities

### New Capabilities

- `windows-replay-uploader`: UI, 파일 탐색, 매칭과 업로드 큐.
- `desktop-client-authorization`: 기기 프로토콜 소비와 Windows 자격 증명 저장.
- `windows-uploader-release`: 독립 빌드·계약 고정·Windows 실행 검증·exe 배포.

### Modified Capabilities

없음. 서버 요구사항은 LLVY에서 관리한다.

## Impact

앱과 CI·릴리스·사용자 문서를 위한 활성 구현 계획이다. 로컬 파일 탐색, Windows 어댑터와 빌드·검사 구성, 고정 후보 계약 기반 로그인·전송·복구를 구현했다. 서버 인증·DB·API는 [LLVY 서버 계획](../../../../llvy/openspec/changes/add-desktop-replay-upload-api/proposal.md)에 있다. 실제 Windows·외부 서비스 검증과 릴리스는 미완료다. 기존 요구사항과 리뷰의 목적지는 [분리 이력](../../../docs/repository-split.md)을 따른다.

## 테스트용 사전 릴리스

정식 릴리스의 실제 PC·외부 서비스 검증 조건은 유지한다. GitHub prerelease 표시와 SemVer 사전 버전 접미사가 모두 있는 테스트용 패키지에 한해, 서버 check·strict·PostgreSQL 검증 및 Linux/Windows 자동 검사를 통과하고 실제 PC·외부 검증의 pending 상태를 명시한 근거를 포함하면 게시할 수 있다. 이를 정식 지원이나 OpenSpec 전체 완료로 기록하지 않는다.
