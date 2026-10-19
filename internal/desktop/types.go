// Package desktop implements the server-owned desktop-v1 candidate contract.
package desktop

import (
	"errors"
	"time"
)

const ClientID = "llvy-windows-uploader"
const Protocol = "desktop-v1"

var ErrLogin = errors.New("다시 로그인하세요")
var ErrNetwork = errors.New("서버 응답을 확인하지 못했습니다")
var ErrContract = errors.New("지원하지 않는 서버 응답입니다")
var ErrDeviceChanged = errors.New("이전 연결의 예약입니다. 서버 결과를 확인한 뒤 파일을 다시 선택하세요. 새 업로드는 예약 한도를 사용합니다")

type APIError struct {
	Code       string
	Status     int
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	switch e.Code {
	case "quota_exhausted":
		return "예약 한도 소진 — 한도 갱신 시각을 확인하세요"
	case "rate_limited":
		return "요청이 많습니다. 잠시 후 다시 시도하세요"
	case "desktop_disabled":
		return "서버의 업로더 기능이 아직 활성화되지 않았습니다"
	case "authority_revoked", "forbidden":
		return "연결 권한이 회수되었습니다. 업로드를 중단합니다"
	case "device_limit":
		return "연결 한도 5개 — 브라우저에서 기존 임시 연결을 선택해 교체하세요"
	case "reservation_expired":
		return "예약 만료 — 이 예약은 다시 사용할 수 없습니다"
	case "invalid_token", "invalid_grant", "refresh_reused":
		return "다시 로그인하세요"
	case "outcome_unknown":
		return "결과 불명확 — 기존 예약의 상태를 확인하세요"
	case "object_not_uploaded":
		return "서버에서 전송 파일을 찾지 못했습니다"
	default:
		return "서버 오류: " + e.Code
	}
}

type Credentials struct {
	Protocol        string    `json:"protocol"`
	TokenType       string    `json:"tokenType"`
	AccessToken     string    `json:"accessToken"`
	RefreshToken    string    `json:"refreshToken"`
	AccessExpiresAt time.Time `json:"accessExpiresAt"`
	DeviceExpiresAt time.Time `json:"deviceExpiresAt"`
	DeviceSessionID string    `json:"deviceSessionId"`
	Scope           string    `json:"scope"`
}
type Quota struct {
	Limit     int        `json:"limit"`
	Remaining int        `json:"remaining"`
	ResetAt   *time.Time `json:"resetAt"`
}
type Session struct {
	Protocol string `json:"protocol"`
	Account  struct {
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"account"`
	DeviceSessionID string    `json:"deviceSessionId"`
	DeviceExpiresAt time.Time `json:"deviceExpiresAt"`
	MaxUploadBytes  int64     `json:"maxUploadBytes"`
	Quota           struct {
		AsOf    time.Time `json:"asOf"`
		Account Quota     `json:"account"`
		Global  Quota     `json:"global"`
	} `json:"quota"`
}
type Metadata struct {
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	LastModified int64  `json:"lastModified"`
	SHA256       string `json:"clientSha256"`
}
type Reservation struct {
	UploadID  string    `json:"uploadId"`
	Nonce     string    `json:"nonce"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type Transfer struct {
	UploadURL string    `json:"uploadUrl"`
	Method    string    `json:"method"`
	Headers   HeaderMap `json:"headers"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type Result struct {
	UploadID  string `json:"uploadId"`
	State     string `json:"state"`
	GameID    string `json:"gameId"`
	Duplicate bool   `json:"duplicate"`
	GamePath  string `json:"gamePath"`
	ErrorCode string `json:"errorCode"`
}

// Vault persists only a refresh token. UI owns it after taking the origin instance lock.
type Vault interface {
	LoadRefresh() (string, error)
	SaveRefresh(string) error
	DeleteRefresh() error
}
