## Why

사용자가 리플레이를 경기 번호로 찾아 일괄 업로드할 수 있도록 Windows 포터블 앱을 제공한다. 웹과 다른 Go·Windows 빌드·릴리스 주기를 별도 저장소에서 관리한다.

## What Changes

- 독립 Go 모듈, Win32 기본 UI와 Windows x64 단일 exe를 구현한다.
- 파일 탐색·경기 ID 매칭·선택·순차 업로드와 상태 복구를 구현한다.
- LLVY 서버의 브라우저 승인·PKCE 계약을 소비하며 로컬 기기 자격 증명은 DPAPI로 보호한다.
- LLVY 계약 원본의 고정 사본으로 검사하고 독립 Windows CI·릴리스를 구성한다.

## Capabilities

### New Capabilities

- `windows-replay-uploader`: UI, 파일 탐색, 매칭과 업로드 큐.
- `desktop-client-authorization`: 기기 프로토콜 소비와 Windows 자격 증명 저장.
- `windows-uploader-release`: 독립 빌드·계약 고정·Windows 실행 검증·exe 배포.

### Modified Capabilities

없음. 서버 요구사항은 LLVY에서 관리한다.

## Impact

앱과 CI·릴리스·사용자 문서를 위한 활성 구현 계획이다. 로컬 파일 미리보기, Windows 어댑터와 빌드·검사 구성을 먼저 구현하며 로그인·전송은 고정 서버 계약을 기다린다. 서버 인증·DB·API는 [LLVY 서버 계획](../../../../llvy/openspec/changes/add-desktop-replay-upload-api/proposal.md)에 있다. 실제 Windows·외부 서비스 검증과 릴리스는 미완료다. 기존 요구사항과 리뷰의 목적지는 [분리 이력](../../../docs/repository-split.md)을 따른다.
