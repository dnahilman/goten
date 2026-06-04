package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/dnahilman/goten/examples/layered-gin/internal/model"
	"github.com/dnahilman/goten/examples/layered-gin/internal/repository"
	"github.com/go-playground/validator/v10"
)

var phoneRe = regexp.MustCompile(`^\+?\d{8,15}$`)

// validate carries a custom "phone" rule backed by phoneRe, so validation goes
// through go-playground/validator instead of an ad-hoc regexp check.
var validate = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	_ = v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		return phoneRe.MatchString(fl.Field().String())
	})
	return v
}()

var ErrInvalidPhone = errors.New("phone must be 8-15 digits, optionally prefixed with +")

type UserService struct{ repo *repository.UserRepo }

func NewUserService(r *repository.UserRepo) *UserService { return &UserService{repo: r} }

func (s *UserService) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *UserService) CreateProfile(ctx context.Context, userID, fullName string) (*model.UserProfile, error) {
	u := &model.UserProfile{
		UserID:   userID,
		FullName: fullName,
		Role:     "customer",
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) UpdatePhone(ctx context.Context, userID, phone string) error {
	if err := validate.Var(phone, "required,phone"); err != nil {
		return ErrInvalidPhone
	}
	return s.repo.UpdatePhone(ctx, userID, phone)
}
