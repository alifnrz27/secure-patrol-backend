package dto

import "secure-patrol-backend/modules/setting/service"

type SettingDto interface {
	ToSettingDTOs(settings []service.Setting, forUnit bool) []SettingDTO
}

type dto struct{}

func NewSettingDto() SettingDto {
	return &dto{}
}

func (d *dto) ToSettingDTOs(settings []service.Setting, forUnit bool) []SettingDTO {
	level := "global"
	if forUnit {
		level = "unit"
	}

	result := make([]SettingDTO, 0, len(settings))
	for _, setting := range settings {
		item := SettingDTO{
			Key:          setting.Key,
			Level:        level,
			GlobalValue:  setting.Typed(setting.GlobalValue),
			IsInherited:  forUnit && setting.UnitValue == nil,
			Group:        setting.Group,
			Type:         setting.Type,
			Value:        setting.Typed(setting.Value),
			DefaultValue: setting.Typed(setting.Default),
			IsDefault:    setting.IsDefault,
			Unit:         setting.Unit,
			Description:  setting.Description,
			UpdatedBy:    setting.UpdatedBy,
			UpdatedAt:    setting.UpdatedAt,
		}
		if setting.Type != service.TypeBoolean {
			min, max := setting.Min, setting.Max
			item.Min, item.Max = &min, &max
		}
		result = append(result, item)
	}
	return result
}
