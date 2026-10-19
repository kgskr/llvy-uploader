# 로컬 미리보기 구현·검증

확인일: 2026-10-18. 상태: 부분 구현. Windows 실행과 실제 서비스 연결 미검증.

## 구현 범위

- 독립 Go 모듈과 고정 Windows 바인딩, 별도 리소스 빌드 도구 모듈.
- 경기 번호 정규화·중복 입력 제거·100개 제한·완전한 숫자 토큰 매칭과 플랫폼 모호성.
- 저장 경로, Documents Known Folder, 제한된 설치 후보, 직속 파일 메타데이터 색인과 후보 선택 화면.
- name-surrogate 및 알 수 없는 특수 reparse 제외, hydrated Cloud Files 허용, offline/recall 상태 안내. 실제 provider 동작은 미검증.
- 크기·mtime 재검증, 읽기 공유 핸들, 스트리밍 SHA-256과 rewind. 실제 전송에는 아직 연결하지 않았다.
- namespace별 단일 인스턴스 잠금, 원자적 설정 교체·삭제, Windows DPAPI 원시 기능. 토큰 프로토콜·저장은 미구현.
- Win32 한국어 컨트롤·폴더 선택·goroutine 탐색, DPI 처리, manifest·개발 버전 리소스, Windows 검사 workflow와 bounded self-test.

## 수행한 검사

macOS ARM64에서 Go 단위 테스트·vet·race, Windows amd64 교차 빌드·vet와 PE 구조 검사를 수행했다.
테스트 데이터는 합성 문자열·임시 파일·가상 파일시스템이며 실제 회원 데이터·리플레이 원본은 사용하지 않았다.
PE는 Windows x64 GUI subsystem과 리소스 섹션을 포함하며 명시적 import table은 `kernel32.dll`을 참조한다.
Win32 시스템 DLL의 동적 로딩은 실행 전 검증되지 않았다. 빌드 산출물은 약 5.1 MiB이며 최종 배포 크기를 보장하지 않는다.

재현 명령은 [개발 안내](../development.md)에 있다. 원격·commit이 없는 초기 작업 상태이며 릴리스 provenance가 확정된 산출물은 없다.

## 서버 선행 조건

LLVY `contracts/desktop-v1/`에는 정책 README만 있고 원본 JSON 스키마·fixture·고정 가능한 계약 commit은 없다.
따라서 작업 1.5, 6.1~6.5의 서버 계약 부분, 7장의 실제 예약·전송·처리 및 9장의 계약/HTTP 통합은 완료하지 않았다.
서버 원본을 임의로 재정의하거나 형제 저장소 소스를 빌드 입력으로 읽지 않는다. 제품 origin도 확정하지 않았다.
다음 연결 작업은 LLVY 소유 서버 계약·API 구현과 실제 Blob PUT 검증 후 시작한다.

## 남은 검증·기능

- Windows에서 최소 exe 실행, 시작 시간·메모리·동적 DLL, 한국어 IME·키보드·스크린리더·100/150/200% DPI·긴 경로.
- 실제 LoL 설정 fixture와 리플레이 파일명, OneDrive hydrated/pinned/online-only/부분 다운로드/오프라인.
- 다른 exe 경로·버전의 두 프로세스 및 crash 후 잠금 복구, Windows CI와 PowerShell 빌드 스크립트 실행.
- 로그인·토큰 교환·refresh/회수·로그인 유지·실제 스트리밍 큐·예약 한도·오류 복구.
- 실제 Discord·Blob·ROFL 통합, 서버 회귀·권한·예산 경쟁 검증과 Windows 10/11 일반 사용자 검증.
- 아이콘·의존성 라이선스 배포 묶음·보호된 release workflow·서명 및 SmartScreen 확인.

작업 1.2, 5.1, 5.4만 완료 표시한다. 나머지는 부분 구현 근거가 있어도 필요한 연결·실행 검증을 마칠 때까지 열린 상태로 유지한다.
활성 change는 정식 명세로 동기화하거나 아카이브하지 않는다.
