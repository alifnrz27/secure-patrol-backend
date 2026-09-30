package service

import (
	"errors"
	"mime/multipart"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	rolerepository "secure-patrol-backend/modules/role/repository"
	settingservice "secure-patrol-backend/modules/setting/service"
	unitrepository "secure-patrol-backend/modules/unit/repository"
	"secure-patrol-backend/modules/user/dto"
	"secure-patrol-backend/modules/user/repository"
	"secure-patrol-backend/pkg/facedetect"
	"strings"
	"time"

	"gorm.io/gorm"
)

const facePhotoDir = "faces"

type service struct {
	repo     repository.UserRepository
	roleRepo rolerepository.RoleRepository
	unitRepo unitrepository.UnitRepository
}

func NewUserService(
	repo repository.UserRepository,
	roleRepo rolerepository.RoleRepository,
	unitRepo unitrepository.UnitRepository,
) UserService {
	return &service{
		repo:     repo,
		roleRepo: roleRepo,
		unitRepo: unitRepo,
	}
}

// GetUsers expects the filter to be limited to the actor's unit by the caller.
func (s *service) GetUsers(filter dto.UserFilter) ([]models.User, int64, error) {
	return s.repo.FindAll(filter)
}

func (s *service) GetUserByID(actor Actor, id int64) (models.User, error) {
	user, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user, ErrUserNotFound
	}
	if err != nil {
		return user, err
	}
	// Users of other units (and head office users) do not exist for a unit manager.
	if actor.UnitID != nil && (user.UnitID == nil || *user.UnitID != *actor.UnitID) {
		return models.User{}, ErrUserNotFound
	}
	return user, nil
}

func (s *service) CreateUser(actor Actor, user models.User, password string, facePhoto *multipart.FileHeader) (models.User, error) {
	user.Name = strings.TrimSpace(user.Name)
	user.Email = normalizeEmail(user.Email)

	if facePhoto == nil {
		return user, ErrFacePhotoRequired
	}

	if err := helper.ValidatePasswordStrength(password); err != nil {
		return user, err
	}

	role, err := s.checkAssignableRole(actor, user.RoleID)
	if err != nil {
		return user, err
	}

	user.UnitID, err = s.resolveUnit(actor, role, user.UnitID)
	if err != nil {
		return user, err
	}

	if err := s.checkEmailAvailable(user.Email, 0); err != nil {
		return user, err
	}

	hash, err := helper.HashPassword(password)
	if err != nil {
		return user, err
	}

	photoPath, err := saveFacePhoto(facePhoto, user.UnitID)
	if err != nil {
		return user, err
	}

	now := time.Now()
	user.PasswordHash = hash
	user.PasswordChangedAt = &now
	user.FacePhotoPath = photoPath
	user.FacePhotoUpdatedAt = &now

	if err := s.repo.Create(&user); err != nil {
		helper.DeleteStoredFile(photoPath)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return user, ErrEmailTaken
		}
		return user, err
	}

	return user, nil
}

func (s *service) UpdateUser(actor Actor, id int64, input models.User, facePhoto *multipart.FileHeader) (models.User, error) {
	user, err := s.GetUserByID(actor, id)
	if err != nil {
		return user, err
	}

	if err := s.checkManageableUser(actor, user); err != nil {
		return user, err
	}

	role := user.Role
	if input.RoleID != user.RoleID {
		if role, err = s.checkAssignableRole(actor, input.RoleID); err != nil {
			return user, err
		}
	}

	// Keep the current unit when the form does not send one.
	requestedUnit := input.UnitID
	if requestedUnit == nil {
		requestedUnit = user.UnitID
	}
	unitID, err := s.resolveUnit(actor, role, requestedUnit)
	if err != nil {
		return user, err
	}
	unitChanged := !sameUnit(unitID, user.UnitID)

	if actor.UserID == user.ID && (!input.IsActive || input.RoleID != user.RoleID || unitChanged) {
		return user, ErrCannotModifySelf
	}

	email := normalizeEmail(input.Email)
	if email != user.Email {
		if err := s.checkEmailAvailable(email, user.ID); err != nil {
			return user, err
		}
	}

	oldPhotoPath := user.FacePhotoPath
	if facePhoto != nil {
		photoPath, err := saveFacePhoto(facePhoto, unitID)
		if err != nil {
			return user, err
		}
		user.FacePhotoPath = photoPath
		now := time.Now()
		user.FacePhotoUpdatedAt = &now
	}

	// Sessions must be dropped when access is reduced so the change applies immediately.
	revokeSessions := (user.IsActive && !input.IsActive) || input.RoleID != user.RoleID || unitChanged

	user.Name = strings.TrimSpace(input.Name)
	user.Email = email
	user.RoleID = input.RoleID
	user.UnitID = unitID
	user.Unit = nil
	user.IsActive = input.IsActive

	if err := s.repo.Update(&user); err != nil {
		if user.FacePhotoPath != oldPhotoPath {
			helper.DeleteStoredFile(user.FacePhotoPath)
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return user, ErrEmailTaken
		}
		return user, err
	}

	if user.FacePhotoPath != oldPhotoPath {
		helper.DeleteStoredFile(oldPhotoPath)
	}

	if revokeSessions {
		if err := s.repo.RevokeSessions(user.ID); err != nil {
			return user, err
		}
	}

	return user, nil
}

func (s *service) DeleteUser(actor Actor, id int64) error {
	user, err := s.GetUserByID(actor, id)
	if err != nil {
		return err
	}

	if actor.UserID == user.ID {
		return ErrCannotModifySelf
	}

	if err := s.checkManageableUser(actor, user); err != nil {
		return err
	}

	// The face photo is kept for audit purposes; only the account is soft deleted.
	return s.repo.Delete(user)
}

func (s *service) ResetPassword(actor Actor, id int64, password string) error {
	user, err := s.GetUserByID(actor, id)
	if err != nil {
		return err
	}

	if err := s.checkManageableUser(actor, user); err != nil {
		return err
	}

	if err := helper.ValidatePasswordStrength(password); err != nil {
		return err
	}

	hash, err := helper.HashPassword(password)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(user.ID, hash, time.Now()); err != nil {
		return err
	}

	return s.repo.RevokeSessions(user.ID)
}

func (s *service) GetFacePhotoPath(actor Actor, id int64) (string, error) {
	user, err := s.GetUserByID(actor, id)
	if err != nil {
		return "", err
	}

	if user.FacePhotoPath == "" {
		return "", ErrFacePhotoNotExists
	}

	return helper.StoragePath(user.FacePhotoPath)
}

// checkAssignableRole makes sure the role exists, is active, and that only the
// Super-Admin gives out head office roles (Super-Admin, Security Manager).
func (s *service) checkAssignableRole(actor Actor, roleID int64) (models.Role, error) {
	role, err := s.roleRepo.FindByID(roleID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return role, ErrRoleInvalid
	}
	if err != nil {
		return role, err
	}

	if !role.IsActive {
		return role, ErrRoleInvalid
	}

	if models.IsCentralRole(role.Code) && actor.RoleCode != models.RoleSuperAdmin {
		return role, ErrForbiddenRole
	}

	return role, nil
}

// resolveUnit returns the unit a user with the role belongs to. Head office
// roles have no unit. Unit managers can only place users in their own unit,
// whatever was requested; the Super-Admin must pick an existing unit.
func (s *service) resolveUnit(actor Actor, role models.Role, requested *int64) (*int64, error) {
	if models.IsCentralRole(role.Code) {
		return nil, nil
	}

	if actor.UnitID != nil {
		unitID := *actor.UnitID
		return &unitID, nil
	}

	if requested == nil || *requested <= 0 {
		return nil, ErrUnitRequired
	}
	if _, err := s.unitRepo.FindByID(*requested); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnitInvalid
		}
		return nil, err
	}
	unitID := *requested
	return &unitID, nil
}

// checkManageableUser only lets the Super-Admin change head office accounts.
// Unit managers only reach users of their own unit (see GetUserByID).
func (s *service) checkManageableUser(actor Actor, user models.User) error {
	if models.IsCentralRole(user.Role.Code) && actor.RoleCode != models.RoleSuperAdmin {
		return ErrForbiddenRole
	}
	return nil
}

func sameUnit(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (s *service) checkEmailAvailable(email string, exceptUserID int64) error {
	existing, err := s.repo.FindByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != exceptUserID {
		return ErrEmailTaken
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// saveFacePhoto stores a reference face photo after checking it contains
// exactly one large, frontal face, so the mobile app can match against it.
// A rejected photo is never written to storage. The checks use the settings of
// the user's unit.
func saveFacePhoto(file *multipart.FileHeader, unitID *int64) (string, error) {
	data, ext, err := helper.ReadImage(file)
	if err != nil {
		return "", err
	}

	settings := settingservice.ForUnit(unitID)
	if settings.FacePhotoValidation {
		options := facedetect.Options{
			MinFaceRatio:   settings.FaceMinSizeRatio,
			MaxTiltDegrees: settings.FaceMaxTiltDegrees,
			MaxTurnRatio:   settings.FaceMaxTurnRatio,
		}
		if _, err := facedetect.Validate(data, options); err != nil {
			return "", err
		}
	}

	return helper.SaveImageBytes(data, ext, facePhotoDir)
}
