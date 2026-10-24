## Context

별도 `llvy-uploader` 저장소의 Go Windows 앱 활성 계획이다. 고정 후보 계약 기반 로컬 연동 코드와 Windows 빌드·CI 구성이 있으며 실제 Windows·외부 서비스 실행 검증과 릴리스는 남아 있다. [LLVY 서버 설계](../../../../llvy/openspec/changes/add-desktop-replay-upload-api/design.md)가 기기 프로토콜과 권한·예약·저장 처리를 소유한다.

## Goals / Non-Goals

**Goals:**

- Windows 10/11 x64에서 일반 사용자 권한으로 실행하는 단일 exe. 실제 지원 범위는 양쪽 OS 검증 후 릴리스 문서에 명시한다.
- 알려진 위치 탐색과 수동 선택, 하나/여러 경기 ID의 정확한 파일 매칭, 미리보기 후 순차 업로드.
- 브라우저의 기존 Discord 로그인 재사용과 업로드 전용 기기 권한, 선택적 로그인 유지.
- 기존 웹 URL·쿠키·업로드 DTO·사용자 흐름과 관리자/오너 권한의 호환성 유지.
- 서버가 소유한 직접 Blob 전송 계약, 응답 유실 시 상태 복구, 공유 저장 계층과 계정 예산.
- Mac 단위 테스트, Windows 자동 검증, 실제 PC와 스테이징 검증의 증거를 구분한다.

**Non-Goals:**

- 리플레이 자동 다운로드, Riot 계정 로그인, LCU API/lockfile 인증, 게임 프로세스 접근·주입, 실행 중인 게임 감시.
- 리플레이 재생, 클라이언트에서 ROFL 파싱/통계 계산, 디렉토리 감시 및 백그라운드 자동 업로드.
- macOS/Linux 배포, Windows 32비트/ARM 네이티브 exe, 자동 업데이트 서비스, 설치 프로그램.
- 일반 viewer의 업로드 허용, 기존 웹의 기기 토큰 전환, 서버 비밀키를 담은 exe.
- 운영 배포·DB 변경·인증서 구매를 이번 계획 작성의 일부로 수행하는 것.

서버 API·마이그레이션·권한 저장 구현은 이 change의 범위가 아니다. Worker/QStash를 실행 의존성으로 추가하지 않는다.

## Decisions

### 제한된 로컬 진단 기록

origin별 저장소에 `diagnostics.jsonl`과 이전 세대 `diagnostics.1.jsonl`만 기록한다. 각 파일은 최대 256 KiB이며 새 레코드가 한도를 넘기 전에 이전 세대를 교체한다. UTC 시각, 허용된 이벤트·결과·오류 분류, HTTP 상태 번호만 저장하며 오류 원문, 토큰, URL, 사용자·기기·경기 ID, 파일명과 경로를 받을 필드를 제공하지 않는다. 허용 목록 밖 값은 기록하지 않는다. 기록·회전·삭제·종료는 같은 mutex로 직렬화하고 이미 확보한 origin별 인스턴스 잠금의 수명 안에서만 기록한다. 로컬 기록 삭제는 두 로그를 제거하고 활성 인스턴스 잠금을 보존한다. 기록 실패는 업로드 결과를 바꾸지 않으며 실제 Windows 실행 검증과 구분한다.

### 독립 Go 저장소와 고정 API 계약

저장소 루트에 `go.mod`, `go.sum`, `cmd/llvy-uploader`, `internal/{app,desktop,discovery,matching,replay,platform,ui}`를 두었다. Go 1.27.0과 Windows 바인딩 `golang.org/x/sys` v0.48.0을 고정한다. `desktop` 패키지가 고정된 서버 후보 계약에 따라 인증·API·큐를 구현한다. 서버는 TypeScript를 유지하며 이 저장소에 웹 빌드를 가져오지 않는다.

빌드 시 지정한 HTTPS origin에 로그인·순차 업로드를 연결하고 origin별 저장소와 잠금을 사용한다. 일반 `.rofl`과 hydrated Cloud Files, Windows dedup 파일을 허용하고 name-surrogate 및 알려지지 않은 특수 reparse 파일은 제외한다. 실제 LoL 설정 파일 fixture가 없어 설정 파서는 아직 추가하지 않는다. 고정 `go-winres` v0.3.3은 별도 `tools` 모듈에서 manifest·개발 버전 리소스를 생성한다. 아이콘·서명·보호된 릴리스 workflow는 후속 작업이다.

API·인증 규칙은 LLVY가 소유한다. [계약 정책](../../../contracts/desktop-v1/README.md)에 따라 원본 commit·체크섬을 기록한 계약 사본으로 Go 테스트를 실행한다. 형제 저장소 체크아웃 없이 빌드할 수 있어야 한다. 서버 변경이 필요한 릴리스는 관련 서버 검증 결과와 호환 배포를 확인하고 앱을 발행한다.

### Go 표준 라이브러리와 작은 Win32 UI를 사용한다

Go 표준 라이브러리의 HTTP/TLS/JSON/파일 스트리밍과 필요한 `golang.org/x/sys/windows` 바인딩을 기본으로 한다. CGo, Node.js, WebView2, 별도 UI 런타임 없이 동작하도록 구성한다. Win32 창·기본 컨트롤·폴더 선택·Known Folder·DPAPI·브라우저 실행만 작은 Windows 어댑터에 모은다.

UI thread는 고정하고 파일 탐색/네트워크 작업은 별도 goroutine에서 실행하여 메시지로 UI에 반영한다. 한국어 IME, 키보드 탐색, 스크린리더 기본 이름, 100/150/200% DPI, 긴 경로가 첫 UI 검증 범위다. manifest는 `asInvoker`, DPI 인식과 일반 컨트롤 설정을 포함한다. 아이콘/manifest 리소스 생성 도구는 버전을 고정한 빌드 의존성으로만 사용한다.

초기 검증 작업은 창·한국어 입력·폴더 선택·HTTPS·DPAPI가 포함된 최소 exe를 Windows에서 실행하고 크기, idle 메모리, 시작 시간, DLL 의존성을 기록하는 것이다. 작은 실행 파일 수치를 미리 보장하지 않는다. 기본 컨트롤 직접 구현이 요구를 충족하지 못하면 GUI 대안을 검토하고 산출물을 먼저 수정한다. Rust로 언어를 변경하거나 WebView 기반으로 자동 전환하지 않는다.

화면에는 `리플레이 폴더`, `폴더 선택`, `다시 찾기`, `게임 ID`, `파일 찾기`, `LLVY 로그인`, `로그인 유지`, `업로드`, `취소`, `실패 항목 다시 시도`를 둔다. 파일 목록은 입력 ID, 파일명, 크기, 수정 시각, 상태/오류를 표시하고 완료 결과에는 게임 링크를 제공한다.

같은 Windows 사용자와 제품 origin은 하나의 앱 인스턴스만 실행한다. exe 위치나 버전이 달라도 같은 사용자 저장소의 origin별 lock 파일을 공유 없이 열고 프로세스 수명 동안 핸들을 유지한다. 자격 증명/설정을 읽기 전에 잠금을 확보하고, 두 번째 실행은 `이미 실행 중입니다`를 표시하고 종료한다. 프로세스 종료 시 OS가 잠금을 해제하므로 남은 빈 파일을 실행 중 증거로 취급하지 않는다. 로컬 기록 삭제는 실행 중 lock 파일을 삭제하지 않는다. 개발 origin은 별도 잠금과 자격 증명 저장소를 사용한다.

### 설치 경로와 리플레이 경로를 구분한다

탐색 우선순위는 다음과 같다.

1. 사용자가 저장한 리플레이 경로가 현재 접근 가능하면 유지한다.
2. 알려진 설치 후보에서 실제로 검증한 설정 파일 형식에 리플레이 경로가 있으면 해석한다.
3. `SHGetKnownFolderPath(FOLDERID_Documents)` 결과 아래 `League of Legends/Replays`를 후보로 사용한다. OneDrive/문서 폴더 이동을 고려하여 사용자 이름과 `C:\Users`를 조합하지 않는다.
4. `C:\Riot Games\League of Legends` 및 고정 로컬 드라이브의 같은 상대 위치, Program Files 계열과 검증된 Riot 설치 메타데이터를 제한된 후보 목록으로 조사한다. 설치가 있다는 이유만으로 게임 루트를 리플레이 폴더로 확정하지 않는다. 실제 리플레이 하위 경로나 `.rofl`이 있는 루트만 후보로 제시한다.
5. 후보가 없으면 `리플레이 폴더를 찾지 못했습니다. LoL 클라이언트의 리플레이 저장 위치를 확인한 뒤 폴더를 선택하세요.`를 표시하고 네이티브 폴더 선택을 연다. 취소해도 앱은 계속 사용할 수 있다.

읽기 가능한 후보가 하나면 선택하고 발견 근거를 표시한다. 여러 후보가 있으면 선택을 요청한다. 빈 표준/수동 지정 폴더는 유효하며 검색 결과만 0건으로 표시한다. 설치 흔적이 없더라도 사용자가 복사한 `.rofl` 폴더로 동작한다. 사용자 폴더/드라이브 전체 재귀 검색, 설치 파일 수정, 관리자 권한 요청은 하지 않는다. 설정 파일의 위치/키는 공개 안정 계약으로 가정하지 않고 실제 Windows 파일 fixture를 확인한 어댑터만 지원한다.

### 게임 ID는 경기 번호로 처리하고 파일명 토큰을 정확히 매칭한다

입력은 줄바꿈·쉼표·세미콜론·공백으로 구분한다. 숫자 경기 번호 또는 플랫폼 접두어를 포함한 `KR-1234567890`/`KR_1234567890` 형식을 허용하고 숫자를 부동소수점으로 변환하지 않는다. 영문 접두어는 대문자로 정규화하며 의미 없는 앞자리 0은 양쪽 비교에서 정규화한다. 허용 길이는 숫자 1~20자리, 접두어 영숫자 2~8자리이고 최대 100개의 고유 ID를 받는다. Riot ID인 `이름#태그`에는 경기 번호가 필요하다는 오류를 낸다.

선택 폴더의 직속 일반 `.rofl` 파일만 색인한다. 확장자는 대소문자를 구분하지 않고 symlink/junction 등 다른 경로를 가리키는 name-surrogate reparse point는 따라가지 않는다. Cloud Files의 reparse tag만으로 파일을 제외하지 않는다. 로컬에 전체 바이트가 있는 OneDrive 파일은 지원하며, online-only/부분 다운로드 상태는 `로컬 다운로드 필요`로 표시하고 탐색기에서 `항상 이 장치에 유지` 후 다시 찾도록 안내한다. 발견 단계는 메타데이터만 읽고 클라우드 다운로드를 시작하지 않는다. 상태가 바뀌거나 provider/오프라인 문제로 안정적으로 읽지 못하면 별도 파일 오류를 표시하고 예약 전에 중단한다. 이름이 있는 파일을 이 이유로 `파일 없음` 처리하지 않는다. 파일명에서 검증된 숫자/플랫폼 토큰을 추출하며 `123`이 `1234.rofl`에 매칭되지 않도록 한다. 표준 `KR-<id>.rofl` 형식은 초기 지원하고 다른 이름은 완전한 숫자 토큰 일치로 후보만 만들며 자동으로 플랫폼을 추정하지 않는다. 숫자만 입력해 여러 플랫폼/복사본이 잡히면 사용자에게 한 파일을 선택하게 한다. 접두어를 입력했다면 다른 플랫폼 파일은 제외한다.

0건은 `파일 없음`, 여러 건은 `파일 선택 필요`, 0바이트/64 MiB 초과/권한 오류는 해당 오류로 표시한다. 같은 파일이 여러 입력에서 선택되면 한 번만 큐에 넣는다. 파일명 매칭은 선택 보조이며 경기의 실제 정체성과 중복 여부는 서버 파싱/해시 결과가 결정한다. 파일명 ID와 서버 게임 UUID를 혼동하지 않는다.

업로드 직전에 파일을 다시 열어 크기·수정 시각을 확인하고 Windows 읽기 핸들을 통해 전송 중 수정/삭제와의 경쟁을 제한한다. 공유 위반이나 변경 발견 시 해당 항목을 실패 처리한다. 전체 파일을 메모리에 적재하지 않고 스트리밍한다. 원본 파일은 이동·수정·삭제하지 않는다.

### 브라우저 승인과 로컬 자격 증명

앱은 public client이며 제품 LLVY HTTPS origin은 빌드 설정으로 고정한다. 개발 origin은 별도 개발 빌드에서만 설정한다. 제품 UI에 임의 서버 주소 입력을 두지 않는다. `127.0.0.1:0` listener를 먼저 확보하고 state·PKCE S256을 생성한 뒤 서버가 반환한 내부 승인 URL을 기본 브라우저로 연다. Host/path/state/구조 검증, timeout·취소 시 listener 종료와 일회용 코드 교환을 구현한다.

서버의 `desktop-device-authorization` 공개 계약을 따른다. 계정당 기기 한도·모드 수명·명시적 transient 교체·scope·회수는 서버가 판단한다. 앱은 Discord 토큰과 서버 비밀을 받지 않는다. 프로세스 내 refresh single-flight와 사용자/origin 단일 인스턴스를 함께 지키고 갱신 응답 유실 시 재로그인을 요구한다. `429 rate_limited`는 서버가 토큰 교체 전에 반환하므로 메모리·보관 자격 증명을 유지하고 원래 API 오류와 Retry-After를 전달한다. 상태/코드가 일치하지 않거나 응답이 유실·손상된 경우는 계속 재로그인을 요구한다. 시작 시 Restore 오류 후 즉시 Session을 호출하지 않으며, 저장 토큰만 복구되어 기기 ID가 아직 없어도 다음 로그인 동작은 해당 토큰으로 상태 확인을 재시도한다. 로컬 로그아웃은 서버 회수 실패와 무관하게 로컬 비밀을 지우고 결과를 표시한다.

`로그인 유지`는 기본 꺼짐이다. 꺼진 상태의 정상 종료는 최대 2초 동안 서버 회수를 시도한 후 메모리 비밀을 지우며, 회수 실패를 표시하되 종료를 무한 대기시키지 않는다. crash/오프라인 종료로 남은 연결은 12시간 절대 만료로 제거되며 그 전에 한도를 채우면 다음 브라우저 승인에서 명시적 임시 연결 교체로 복구한다. 선택한 경우 갱신 토큰만 사용자 범위 DPAPI로 암호화해 `%LOCALAPPDATA%/LLVY/Uploader/`의 origin별 저장소에 저장한다. 새 refresh 저장은 단일 인스턴스 잠금 아래 원자적으로 교체하고 실패하면 이전 세대를 재전송하지 않고 재로그인한다. access token/verifier는 메모리에만 둔다. 설정과 bounded 로그도 같은 위치에 저장하며 토큰 원문이나 전체 경로가 서버 진단으로 자동 전송되지 않게 한다. 설치가 필요 없다는 의미의 포터블이며 로그인 정보를 USB와 함께 이동시키는 방식은 지원하지 않는다. 로컬 기록 삭제 기능을 제공한다.

### 전송 계약 준수

LLVY `desktop-replay-upload-api`의 URL/method/headers/expiresAt descriptor를 검증하고 표준 HTTP PUT으로 파일을 Blob에 직접 스트리밍한다. LLVY bearer를 전송 요청에 붙이거나 redirect를 따라가지 않는다. 허용 HTTPS endpoint·헤더 이름과 값, 중복 이름·CR/LF·Authorization/Cookie/Host/framing 금지, 검증된 크기로 Content-Length 설정을 지킨다. 객체 URL과 Control API 전송 URL을 구분하고 SDK 버전·store ID·헤더를 앱에 하드코딩하지 않는다. 실제 필수 allowlist는 서버 측 PUT 검증과 계약 fixture 확정 후 소비한다.

### 순차 큐와 응답 유실 복구

기존 예약의 복구는 원본 파일을 열거나 hash를 다시 계산하기 전에 상태·process로 확인한다. 예약 응답이 유실된 경우에도 저장한 key·메타데이터로 예약을 복구한 뒤 같은 절차를 따른다. 서버가 `object_not_uploaded`로 부재를 확정한 경우에만 원본을 열어 기존 메타데이터와 대조하고 재전송한다.

`실패 항목 다시 시도`는 먼저 기존 예약을 복구한다. 서버가 확정 실패·만료 또는 `outcome_unknown`을 반환한 경우에만 UI 스레드에서 새 예약 확인창을 연다. 새 예약 1회 소비와 중복 가능성, 결과 불명확이면 기존 경기 확인이 필요함을 안내하며 기본 선택은 취소다. 확인 후 기존 결과를 다시 확인하고 최신 한도에 여유가 있을 때 새 key로 선택 파일을 업로드한다. 성공·처리 중으로 바뀌었거나 인증·네트워크 확인이 실패하면 새 예약을 만들지 않는다. 취소·기기 변경도 확인창과 후속 실행을 중단한다.

처리 요청은 60초 서버 실행과 네트워크 여유를 위해 access 잔여 수명 2분을 확보한다. 직렬 refresh 이후에도 기기 절대 만료 때문에 이 여유가 없으면 요청을 보내지 않고 재로그인을 요구한다. 예약 시도 여부를 응답 수신 여부와 별도로 보존한다. 한도 snapshot은 한 번도 예약하지 않은 항목에만 적용하며, 응답 유실 항목은 잔여 0에서도 같은 key·메타데이터로 복구한다. 혼합 선택에서 새 항목이 한도를 초과하면 복구 항목만 실행하고 새 항목은 선택 수정 안내와 함께 보류한다.

아직 예약을 시도하지 않은 파일은 명시적 재검색·선택 후 최신 메타데이터로 교체할 수 있다. 예약을 시도한 항목은 파일이 달라져도 key와 원래 메타데이터를 유지한다. 단, 서버 완료·중복 결과가 확정된 파일은 변경된 후보를 명시적으로 업로드할 때 새 항목으로 교체한다. 취소 안내는 이미 완료된 로그아웃·로컬 삭제·세션 상태 변경을 덮어쓰지 않는다.

동시 파일 업로드는 1개다. 입력 100개 허용은 일일 예약 한도 증가를 의미하지 않는다. 사용자가 선택 목록을 확인하고 `업로드`를 눌러야 예약/전송을 시작한다. 상태는 `대기 → 예약 → 전송 → 처리 → 완료/이미 등록됨/실패`이며 별도로 `취소됨`, `결과 확인 필요`를 둔다.

`GET /api/desktop/v1/session`의 한도 snapshot은 `asOf`, account/global 각각의 `limit`, `remaining`, `resetAt`를 제공한다. 만료된 counter는 현재 구간의 잔여량으로 해석하며 아직 활성 구간이 없으면 resetAt은 null이다. UI는 `선택한 파일 N개`와 `현재 24시간 구간의 남은 예약 M회`를 업로드 전에 나란히 표시하고 M은 account/global 잔여량 중 작은 값으로 계산한다. 자정 초기화를 뜻하는 `오늘`은 쓰지 않는다. 중복·실패도 이미 생성한 예약 1회를 소비하며 환급하지 않음을 안내한다. 선택 수가 M보다 크면 일부만 가능하다는 경고와 선택 수정 기회를 제공한다. snapshot은 실행 권한/슬롯 예약이 아니므로 다른 웹·앱 요청으로 달라질 수 있으며 각 예약과 429가 최종 판단이고 이후 snapshot을 갱신한다. 클라이언트 hash만으로 서버 중복을 확정해 예약/바이트 검증을 건너뛰는 기능은 MVP에서 제외한다.

네트워크 오류 시 reservation/조회처럼 안전한 작업만 최대 3회 jitter backoff 재시도한다. PUT 응답 유실은 같은 경로의 overwrite를 시도하지 않고 처리 API가 정확한 객체 존재/바이트를 확인하도록 한다. 객체가 없다고 확인된 경우에만 같은 예약으로 재전송한다. access 만료는 한 번의 직렬 갱신 후 동일 작업을 재시도하고, refresh 응답 유실·403은 큐를 멈춘다. 429는 Retry-After/reset 정보를 표시하고 일일 예산 소진이면 전체 대기를 일시 중지한다. 단일 파일 오류는 다른 항목을 계속 진행할 수 있다.

취소는 이후 큐와 현재 HTTP 전송을 중단한다. 이미 시작된 서버 저장을 취소했다고 단정하거나 성공 데이터를 삭제하지 않고 상태를 확인한다. 앱 종료 후 자동 큐 복원은 MVP에 포함하지 않으며, 재실행 후 같은 파일 업로드는 서버 중복 방지를 따른다.

예약 응답 유실은 같은 key·메타데이터/hash로 복구하고 새 예약으로 숨기지 않는다. PUT 응답을 받지 못해도 uploadId/nonce로 서버에 존재 확인·처리를 요청한다. 확정 부재일 때만 재전송하며 timeout/5xx를 부재로 해석하지 않는다. process 응답 유실은 상태 조회로 복구한다. 결과는 예약 생성부터 7일, 멱등 tombstone은 기기 절대 만료까지라는 서버 계약을 따른다. 8일차 key 재사용의 `reservation_expired`/`idempotency_conflict`를 검증한다. 처리 기한 후 outcome_unknown은 자동 재수행하지 않고 한도 소비 안내와 명시적 재시도를 제공한다.

### 검증과 릴리스는 Windows 실행 증거를 포함한다

- 로컬 Mac(선택): 순수 Go 패키지 `go test`, 지원 호스트의 race 검사, `go vet`, matcher/큐/HTTP contract fixture 테스트. `GOOS=windows GOARCH=amd64 CGO_ENABLED=0` 교차 빌드는 컴파일 확인으로만 기록한다.
- Windows CI: 고정 `windows-2025` x64 runner, 고정 Go 버전과 action commit, `go test -race -timeout 120s ./...`(CGo/GCC), `go vet ./...`, gofmt, Windows 전용 discovery/DPAPI/loopback/파일 잠금 및 두 프로세스 단일 인스턴스 테스트. 실제 exe 빌드와 명시적 self-test는 release.published workflow에서 수행한다. self-test는 UI/자동 로그인/운영 업로드를 실행하지 않는다.
- 결과물: `-trimpath`, GUI subsystem, 버전/commit resource를 갖는 exe, SHA-256 checksum, 의존성/라이선스 목록과 빌드 provenance. exe 파일만으로 실행되는지 DLL import를 확인한다. UPX 압축은 사용하지 않는다. 바이너리는 Git에 커밋하지 않는다.
- 통합: mock API의 실패/응답 유실/429/갱신 동시성, TypeScript 계약 fixture, 기존 웹 회귀 및 `npm run check`; 인증·예약 회수 경쟁은 disposable localhost PostgreSQL로 검증한다.
- 스테이징: 실제 Discord 승인, Go→Blob 전송, 실제 ROFL 파싱/중복, 웹·앱 공유 한도, 기기 회수와 처리 경쟁을 검증한다. 실제 리플레이는 허용된 로컬/보호 fixture로 제공하고 공개 CI artifact에 올리지 않는다.
- 실제 Windows 10/11 PC: 일반 사용자 권한, 한글 사용자 경로/IME, OneDrive 문서 위치 및 hydrated/pinned/online-only/오프라인 파일, LoL 기본/변경 경로, DPI, 네트워크 끊김과 포터블 실행. GitHub Windows Server runner의 관리자 실행 성공은 이 증거를 대체하지 않는다.

PR/push/수동 일반 검사는 테스트만 실행한다. 버전 Release의 published 이벤트에서만 Linux 검사를 통과한 동일 commit의 exe를 빌드해 GitHub Release 첨부 파일로 게시한다. 지원 태그는 `vX.Y.Z`와 `uploader-vX.Y.Z`이며 릴리스 게시 권한/태그 보호는 저장소에서 관리한다. PR/fork 작업에는 운영 자격 증명이나 서명 키를 노출하지 않는다. 코드 서명 인증서가 준비되지 않았으면 미서명 파일과 SmartScreen 확인 결과를 명시하며 경고 제거를 보장하지 않는다. 배포 승인은 구매/서명 인프라 도입과 구분한다.

## Risks / Trade-offs

- [Win32 직접 UI의 구현 부담] → 화면을 하나로 제한하고 초기 spike로 한국어/DPI/접근성/스레드 모델을 먼저 확인한다.
- [LoL 설정 파일과 설치 흔적의 변화] → 문서 Known Folder와 수동 선택을 기본 탈출 경로로 유지하고 확인한 형식만 읽는다.
- [Blob 서명 PUT의 실제 store 호환성] → 현재 SDK 타입은 확인했지만 배포 기능을 가정하지 않고 스테이징 계약 검증을 선행한다.
- [인증 권한 확대 또는 기존 웹 회귀] → 별도 namespace/kind/scope를 사용하고 기존 URL/DTO/로그인에 영구 회귀 테스트를 둔다.
- [회수 후 이미 발급한 Blob URL] → 발급 후 최대 5분 동안 바이트 전송은 가능할 수 있다. 새 승인과 최종 저장은 즉시 현재 DB 권한으로 차단하고 잔여 객체는 기존 정리 작업에 맡긴다.
- [큰 큐와 작은 현재 한도] → 계정별 10회·전체 50회/24시간을 유지하며 명확한 중단/잔여 안내를 제공한다. 배치가 한도를 우회하지 않는다.
- [기기 토큰 회전 응답 유실] → 자동 재시도 대신 재로그인으로 복구해 재사용 감지 정책을 단순하고 검증 가능하게 유지한다.
- [일반 Windows와 CI 환경 차이] → runner는 빌드/자동 테스트 증거로만 사용하고 사용자 권한·실제 LoL·DPI는 별도 검증한다.
- [진행 중인 다른 OpenSpec changes] → 캐시 등 무관한 변경은 건드리지 않는다. `harden-security-scan-findings`의 미완료 외부 검증과 업로드 소유권 요구는 함께 재확인하되 이 change의 성공으로 완료 처리하지 않는다.

독립 저장소의 계약 갱신은 서버 계약 원본과 고정 사본의 명시적 검토로 관리한다. 양쪽 저장소를 한 번에 체크아웃해야 하는 CI 경로 필터 대신 계약 버전·호환 서버 검증 증거로 릴리스를 연결한다.

## Migration Plan

1. 최소 Win32 exe와 실제 Blob 전송 검증에 참여하고 원본 계약을 고정한다.
2. 로컬 파일·인증·큐와 Windows CI를 구현한다.
3. Go/Windows 자동 검사와 별도 LLVY 서버 검사 결과를 기록한다.
4. 실제 Windows 10/11·OneDrive·스테이징 전체 흐름을 검증한다.
5. 호환 서버가 먼저 배포된 뒤 승인된 범위에서 고정 commit의 exe를 릴리스한다.
6. 문제 발생 시 새 앱 배포를 중단하고 서버의 desktop 비활성화·회수 절차와 연계한다. 앱에서 서버 데이터나 원본 파일을 삭제하지 않는다.
7. 모든 앱 구현·필수 검증 후에만 이 기능 change를 아카이브한다.

## Open Questions

다음은 구현 초기에 증거로 확정할 항목이며, 현재 구현 완료를 의미하지 않는다.

- 실제 Windows LoL 설정의 replay 경로 키/파일 형식과 지역별 파일명 사례: 기본 후보와 수동 선택은 명세대로 구현하고 검증된 추가 형식만 확장한다.
- 실제 대상 Blob store에서 현재 SDK 서명 PUT의 동작과 응답: private/public 및 제한 enforcement를 staging에서 확인해야 한다.
- 배포 파일의 최종 크기/메모리/시작 시간과 Win32 resource 빌드 도구: 최소 exe 결과로 결정한다.
- Windows 코드 서명 인증서/신뢰된 서명 서비스의 제공 여부: 미서명 배포의 한계를 문서화하며 자동 구매하지 않는다.
- 실제 Windows 10/11 및 스테이징에서 필요한 검증 접근성: 확보되지 않으면 해당 작업과 change를 열린 상태로 유지한다.

## References

- [Riot Replay FAQ](https://support.riotgames.com/en-us/league-of-legends/gameplay/replays-faq-pro-tips/): 클라이언트에서 리플레이 저장 위치를 바꿀 수 있음. 특정 로컬 설정 파일 형식의 근거로 사용하지 않는다.
- [Native Apps OAuth / RFC 8252](https://www.rfc-editor.org/rfc/rfc8252.html): 외부 브라우저, loopback 및 PKCE.
- [Windows DPAPI](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata): 사용자 범위 로컬 자격 증명 보호.
- [Windows Cloud Files 호환성](https://learn.microsoft.com/en-us/windows/win32/cfapi/build-a-cloud-file-sync-engine) 및 [Reparse tags](https://learn.microsoft.com/en-us/windows/win32/fileio/reparse-point-tags): cloud placeholder와 name-surrogate를 구분하며 실제 사용 API의 노출/읽기 동작은 Windows 검증으로 확인한다.
- [Go Windows cross-compiling](https://go.dev/wiki/WindowsCrossCompiling): 크로스 빌드와 CGo 제약.
- [GitHub runner specifications](https://docs.github.com/en/actions/reference/runners/github-hosted-runners): `windows-2025` x64와 기본 관리자 실행 환경.
- [Vercel client uploads](https://vercel.com/docs/vercel-blob/client-upload): 함수 본문을 경유하지 않는 직접 전송. 서명 API의 구체적인 현 버전 계약은 설치된 `@vercel/blob` 2.5.0의 `dist/index.d.ts`와 `dist/create-folder-DFjrvss1.d.ts`에서 확인했으며 실제 전송 검증은 미수행이다.

외부 자료·SDK 관찰은 분리 전 계획의 근거이며 구현 전 다시 확인한다. SDK 소스는 LLVY가 소유하고 이 앱의 로컬 의존성으로 가정하지 않는다.

## 서버 연동 구현 (2026-10-18)

초기 미리보기에 Go 인증·전송·결과 복구를 추가했다. 제품 origin은 기본값 없이 빌드 시 주입하며 제품 UI에는 URL 입력이 없다. `development` build tag만 명시적으로 지정한 loopback HTTP origin도 허용한다. 계약은 LLVY의 후보 JSON 파일을 실제 원본 commit·SHA-256으로 고정하며 endpoint/헤더 정책을 embed한다. 초기 preview namespace 대신 제품 origin을 잠금과 자격 증명 namespace로 쓴다.

큐와 uploadId/nonce·멱등 키는 현재 프로세스의 메모리에 보관하며 종료 후 자동 재개하지 않는다. 재시도는 같은 entry를 유지하고 완료·중복 행의 더블 클릭은 서버가 검증한 game UUID 경로를 연다. 실제 Blob·Windows·Discord 검증과 릴리스는 남아 있으며 초기 미리보기 설명보다 이 구현 상태를 우선한다.

## 기기 세션 전환과 취소 효과

### 완료한 경로의 새 파일과 한도 안내

완료·중복 결과와 같은 경로라도 재검색에서 이름·크기·수정 시각이 달라지면 이전 결과를 새 파일의 성공으로 표시하지 않는다. 변경된 후보를 표시하며 사용자가 `업로드`를 눌렀을 때 새 예약 후보를 만든다. 검색·화면 갱신 자체는 예약이나 key를 만들지 않는다. 미확정·실패 예약은 기존 key와 메타데이터를 보존하며 기존 복구·새 시도 확인 절차를 따른다. 메타데이터가 같은 파일의 내용 변경은 탐색 단계에서 판단하지 않는다.

업로드 전 한도 요약에 계정·전체 reset 시각을 각각 표시하며 null은 아직 시작되지 않은 구간으로 설명한다. 한도가 없어 새 시도를 보류할 때도 같은 시각을 안내한다. 서버의 429 Retry-After는 올림한 초 단위 대기 시간으로 표시하며 자동 새 예약이나 대기 후 재실행을 시작하지 않는다.

예약 시도 전에 생성 기기 session ID를 Entry에 기록하고 다른 기기 항목은 네트워크 전에 거부한다. 작업 종료 시 UI는 실제 로컬 연결 ID와 서버 snapshot을 구분하여 동기화한다. 이전 기기 작업이 남은 채 연결이 해제·변경되면 큐와 선택을 비우고 이전 서버 결과 확인과 새 예약 한도 소비를 안내한다. 같은 기기의 refresh·일시적 조회 실패는 예약 정체성을 보존한다.

토큰 교환 후 session 조회가 실패·취소되면 연결 정보가 남아 있음을 표시하고 로그인 버튼은 같은 연결의 조회만 재시도한다. stale snapshot이 삭제된 인증을 되살리지 않게 한다. 취소된 자동 탐색·파일 검색의 선택 교체와 picker 후속 동작은 버리되 완료된 로컬 삭제·로그아웃·수동 폴더 저장과 연결 상태는 보존한다.

### 계약 사본의 체크아웃 바이트 보존

고정 계약 JSON은 원본 SHA-256과 바이트 단위로 일치해야 한다. `.gitattributes`에서 `contracts/desktop-v1/*.json text eol=lf`를 지정해 Windows의 `core.autocrlf=true`에서도 LF를 유지한다. 체크섬 검사는 원본 바이트 그대로 수행하며 검사 시 줄바꿈을 정규화하지 않는다.

### 빌드 주소와 다운로드 아티팩트

GitHub Actions repository variable `LLVY_SERVER_ORIGIN`을 PowerShell 환경변수로 전달하고 Go linker의 `desktop.ServerOrigin`에 주입한다. 주소는 HTTPS origin만 허용하며 사용자 정보·경로·query·fragment·공백을 거부한다. 기본값과 실행 시 환경변수 fallback은 없다. 누락된 배포 빌드는 실패하고, CI의 명시적 `-AllowUnconfigured` 빌드는 self-test만 가능하며 배포하지 않는다. 개발 tag의 loopback HTTP도 기본값 없이 지정한다. 고정 서버 계약의 `productOrigin`은 해당 원본 revision의 배포 메타데이터로 보존하고 앱의 주소 기본값이나 허용 목록으로 사용하지 않는다. API·Blob endpoint/헤더 계약과 체크섬은 그대로 유지한다.

`Publish Windows executable`는 `release.published`에서만 실행한다. `vX.Y.Z`와 `uploader-vX.Y.Z` 및 SemVer 사전 버전 접미사를 검증하고 해당 tag를 체크아웃한다. Windows 리소스 제약에 따라 각 숫자는 0–65535이며 build metadata(`+...`)는 받지 않는다. 정규화한 버전은 exe 파일명·앱/PE 버전에 쓰고 commit은 provenance에 보존한다.

클라우드 CI는 `ubuntu-24.04`의 공통 tests/race·vet·gofmt·PowerShell 스크립트 검사를 먼저 통과해야 Windows tests/race·vet를 실행한다. push·PR·수동 검사에는 배포 exe 빌드가 없다. 릴리스 workflow만 CGo 없는 exe 빌드·PE 검사·self-test·내장 주소 검사를 실행한다. macOS job은 없으며 로컬 Mac 검사는 개발 편의용이다.

검증한 exe·체크섬·빌드/출처·의존성 고지·검증 근거를 GitHub CLI로 해당 Release에 직접 첨부한다. 배포 job에만 `contents: write`를 부여하고 토큰은 게시 단계에서만 전달한다. 태그 버전, 실제 Git tag commit, Release 채널을 검증한 후 업로드한다. 기존 자산은 덮어쓰지 않으며 재실행 시 같은 바이트만 재사용하고 다른 바이트가 있으면 실패한다. 첨부 파일을 다시 내려받아 모든 SHA-256을 비교한다. 사용자는 브라우저에서 exe를 바로 받으며 Docker·ORAS를 설치하지 않는다. GHCR 게시와 packages 쓰기 권한은 제거한다.

### 릴리스 근거와 라이선스 묶음

일반 빌드는 독립적으로 유지하며 게시 workflow는 저장소 변수 `LLVY_RELEASE_EVIDENCE_JSON`에 등록한 검토된 근거를 검사한다. 정확한 앱 commit·버전·origin·계약 source commit/해시와 검증된 서버 commit·배포 URL, 서버 check/strict/PostgreSQL 경쟁, 실제 Windows 10/11·Discord·Blob·리플레이/기존 웹·회수 검증의 통과 기록 및 HTTPS 근거 링크가 모두 필요하다. 누락·불일치·미완료는 게시 전에 실패한다. 근거 문서 자체는 검토자의 진술이며 원격 검증 실행을 대신하지 않는다. 로컬 빌드는 이 자료 없이 가능하지만 게시 명령은 재검사한다. 근거는 산출물에 포함한다.

Go와 런타임/빌드 도구의 고정 버전별 라이선스 원문, 계약 원본 권리 조건을 `THIRD_PARTY_NOTICES.txt`에 보존하고 빌드·게시 묶음에 포함한다. 자체 라이선스 선택은 별도이며 외부 코드 권리 조건을 덮어쓰지 않는다.

### Actions 실행 런타임

2026-10-19 공식 안정 릴리스 기준 checkout v7.0.1, setup-go v7.0.0의 검증된 commit SHA를 고정하며 모두 Node.js 24를 사용한다. 별도 Node.js 20 기반 action을 추가하지 않는다. 실제 버전 릴리스 발행·Release 첨부 실행과 Windows 일반 사용자·외부 서비스 수동 검증은 수행하기 전까지 미완료다.

### 업로더 역할 확장

앱 session 파서는 uploader/admin/owner를 허용하고 viewer/unknown은 거부한다. 업로더 권한의 생성·회수는 서버 관리자/오너만 수행한다. UI는 session의 계정명을 그대로 표시하며 역할을 admin으로 바꾸지 않는다. 원본 계약 commit 고정 → 앱 사본/hash 갱신 및 검사 → 호환 서버/앱 배포 → 실제 Windows 검증 → uploader 권한 활성화 순서를 따른다. 계약 원본은 로컬 `fe32e8a93a426b0446338d24c201c5b778dab682`로 고정했다. 원격 공개와 실제 검증 전에는 릴리스 호환 완료로 기록하지 않는다.

### 테스트용 사전 릴리스 채널

GitHub Release의 prerelease boolean에서 채널을 도출하며 버전 접미사와 일치해야 한다. `RELEASE_CHANNEL=prerelease`와 근거의 `channel=prerelease`가 모두 일치하는 사전 버전에서만 Windows 10/11·Discord·Blob·리플레이/웹 회귀·회수 항목을 pending으로 허용한다. 각 pending 항목도 검증 한계를 설명한 HTTPS 문서를 가리킨다. 서버 check·strict·PostgreSQL 경쟁은 passed가 필수이며 자동 Linux/Windows tests/race/vet, PE·self-test·내장 origin 검사는 모든 채널에 유지한다. 다른 채널·누락·failed·commit/계약/version/origin 불일치는 게시를 거부한다.

정식 채널의 기본값과 모든 실환경 검증 passed 요구는 유지한다. 사전 릴리스 패키지의 provenance·실행 요약과 공개 Release에는 미서명·실환경 검증 미완료 상태를 표시한다. 같은 검증 근거를 빌드와 게시 직전에 재검사하고 Release 첨부를 다운로드해 전체 파일 해시를 비교한다. 사전 릴리스는 실제 PC 검증이나 아카이브 조건을 완료하지 않는다.
