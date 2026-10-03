package dto

import (
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auth/service"
	licensehttp "secure-patrol-backend/modules/license/http"
	settingservice "secure-patrol-backend/modules/setting/service"
	userdto "secure-patrol-backend/modules/user/dto"
	"secure-patrol-backend/pkg/log"
	"time"
)

type AuthDto interface {
	ToTokenDTO(pair service.TokenPair) TokenDTO
	// ToLoginTokenDTO also embeds the user's face photo as base64.
	ToLoginTokenDTO(pair service.TokenPair) TokenDTO
	// ToAppConfigDTO uses the settings of the unit (nil = global settings).
	ToAppConfigDTO(unitID *int64) AppConfigDTO
	ToProfileDTO(user models.User) ProfileDTO
}

type dto struct {
	userDto userdto.UserDto
}

func NewAuthDto() AuthDto {
	return &dto{userDto: userdto.NewUserDto()}
}

func (d *dto) ToTokenDTO(pair service.TokenPair) TokenDTO {
	return TokenDTO{
		TokenType:        "Bearer",
		AccessToken:      pair.AccessToken,
		ExpiresIn:        int64(time.Until(pair.AccessExpiresAt).Seconds()),
		ExpiresAt:        pair.AccessExpiresAt,
		RefreshToken:     pair.RefreshToken,
		RefreshExpiresAt: pair.RefreshExpiresAt,
		User:             LoginUserDTO{UserDTO: d.userDto.ToUserDTO(pair.User)},
		Config:           d.ToAppConfigDTO(pair.User.UnitID),
		Settings:         settingservice.ForUnit(pair.User.UnitID).PublicMap(),
		License:          licensehttp.Summary(),
	}
}

func (d *dto) ToProfileDTO(user models.User) ProfileDTO {
	return ProfileDTO{
		UserDTO:  d.userDto.ToUserDTO(user),
		Settings: settingservice.ForUnit(user.UnitID).PublicMap(),
		License:  licensehttp.Summary(),
	}
}

func (d *dto) ToAppConfigDTO(unitID *int64) AppConfigDTO {
	settings := settingservice.ForUnit(unitID)
	var minScore *float64
	if score := settings.FaceMatchMinScore; score > 0 {
		minScore = &score
	}

	return AppConfigDTO{
		ServerTime:                 time.Now().In(helper.AppLocation()),
		Timezone:                   helper.AppLocation().String(),
		RequestTimestampToleranceS: 300,
		LocationRadiusMeters:       settings.PatrolLocationRadiusMeters,
		FaceMatchMinScore:          minScore,
		MaxOfflineHours:            settings.PatrolMaxOfflineHours,
		MaxScanPhotos:              helper.MaxScanPhotos,
		MaxPhotoSizeBytes:          helper.MaxFacePhotoSize,
	}
}

func (d *dto) ToLoginTokenDTO(pair service.TokenPair) TokenDTO {
	token := d.ToTokenDTO(pair)

	if pair.User.FacePhotoPath == "" {
		return token
	}

	// A missing photo file must not block the login, so it is only logged.
	encoded, contentType, err := helper.ReadImageBase64(pair.User.FacePhotoPath)
	if err != nil {
		log.Warnf("login: cannot read face photo of user %d: %v", pair.User.ID, err)
		return token
	}

	token.User.FacePhotoBase64 = &encoded
	token.User.FacePhotoMimeType = &contentType
	return token
}
