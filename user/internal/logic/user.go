package logic

import (
	"context"

	"user/internal/dao"
	"user/internal/model/entity"
	"user/internal/service"
)

type userService struct{}

func NewUserService() service.UserService { return userService{} }

func (userService) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	columns := dao.User.Columns()
	var users []entity.User
	err := dao.User.Ctx(ctx).
		Where(columns.Username, username).
		Where(columns.IsDeleted, 0).
		Limit(1).
		Scan(&users)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, service.ErrUserNotFound
	}
	return &users[0], nil
}
