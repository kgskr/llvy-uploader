## Why

Windows에서 내려받은 LoL 리플레이를 게임 ID로 찾아 여러 개 업로드하려면 현재는 사용자가 파일을 직접 찾아 웹에서 하나씩 선택해야 한다. 기존 웹 업로드를 유지하면서, 별도 설치 없이 실행하는 Go 기반 포터블 클라이언트로 파일 탐색과 일괄 업로드를 제공한다.

## What Changes

- 같은 저장소의 `clients/windows-uploader/`에 독립 Go 모듈을 추가한다. Windows x64용 단일 exe와 Win32 기본 UI를 사용하고, 웹과 빌드·버전·배포를 분리한다.
- 시작 시 저장한 경로, Windows 문서 Known Folder, 알려진 LoL 설치 위치와 검증 가능한 로컬 설정을 통해 리플레이 저장 폴더를 탐색한다. 설치 폴더와 리플레이 폴더를 구분하고, 찾지 못하면 한국어 안내와 폴더 선택을 제공한다.
- 게임 ID는 Riot ID가 아닌 경기 번호로 정의한다. 하나 또는 여러 경기 번호를 입력받아 `.rofl` 파일명의 완전한 ID 토큰을 대조하고, 누락·중복 후보를 표시한 뒤 선택한 파일을 순차 업로드한다.
- 기본 브라우저의 기존 Discord 로그인과 명시적 연결 승인을 이용해 앱을 인증한다. LLVY가 일회용 코드와 PKCE S256으로 업로드 전용 기기 세션을 발급하고, 선택적 로그인 유지 정보는 Windows 사용자 범위 DPAPI로 보호한다.
- `/api/desktop/v1/`에 별도 인증·예약·전송 승인·처리·상태 조회 계약을 추가한다. Go는 서버가 발급한 제한된 Blob 서명 URL로 직접 전송하고, 기존 검증·파싱·중복 처리·권한 잠금·감사·정리 로직을 공유한다.
- 기존 `/upload`, Discord 콜백, 세션 쿠키와 `/api/uploads`, `/api/blob/upload`, `/api/process`의 외부 계약은 유지한다. 새 경로도 현재 관리자/오너 권한과 계정별 10회·전체 50회/24시간 예약 한도를 공유한다.
- ARM Mac에서 공통 로직을 개발·테스트하고, GitHub 호스팅 `windows-2025` x64 러너에서 Windows 연동 테스트와 배포 exe를 빌드한다. 실제 Windows PC 및 외부 서비스 검증은 별도 완료 조건으로 관리한다.

## Capabilities

### New Capabilities

- `windows-replay-uploader`: Go 포터블 UI, 리플레이 폴더 탐색, 경기 ID 매칭, 파일 선택과 순차 업로드 상태 관리.
- `desktop-device-authorization`: 기존 브라우저 로그인을 이용한 PKCE 연결 승인, 제한된 기기 토큰, 갱신·회수와 Windows 자격 증명 보관.
- `desktop-replay-upload-api`: 버전이 있는 데스크톱 API, 직접 Blob 전송, 예약 멱등성, 처리 결과 조회와 기존 저장 계층 연동.
- `windows-uploader-release`: 독립 Go 빌드, Windows x64 CI, 포터블 산출물과 릴리스·호환성 검증.

### Modified Capabilities

- `replay-upload`: 기존 웹 계약을 보존하면서 승인된 데스크톱 세션과 파일 시스템 수정 시각을 추가 입력 경로로 허용한다.
- `discord-session-authorization`: 브라우저 세션과 기기 세션을 구분하고 기존 권한 회수·탈퇴 처리·트랜잭션 잠금이 기기 세션에도 적용되게 한다.
- `account-audit-upload-binding`: 데스크톱 예약의 기기 바인딩과 감사 식별을 추가하고, 웹·앱 사이의 계정 예산 및 권한 회수 일관성을 명시한다.

## Impact

- 클라이언트: `clients/windows-uploader/`, Go 표준 라이브러리와 필요한 Windows 바인딩. Node.js, WebView2, 별도 설치형 런타임을 클라이언트 실행 의존성으로 추가하지 않는다.
- 서버: 새로운 연결/기기 관리 화면과 Route Handlers, 명시적인 Proxy 분기, 인증 저장소·권한·업로드 서비스의 재사용 가능한 경계. 필요한 Next.js API는 설치된 `node_modules/next/dist/docs/`를 따른다.
- 데이터: 기기 세션/자격 증명, 짧은 수명의 연결 요청/인증 코드, 데스크톱 업로드 작업·멱등 키·결과 저장을 위한 추가 마이그레이션. 기존 세션과 예약은 브라우저 유형으로 유지한다.
- 저장소/운영: Windows 전용 GitHub Actions, GitHub Releases, 웹 빌드·배포 경로 필터, 기기 연결 및 장애 대응 문서. Vercel Blob SDK의 서명 PUT 기능은 구현 초기 검증 과제로 둔다.
- 이 문서의 범위는 구현 가능한 계획 산출물 작성까지다. 앱/서버 구현, 배포, 운영 DB 변경은 수행하지 않으며 구현·필수 검증이 끝나기 전에는 change를 아카이브하지 않는다.
