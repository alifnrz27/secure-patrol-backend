package service

import (
	"errors"
	"regexp"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/role/repository"
	"strings"

	"gorm.io/gorm"
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,49}$`)

type service struct {
	repo repository.RoleRepository
}

func NewRoleService(repo repository.RoleRepository) RoleService {
	return &service{repo: repo}
}

func (s *service) GetRoles(pagination helper.Pagination) ([]models.Role, int64, error) {
	return s.repo.FindAll(pagination)
}

func (s *service) GetRoleByID(id int64) (models.Role, error) {
	role, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return role, ErrRoleNotFound
	}
	return role, err
}

func (s *service) CreateRole(role models.Role) (models.Role, error) {
	role.Code = strings.ToLower(strings.TrimSpace(role.Code))
	role.Name = strings.TrimSpace(role.Name)
	role.IsSystem = false

	if !roleCodePattern.MatchString(role.Code) {
		return role, ErrRoleCodeInvalid
	}

	if _, err := s.repo.FindByCode(role.Code); err == nil {
		return role, ErrRoleCodeTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return role, err
	}

	if err := s.repo.Create(&role); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return role, ErrRoleCodeTaken
		}
		return role, err
	}

	return role, nil
}

func (s *service) UpdateRole(id int64, input models.Role) (models.Role, error) {
	role, err := s.GetRoleByID(id)
	if err != nil {
		return role, err
	}

	if role.IsSystem && !input.IsActive {
		return role, ErrRoleSystemLocked
	}

	// The code is immutable because it is referenced by authorization checks.
	role.Name = strings.TrimSpace(input.Name)
	role.Description = input.Description
	role.IsActive = input.IsActive

	if err := s.repo.Update(&role); err != nil {
		return role, err
	}

	return role, nil
}

func (s *service) DeleteRole(id int64) error {
	role, err := s.GetRoleByID(id)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrRoleSystemLocked
	}

	count, err := s.repo.CountUsers(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleInUse
	}

	return s.repo.Delete(id)
}
