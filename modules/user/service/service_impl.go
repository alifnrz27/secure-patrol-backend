package service

import (
	"errors"
	"mime/multipart"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	rolerepository "secure-patrol-backend/modules/role/repository"
	settingservice "secure-patrol-backend/modules/setting/service"
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
}

func NewUserService(
	repo repository.UserRepository,
	roleRepo rolerepository.RoleRepository,
) UserService {
	return &service{
		repo:     repo,
		roleRepo: roleRepo,
	}
}

func (s *service) GetUsers(filter dto.UserFilter) ([]models.User, int64, error) {
	return s.repo.FindAll(filter)
}

func (s *service) GetUserByID(id int64) (models.User, error) {
	user, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user, ErrUserNotFound
	}
	return user, err
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

	if err := s.checkAssignableRole(actor, user.RoleID); err != nil {
		return user, err
	}

	if err := s.checkEmailAvailable(user.Email, 0); err != nil {
		return user, err
	}

	hash, err := helper.HashPassword(password)
	if err != nil {
		return user, err
	}

	photoPath, err := saveFacePhoto(facePhoto)
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
	user, err := s.GetUserByID(id)
	if err != nil {
		return user, err
	}

	if err := s.checkManageableUser(actor, user); err != nil {
		return user, err
	}

	if actor.UserID == user.ID && (!input.IsActive || input.RoleID != user.RoleID) {
		return user, ErrCannotModifySelf
	}

	if input.RoleID != user.RoleID {
		if err := s.checkAssignableRole(actor, input.RoleID); err != nil {
			return user, err
		}
	}

	email := normalizeEmail(input.Email)
	if email != user.Email {
		if err := s.checkEmailAvailable(email, user.ID); err != nil {
			return user, err
		}
	}

	oldPhotoPath := user.FacePhotoPath
	if facePhoto != nil {
		photoPath, err := saveFacePhoto(facePhoto)
		if err != nil {
			return user, err
		}
		user.FacePhotoPath = photoPath
		now := time.Now()
		user.FacePhotoUpdatedAt = &now
	}

	// Sessions must be dropped when access is reduced so the change applies immediately.
	revokeSessions := (user.IsActive && !input.IsActive) || input.RoleID != user.RoleID

	user.Name = strings.TrimSpace(input.Name)
	user.Email = email
	user.RoleID = input.RoleID
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
	user, err := s.GetUserByID(id)
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
	user, err := s.GetUserByID(id)
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

func (s *service) GetFacePhotoPath(id int64) (string, error) {
	user, err := s.GetUserByID(id)
	if err != nil {
		return "", err
	}

	if user.FacePhotoPath == "" {
		return "", ErrFacePhotoNotExists
	}

	return helper.StoragePath(user.FacePhotoPath)
}

// checkAssignableRole makes sure the role exists, is active, and that only a
// Super-Admin can give out the Super-Admin role.
func (s *service) checkAssignableRole(actor Actor, roleID int64) error {
	role, err := s.roleRepo.FindByID(roleID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRoleInvalid
	}
	if err != nil {
		return err
	}

	if !role.IsActive {
		return ErrRoleInvalid
	}

	if role.Code == models.RoleSuperAdmin && actor.RoleCode != models.RoleSuperAdmin {
		return ErrForbiddenRole
	}

	return nil
}

// checkManageableUser prevents non Super-Admin users from changing Super-Admin accounts.
func (s *service) checkManageableUser(actor Actor, user models.User) error {
	if user.Role.Code == models.RoleSuperAdmin && actor.RoleCode != models.RoleSuperAdmin {
		return ErrForbiddenRole
	}
	return nil
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
// A rejected photo is never written to storage.
func saveFacePhoto(file *multipart.FileHeader) (string, error) {
	data, ext, err := helper.ReadImage(file)
	if err != nil {
		return "", err
	}

	settings := settingservice.Current()
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
