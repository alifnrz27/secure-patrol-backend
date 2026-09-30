package dto

import (
	"fmt"
	"math"
	"secure-patrol-backend/models"
	"time"
)

type PatrolDto interface {
	ToGroupDTO(group models.PatrolGroup) PatrolGroupDTO
	ToGroupDTOs(groups []models.PatrolGroup) []PatrolGroupDTO
	ToGroupDetailDTO(group models.PatrolGroup, items []models.PatrolListItem) PatrolGroupDetailDTO
	ToItemDTOs(items []models.PatrolListItem, withGroup bool) []PatrolListItemDTO
	ToScanDTO(scan models.PatrolScan) PatrolScanDTO
	ToScanDTOs(scans []models.PatrolScan) []PatrolScanDTO
}

type dto struct{}

func NewPatrolDto() PatrolDto {
	return &dto{}
}

func groupStatus(group models.PatrolGroup, now time.Time) string {
	switch {
	case now.Before(group.StartAt):
		return "upcoming"
	case now.Before(group.EndAt):
		return "ongoing"
	default:
		return "finished"
	}
}

func groupSummary(group models.PatrolGroup) GroupSummaryDTO {
	return GroupSummaryDTO{
		ID:        group.ID,
		ShiftID:   group.PatrolShiftID,
		ShiftName: group.ShiftName,
		ShiftDate: group.ShiftDate.Format("2006-01-02"),
		StartAt:   group.StartAt,
		EndAt:     group.EndAt,
	}
}

func userSummary(user *models.User) *UserSummaryDTO {
	if user == nil || user.ID == 0 {
		return nil
	}
	return &UserSummaryDTO{ID: user.ID, Name: user.Name, Email: user.Email}
}

func (d *dto) ToGroupDTO(group models.PatrolGroup) PatrolGroupDTO {
	return PatrolGroupDTO{
		ID:        group.ID,
		Shift:     ShiftSummaryDTO{ID: group.PatrolShiftID, Name: group.ShiftName},
		ShiftDate: group.ShiftDate.Format("2006-01-02"),
		StartAt:   group.StartAt,
		EndAt:     group.EndAt,
		Status:    groupStatus(group, time.Now()),
		Progress: ProgressDTO{
			TotalPoints:     group.TotalPoints,
			ScannedPoints:   group.ScannedPoints,
			UnscannedPoints: group.TotalPoints - group.ScannedPoints,
			TotalScans:      group.TotalScans,
			AbnormalScans:   group.AbnormalScans,
		},
		CreatedAt: group.CreatedAt,
	}
}

func (d *dto) ToGroupDTOs(groups []models.PatrolGroup) []PatrolGroupDTO {
	result := make([]PatrolGroupDTO, 0, len(groups))
	for _, group := range groups {
		result = append(result, d.ToGroupDTO(group))
	}
	return result
}

func (d *dto) ToGroupDetailDTO(group models.PatrolGroup, items []models.PatrolListItem) PatrolGroupDetailDTO {
	return PatrolGroupDetailDTO{
		PatrolGroupDTO: d.ToGroupDTO(group),
		Items:          d.ToItemDTOs(items, false),
	}
}

func (d *dto) ToItemDTOs(items []models.PatrolListItem, withGroup bool) []PatrolListItemDTO {
	result := make([]PatrolListItemDTO, 0, len(items))
	for _, item := range items {
		itemDTO := PatrolListItemDTO{
			ID:                       item.ID,
			PatrolPointID:            item.PatrolPointID,
			Name:                     item.Name,
			Location:                 item.Location,
			NFCCode:                  item.NFCCode,
			Latitude:                 item.Latitude,
			Longitude:                item.Longitude,
			IsLocationMatchRequired:  item.IsLocationMatchRequired,
			IsFaceValidationRequired: item.IsFaceValidationRequired,
			IsScanned:                item.ScanCount > 0,
			ScanCount:                item.ScanCount,
			LastScannedAt:            item.LastScannedAt,
			LastScannedBy:            userSummary(item.LastScannedByUser),
			LastCondition:            item.LastCondition,
		}
		if withGroup && item.PatrolGroup != nil {
			summary := groupSummary(*item.PatrolGroup)
			itemDTO.Group = &summary
		}
		result = append(result, itemDTO)
	}
	return result
}

func (d *dto) ToScanDTO(scan models.PatrolScan) PatrolScanDTO {
	photos := make([]ScanPhotoDTO, 0, len(scan.Photos))
	for _, photo := range scan.Photos {
		photos = append(photos, ScanPhotoDTO{
			ID:  photo.ID,
			URL: fmt.Sprintf("/api/v1/patrol-scans/%d/photos/%d", scan.ID, photo.ID),
		})
	}

	scannedBy := UserSummaryDTO{ID: scan.ScannedBy}
	if user := userSummary(&scan.ScannedByUser); user != nil {
		scannedBy = *user
	}

	return PatrolScanDTO{
		ID:           scan.ID,
		ClientScanID: scan.ClientScanID,
		Group:        groupSummary(scan.PatrolGroup),
		PatrolPoint: ScanPointDTO{
			PatrolListItemID: scan.PatrolListItemID,
			PatrolPointID:    scan.PatrolPointID,
			Name:             scan.PatrolListItem.Name,
			Location:         scan.PatrolListItem.Location,
			NFCCode:          scan.NFCCode,
		},
		Condition:       scan.Condition,
		Note:            scan.Note,
		Latitude:        scan.Latitude,
		Longitude:       scan.Longitude,
		DistanceMeters:  math.Round(scan.DistanceMeters*10) / 10,
		IsLocationValid: scan.IsLocationValid,
		IsFaceVerified:  scan.IsFaceVerified,
		FaceMatchScore:  scan.FaceMatchScore,
		ScannedAt:       scan.ScannedAt,
		ReceivedAt:      scan.ReceivedAt,
		ScannedBy:       scannedBy,
		Photos:          photos,
	}
}

func (d *dto) ToScanDTOs(scans []models.PatrolScan) []PatrolScanDTO {
	result := make([]PatrolScanDTO, 0, len(scans))
	for _, scan := range scans {
		result = append(result, d.ToScanDTO(scan))
	}
	return result
}
