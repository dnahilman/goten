package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dnahilman/goten/examples/layered-gin/internal/model"
)

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) FindByID(ctx context.Context, userID string) (*model.UserProfile, error) {
	var u model.UserProfile
	if err := r.db.WithContext(ctx).First(&u, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *model.UserProfile) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *UserRepo) UpdatePhone(ctx context.Context, userID, phone string) error {
	return r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		Update("phone", phone).Error
}
