# 업로더의 데스크톱 API 계약

상태: LLVY가 소유하는 desktop-v1 **후보 계약**의 고정 사본이다. 실제 Blob public/private 전송과 Discord·Windows 검증 전이며 릴리스 호환성을 확정하지 않았다.

- [원본 revision·파일별 SHA-256](source.json)
- [프로토콜·경로·전송 헤더 정책](protocol.json)
- [JSON 스키마](schema.json)
- [비밀 없는 fixture](fixtures.json)
- [서버 원본 정책](../../../llvy/contracts/desktop-v1/README.md)

원본 계약 commit은 `f6c218f8d9b0476623b73dc75579722c8872d479`다. 이 commit은 계약 파일만 포함하며 서버 기능 구현의 배포 commit이 아니다. 앱과 서버 구현은 별도로 검증한다.

`go test ./internal/desktop`은 이 저장소의 사본·체크섬·fixture와 Go DTO, 전송 검증기를 대조한다. `.gitattributes`는 계약 JSON의 체크아웃 줄바꿈을 LF로 고정해 Windows에서도 원본 SHA-256을 보존한다. 일반 빌드·검사는 형제 소스나 최신 파일 다운로드를 사용하지 않는다. 계약 변경은 LLVY 원본에서 시작하며 앱 사본만 임의로 바꾸지 않는다.

제품 origin은 `https://llvy.my`다. session 프로토콜이 다르거나 전송 endpoint/헤더가 계약을 벗어나면 중단한다. 서버 변경이 필요한 릴리스는 호환 서버 배포와 실제 검증을 먼저 완료해야 한다.

복사한 계약 파일의 출처·권리 조건은 [LLVY 라이선스](https://github.com/kgskr/llvy/blob/f6c218f8d9b0476623b73dc75579722c8872d479/LICENSE)를 따른다. 업로더 자체 코드의 공개 라이선스는 별도 결정 사항이다. 원격 저장소는 [kgskr/llvy-uploader](https://github.com/kgskr/llvy-uploader)다.
