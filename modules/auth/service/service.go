package service

import (
	"errors"
	"secure-patrol-backend/models"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("email or password is incorrect")
	ErrAccountLocked      = errors.New("account is temporarily locked because of too many failed login attempts")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrPlatformNotAllowed = errors.New("your role is not allowed to sign in on this platform")
	ErrSessionInvalid     = errors.New("session is invalid or expired, please login again")
	ErrTokenAppMismatch   = errors.New("token was not issued for this app")
	ErrOldPasswordInvalid = errors.New("old password is incorrect")
	ErrSamePassword       = errors.New("new password must be different from the old password")
)

// AccountLockedError is returned while an account is locked after too many
// failed logins. It matches ErrAccountLocked with errors.Is.
type AccountLockedError struct {
	Until time.Time
}

func (e *AccountLockedError) Error() string { return ErrAccountLocked.Error() }
func (e *AccountLockedError) Unwrap() error { return ErrAccountLocked }

// ClientInfo describes where a login request came from.
type ClientInfo struct {
	AppClientID int64
	AppID       string
	AppPlatform string
	UserAgent   string
	IPAddress   string
}

type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	User             models.User
}

// Principal is the authenticated user of a request.
type Principal struct {
	UserID    int64
	RoleCode  string
	SessionID string
}

type AuthService interface {
	Login(email string, password string, client ClientInfo) (TokenPair, error)
	Refresh(refreshToken string, client ClientInfo) (TokenPair, error)
	Logout(sessionID string) error
	LogoutAll(userID int64) error
	// Authenticate validates the access token for a request from the given app client.
	Authenticate(accessToken string, appID string, appPlatform string) (Principal, error)
	Me(userID int64) (models.User, error)
	ChangePassword(userID int64, sessionID string, oldPassword string, newPassword string) error
}
