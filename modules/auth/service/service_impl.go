package service

import (
	"crypto/subtle"
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auth/repository"
	settingservice "secure-patrol-backend/modules/setting/service"
	"strings"
	"time"

	"gorm.io/gorm"
)

type service struct {
	repo repository.AuthRepository
}

func NewAuthService(repo repository.AuthRepository) AuthService {
	return &service{repo: repo}
}

func (s *service) Login(email string, password string, client ClientInfo) (TokenPair, error) {
	now := time.Now()

	user, err := s.repo.FindUserByEmail(strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		helper.CheckDummyPassword(password)
		return TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return TokenPair{}, err
	}

	if user.IsLocked(now) {
		return TokenPair{}, &AccountLockedError{Until: *user.LockedUntil}
	}

	if !helper.CheckPassword(user.PasswordHash, password) {
		failedCount := user.FailedLoginCount + 1
		var lockedUntil *time.Time
		settings := settingservice.ForUnit(user.UnitID)
		if failedCount >= settings.LoginMaxFailedAttempts {
			until := now.Add(settings.LoginLockDuration())
			lockedUntil = &until
			failedCount = 0
		}

		if err := s.repo.RecordLoginFailure(user.ID, failedCount, lockedUntil); err != nil {
			return TokenPair{}, err
		}

		if lockedUntil != nil {
			return TokenPair{}, &AccountLockedError{Until: *lockedUntil}
		}
		return TokenPair{}, ErrInvalidCredentials
	}

	// Checked after the password so inactive accounts cannot be enumerated.
	if !user.IsActive || !user.Role.IsActive {
		return TokenPair{}, ErrAccountInactive
	}
	if !UnitIsUsable(user) {
		return TokenPair{}, ErrUnitInactive
	}
	if err := checkLicense(user); err != nil {
		return TokenPair{}, err
	}

	// Checked after the password, so the answer does not reveal the role of an account.
	if !models.RoleCanUsePlatform(user.Role.Code, client.AppPlatform) {
		return TokenPair{}, ErrPlatformNotAllowed
	}

	if err := s.repo.RecordLoginSuccess(user.ID, now); err != nil {
		return TokenPair{}, err
	}
	user.LastLoginAt = &now

	sessionID, err := helper.RandomHex(16)
	if err != nil {
		return TokenPair{}, err
	}

	refreshSecret, err := helper.RandomURLToken(32)
	if err != nil {
		return TokenPair{}, err
	}

	session := models.UserSession{
		ID:               sessionID,
		UserID:           user.ID,
		AppClientID:      client.AppClientID,
		RefreshTokenHash: helper.SHA256Hex([]byte(refreshSecret)),
		UserAgent:        truncate(client.UserAgent, 255),
		IPAddress:        truncate(client.IPAddress, 45),
		ExpiresAt:        now.Add(settingservice.ForUnit(user.UnitID).RefreshTokenTTL()),
		LastUsedAt:       now,
	}

	if err := s.repo.CreateSession(&session); err != nil {
		return TokenPair{}, err
	}

	return s.issueTokens(user, session, refreshSecret, client.AppID, now)
}

// Refresh exchanges a refresh token for a new token pair. The refresh token is
// rotated on every use; presenting an old one revokes the whole session because
// it means the token was copied.
func (s *service) Refresh(refreshToken string, client ClientInfo) (TokenPair, error) {
	now := time.Now()

	sessionID, secret, ok := strings.Cut(refreshToken, ".")
	if !ok || sessionID == "" || secret == "" {
		return TokenPair{}, ErrSessionInvalid
	}

	session, err := s.repo.FindSessionByID(sessionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TokenPair{}, ErrSessionInvalid
	}
	if err != nil {
		return TokenPair{}, err
	}

	if !session.IsValid(now) || session.AppClientID != client.AppClientID {
		return TokenPair{}, ErrSessionInvalid
	}

	hash := helper.SHA256Hex([]byte(secret))
	if subtle.ConstantTimeCompare([]byte(hash), []byte(session.RefreshTokenHash)) != 1 {
		if err := s.repo.RevokeSession(session.ID); err != nil {
			return TokenPair{}, err
		}
		return TokenPair{}, ErrSessionInvalid
	}

	user, err := s.repo.FindUserByID(session.UserID)
	if err != nil || !user.IsActive || !user.Role.IsActive {
		return TokenPair{}, ErrSessionInvalid
	}
	if !UnitIsUsable(user) {
		return TokenPair{}, ErrUnitInactive
	}
	if err := checkLicense(user); err != nil {
		return TokenPair{}, err
	}

	if !models.RoleCanUsePlatform(user.Role.Code, client.AppPlatform) {
		return TokenPair{}, ErrPlatformNotAllowed
	}

	newSecret, err := helper.RandomURLToken(32)
	if err != nil {
		return TokenPair{}, err
	}

	session.ExpiresAt = now.Add(settingservice.ForUnit(user.UnitID).RefreshTokenTTL())
	if err := s.repo.RotateSessionToken(session.ID, helper.SHA256Hex([]byte(newSecret)), session.ExpiresAt, now); err != nil {
		return TokenPair{}, err
	}

	return s.issueTokens(user, session, newSecret, client.AppID, now)
}

func (s *service) Logout(sessionID string) error {
	return s.repo.RevokeSession(sessionID)
}

func (s *service) LogoutAll(userID int64) error {
	return s.repo.RevokeUserSessions(userID, "")
}

func (s *service) Authenticate(accessToken string, appID string, appPlatform string) (Principal, error) {
	claims, err := helper.ParseAccessToken(accessToken)
	if err != nil {
		return Principal{}, err
	}

	if claims.AppID != appID {
		return Principal{}, ErrTokenAppMismatch
	}

	session, err := s.repo.FindSessionByID(claims.SessionID)
	if err != nil || !session.IsValid(time.Now()) || session.UserID != claims.Subject {
		return Principal{}, ErrSessionInvalid
	}

	// Load the user on every request so deactivation and role changes apply immediately.
	user, err := s.repo.FindUserByID(claims.Subject)
	if err != nil || !user.IsActive || !user.Role.IsActive {
		return Principal{}, ErrSessionInvalid
	}
	// Checked on every request so deactivating a unit signs its users out at once.
	if !UnitIsUsable(user) {
		return Principal{}, ErrUnitInactive
	}
	if err := checkLicense(user); err != nil {
		return Principal{}, err
	}

	// Re-checked on every request so the rule also applies to tokens issued
	// before it existed or before the role changed.
	if !models.RoleCanUsePlatform(user.Role.Code, appPlatform) {
		return Principal{}, ErrPlatformNotAllowed
	}

	return Principal{
		UserID:    user.ID,
		RoleCode:  user.Role.Code,
		SessionID: session.ID,
		UnitID:    user.UnitID,
	}, nil
}

func (s *service) Me(userID int64) (models.User, error) {
	return s.repo.FindUserByID(userID)
}

func (s *service) ChangePassword(userID int64, sessionID string, oldPassword string, newPassword string) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	if !helper.CheckPassword(user.PasswordHash, oldPassword) {
		return ErrOldPasswordInvalid
	}

	if oldPassword == newPassword {
		return ErrSamePassword
	}

	if err := helper.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	hash, err := helper.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(userID, hash, time.Now()); err != nil {
		return err
	}

	// Keep the current device logged in, sign out every other device.
	return s.repo.RevokeUserSessions(userID, sessionID)
}

func (s *service) issueTokens(user models.User, session models.UserSession, refreshSecret string, appID string, now time.Time) (TokenPair, error) {
	accessExpiresAt := now.Add(settingservice.ForUnit(user.UnitID).AccessTokenTTL())

	accessToken, err := helper.GenerateAccessToken(helper.AccessTokenClaims{
		Subject:   user.ID,
		SessionID: session.ID,
		Role:      user.Role.Code,
		AppID:     appID,
		IssuedAt:  now.Unix(),
		ExpiresAt: accessExpiresAt.Unix(),
	})
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     session.ID + "." + refreshSecret,
		RefreshExpiresAt: session.ExpiresAt,
		User:             user,
	}, nil
}

func truncate(value string, max int) string {
	if len(value) > max {
		return value[:max]
	}
	return value
}
