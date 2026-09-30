package service

import (
	"context"
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/appclient/repository"
	"secure-patrol-backend/pkg/log"
	"secure-patrol-backend/pkg/nonce"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	// Maximum difference between the client timestamp and server time.
	maxClockSkew = 5 * time.Minute
	minNonceLen  = 16
	maxNonceLen  = 64
)

type service struct {
	repo       repository.AppClientRepository
	nonceStore nonce.Store
}

func NewAppClientService(repo repository.AppClientRepository, nonceStore nonce.Store) AppClientService {
	return &service{repo: repo, nonceStore: nonceStore}
}

func (s *service) GetAppClients(pagination helper.Pagination) ([]models.AppClient, int64, error) {
	return s.repo.FindAll(pagination)
}

func (s *service) GetAppClientByID(id int64) (models.AppClient, error) {
	client, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return client, ErrAppClientNotFound
	}
	return client, err
}

func (s *service) CreateAppClient(client models.AppClient) (models.AppClient, string, error) {
	client.Name = strings.TrimSpace(client.Name)
	client.Platform = strings.ToLower(strings.TrimSpace(client.Platform))

	if _, ok := helper.AppPlatformCodes[client.Platform]; !ok {
		return client, "", ErrPlatformInvalid
	}

	appID, err := helper.GenerateAppID(client.Platform)
	if err != nil {
		return client, "", err
	}

	appKey, encryptedKey, err := newAppKey()
	if err != nil {
		return client, "", err
	}

	client.AppID = appID
	client.KeyEncrypted = encryptedKey
	client.KeyHint = helper.AppKeyHint(appKey)
	client.IsActive = true

	if err := s.repo.Create(&client); err != nil {
		return client, "", err
	}

	return client, appKey, nil
}

func (s *service) UpdateAppClient(id int64, input models.AppClient, currentAppClientID int64) (models.AppClient, error) {
	client, err := s.GetAppClientByID(id)
	if err != nil {
		return client, err
	}

	if client.ID == currentAppClientID && !input.IsActive {
		return client, ErrCannotModifyActive
	}

	client.Name = strings.TrimSpace(input.Name)
	client.Description = input.Description
	client.IsActive = input.IsActive
	client.ExpiresAt = input.ExpiresAt

	if err := s.repo.Update(&client); err != nil {
		return client, err
	}

	return client, nil
}

func (s *service) RotateKey(id int64, gracePeriod time.Duration) (models.AppClient, string, error) {
	client, err := s.GetAppClientByID(id)
	if err != nil {
		return client, "", err
	}

	appKey, encryptedKey, err := newAppKey()
	if err != nil {
		return client, "", err
	}

	now := time.Now()
	if gracePeriod > 0 {
		previousKey := client.KeyEncrypted
		previousExpiresAt := now.Add(gracePeriod)
		client.PreviousKeyEncrypted = &previousKey
		client.PreviousKeyExpiresAt = &previousExpiresAt
	} else {
		client.PreviousKeyEncrypted = nil
		client.PreviousKeyExpiresAt = nil
	}

	client.KeyEncrypted = encryptedKey
	client.KeyHint = helper.AppKeyHint(appKey)
	client.KeyRotatedAt = &now

	if err := s.repo.UpdateKeys(&client); err != nil {
		return client, "", err
	}

	return client, appKey, nil
}

func (s *service) DeleteAppClient(id int64, currentAppClientID int64) error {
	client, err := s.GetAppClientByID(id)
	if err != nil {
		return err
	}

	if client.ID == currentAppClientID {
		return ErrCannotModifyActive
	}

	return s.repo.Delete(client.ID)
}

// VerifyRequest checks, in order from cheapest to most expensive:
// headers present -> App ID checksum -> timestamp window -> nonce format ->
// app client active -> HMAC signature (current key, then previous key during
// its grace period) -> nonce not used before.
func (s *service) VerifyRequest(req SignedRequest) (models.AppClient, error) {
	var client models.AppClient

	if req.AppID == "" || req.Timestamp == "" || req.Nonce == "" || req.Signature == "" {
		return client, ErrAppCredentialMissing
	}

	if !helper.ValidateAppIDFormat(req.AppID) {
		return client, ErrAppCredentialInvalid
	}

	now := time.Now()
	unix, err := strconv.ParseInt(req.Timestamp, 10, 64)
	if err != nil {
		return client, ErrTimestampInvalid
	}
	skew := now.Sub(time.Unix(unix, 0))
	if skew > maxClockSkew || skew < -maxClockSkew {
		return client, ErrTimestampInvalid
	}

	if len(req.Nonce) < minNonceLen || len(req.Nonce) > maxNonceLen {
		return client, ErrNonceInvalid
	}

	client, err = s.repo.FindByAppID(req.AppID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return client, ErrAppCredentialInvalid
	}
	if err != nil {
		return client, err
	}

	if !client.IsActive || (client.ExpiresAt != nil && now.After(*client.ExpiresAt)) {
		return client, ErrAppClientDisabled
	}

	payload := helper.BuildSignaturePayload(req.Method, req.RequestURI, req.Timestamp, req.Nonce, req.Body)
	if !s.signatureMatches(client, payload, req.Signature, now) {
		return client, ErrSignatureInvalid
	}

	// The nonce is only stored after the signature is valid, so unsigned
	// requests cannot fill up the store. It is kept for twice the skew window,
	// which covers every timestamp that would still be accepted.
	fresh, err := s.nonceStore.Use(context.Background(), client.AppID+":"+req.Nonce, 2*maxClockSkew)
	if err != nil {
		return client, err
	}
	if !fresh {
		return client, ErrNonceReused
	}

	go func(id int64) {
		if err := s.repo.TouchLastUsed(id, time.Now()); err != nil {
			log.Errorf("app client touch last used: %v", err)
		}
	}(client.ID)

	return client, nil
}

func (s *service) signatureMatches(client models.AppClient, payload string, signature string, now time.Time) bool {
	if key, err := helper.DecryptAppKey(client.KeyEncrypted); err == nil &&
		helper.VerifyPayloadSignature(key, payload, signature) {
		return true
	}

	if client.PreviousKeyEncrypted != nil && client.PreviousKeyExpiresAt != nil && now.Before(*client.PreviousKeyExpiresAt) {
		if key, err := helper.DecryptAppKey(*client.PreviousKeyEncrypted); err == nil &&
			helper.VerifyPayloadSignature(key, payload, signature) {
			return true
		}
	}

	return false
}

func newAppKey() (plain string, encrypted string, err error) {
	plain, err = helper.GenerateAppKey()
	if err != nil {
		return "", "", err
	}

	encrypted, err = helper.EncryptAppKey(plain)
	return plain, encrypted, err
}
