package service

import (
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"time"
)

var (
	ErrAppClientNotFound  = errors.New("app client not found")
	ErrPlatformInvalid    = errors.New("platform must be one of android, ios, web, server")
	ErrCannotModifyActive = errors.New("you cannot delete or deactivate the app client used by this request")

	// Request verification errors. They are all answered with 401 so a caller
	// cannot learn which part of its credentials is wrong.
	ErrAppCredentialMissing = errors.New("app credential headers are missing")
	ErrAppCredentialInvalid = errors.New("app credential is invalid")
	ErrAppClientDisabled    = errors.New("app client is disabled or expired")
	ErrTimestampInvalid     = errors.New("request timestamp is invalid or outside the allowed window")
	ErrNonceInvalid         = errors.New("request nonce is invalid")
	ErrNonceReused          = errors.New("request nonce has already been used")
	ErrSignatureInvalid     = errors.New("request signature is invalid")
)

// SignedRequest holds the parts of an HTTP request covered by the signature.
type SignedRequest struct {
	AppID      string
	Timestamp  string
	Nonce      string
	Signature  string
	Method     string
	RequestURI string
	Body       []byte
}

type AppClientService interface {
	GetAppClients(pagination helper.Pagination) ([]models.AppClient, int64, error)
	GetAppClientByID(id int64) (models.AppClient, error)
	// CreateAppClient returns the plain app key; it cannot be retrieved again later.
	CreateAppClient(client models.AppClient) (models.AppClient, string, error)
	UpdateAppClient(id int64, input models.AppClient, currentAppClientID int64) (models.AppClient, error)
	// RotateKey issues a new key. The old key keeps working for gracePeriod so
	// installed apps can be updated.
	RotateKey(id int64, gracePeriod time.Duration) (models.AppClient, string, error)
	DeleteAppClient(id int64, currentAppClientID int64) error
	VerifyRequest(req SignedRequest) (models.AppClient, error)
}
