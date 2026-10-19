# 업로더 OpenSpec

[활성 계획](changes/add-go-windows-replay-uploader/proposal.md)은 Go 앱과 Windows 배포 범위다.
서버 후보 계약에 연결한 Go 인증·전송과 Windows 어댑터·빌드 구성을 구현했다. 실제 Windows·Blob·Discord 검증이 남아 있어 정식 기능 명세로 동기화하지 않았다.
LLVY의 서버 구현 계획과 계약은 [계약 안내](../contracts/desktop-v1/README.md)에서 찾는다.

저장소 루트에서 `openspec list --json`, `openspec status --change add-go-windows-replay-uploader --json`,
`openspec instructions apply --change add-go-windows-replay-uploader --json`으로 범위를 확인한다.
산출물 준비 상태가 `ready`여도 구현·외부 검증 완료를 뜻하지 않는다.
