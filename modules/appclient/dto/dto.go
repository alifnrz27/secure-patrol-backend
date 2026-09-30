package dto

import "secure-patrol-backend/models"

const appKeyWarning = "Store this app key securely. It is shown only once and cannot be retrieved again."

type AppClientDto interface {
	ToAppClientDTO(client models.AppClient) AppClientDTO
	ToAppClientDTOs(clients []models.AppClient) []AppClientDTO
	ToCredentialDTO(client models.AppClient, appKey string) AppClientCredentialDTO
}

type dto struct{}

func NewAppClientDto() AppClientDto {
	return &dto{}
}

func (d *dto) ToAppClientDTO(client models.AppClient) AppClientDTO {
	return AppClientDTO{
		ID:                   client.ID,
		Name:                 client.Name,
		Platform:             client.Platform,
		AppID:                client.AppID,
		KeyHint:              "****" + client.KeyHint,
		Description:          client.Description,
		IsActive:             client.IsActive,
		ExpiresAt:            client.ExpiresAt,
		LastUsedAt:           client.LastUsedAt,
		KeyRotatedAt:         client.KeyRotatedAt,
		PreviousKeyExpiresAt: client.PreviousKeyExpiresAt,
		CreatedAt:            client.CreatedAt,
		UpdatedAt:            client.UpdatedAt,
	}
}

func (d *dto) ToAppClientDTOs(clients []models.AppClient) []AppClientDTO {
	result := make([]AppClientDTO, 0, len(clients))
	for _, client := range clients {
		result = append(result, d.ToAppClientDTO(client))
	}
	return result
}

func (d *dto) ToCredentialDTO(client models.AppClient, appKey string) AppClientCredentialDTO {
	return AppClientCredentialDTO{
		AppClientDTO: d.ToAppClientDTO(client),
		AppKey:       appKey,
		Warning:      appKeyWarning,
	}
}
