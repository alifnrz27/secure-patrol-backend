package routes

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/setting/dto"
	"secure-patrol-backend/modules/setting/http"
	"secure-patrol-backend/modules/setting/repository"
	"secure-patrol-backend/modules/setting/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SettingRouter(app *fiber.App, db *gorm.DB) {
	// Use the instance shared with the rest of the app, so an update is applied
	// everywhere at once instead of after the other instances' cache expires.
	settingService := service.Instance()
	if settingService == nil {
		settingService = service.NewSettingService(repository.NewSettingRepository(db))
	}
	settingDto := dto.NewSettingDto()
	unitExists := func(unitID int64) (bool, error) {
		var count int64
		err := db.Model(&models.Unit{}).Where("id = ?", unitID).Count(&count).Error
		return count > 0, err
	}
	settingHandler := http.NewSettingHandler(settingService, settingDto, unitExists)

	http.SettingRoutes(app, settingHandler)
}
