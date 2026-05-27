package handler

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type UserHandler struct {
	resetSecret string
	log         *zap.SugaredLogger
	userService UserService
}

type UserService interface {
	Create(ctx context.Context, request *model.CreateUserRequest) (*model.UserResponse, error)
	Update(ctx context.Context, request *model.UpdateUserRequest) (*model.UserResponse, error)
	Delete(ctx context.Context, request *model.DeleteUserRequest) (bool, error)
	FindById(ctx context.Context, request *model.FindByIdUserRequest) (*model.UserResponse, error)
	FindAll(ctx context.Context, request *model.FindAllUserRequest) ([]model.UserResponse, int64, error)
	ResetPassword(ctx context.Context, request *model.ResetPasswordRequest) error
	GetUsersStats(ctx context.Context) (*model.UsersStatsResponse, error)
	Login(ctx context.Context, request *model.LoginUserRequest) (*model.Auth, error)
	RefreshSession(ctx context.Context, request *model.RefreshSessionRequest) (*model.Auth, error)
	Logout(ctx context.Context, request *model.LogoutUserRequest) (bool, error)
	Heartbeat(ctx context.Context, userID int) error
}

var _ UserService = (*service.UserService)(nil)

func NewUserHandler(resetSecret string, logger *zap.SugaredLogger, userService UserService) *UserHandler {
	return &UserHandler{
		resetSecret: resetSecret,
		log:         logger,
		userService: userService,
	}
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	request := new(model.CreateUserRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	response, err := h.userService.Create(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to create user: %v", err)
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusCreated,
		Message: "create user success",
		Data:    response,
	})
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	userId, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		h.log.Warnf("failed to parse user id: %v", err)
		return err
	}

	request := new(model.UpdateUserRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.ID = userId

	response, err := h.userService.Update(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to update user: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusOK,
		Message: "update user success",
		Data:    response,
	})
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	userId, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		h.log.Warnf("failed to parse user id: %v", err)
		return err
	}

	request := &model.DeleteUserRequest{
		AuthID: auth.ID,
		ID:     userId,
	}

	response, err := h.userService.Delete(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to delete user by id: %v", err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "delete user success",
		Data:    response,
	})
}

func (h *UserHandler) FindById(c *fiber.Ctx) error {
	userId, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		h.log.Warnf("failed to parse user id: %v", err)
		return err
	}

	request := &model.FindByIdUserRequest{
		ID: userId,
	}

	response, err := h.userService.FindById(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find user by id: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusOK,
		Message: "find user by id success",
		Data:    response,
	})
}

func (h *UserHandler) FindAll(c *fiber.Ctx) error {
	request := &model.FindAllUserRequest{
		Fullname: c.Query("fullname", ""),
		Page:     c.QueryInt("page", 1),
		Size:     c.QueryInt("size", 10),
	}

	response, total, err := h.userService.FindAll(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find all users: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.UserResponse]{
		Status:  fiber.StatusOK,
		Message: "find all users success",
		Data:    response,
		Paging:  newPageMetadata(request.Page, request.Size, total),
	})
}

func (h *UserHandler) Register(c *fiber.Ctx) error {
	if h.resetSecret == "" || c.Get("X-Reset-Secret") != h.resetSecret {
		h.log.Warnf("register: invalid or missing secret")
		return fiber.ErrUnauthorized
	}

	request := new(model.CreateUserRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("register: failed to parse body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	response, err := h.userService.Create(c.Context(), request)
	if err != nil {
		h.log.Warnf("register: failed to create user: %v", err)
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusCreated,
		Message: "register user success",
		Data:    response,
	})
}

func (h *UserHandler) ResetPassword(c *fiber.Ctx) error {
	if h.resetSecret == "" || c.Get("X-Reset-Secret") != h.resetSecret {
		h.log.Warnf("reset-password: invalid or missing secret")
		return fiber.ErrUnauthorized
	}

	request := new(model.ResetPasswordRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("reset-password: failed to parse body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.userService.ResetPassword(c.Context(), request); err != nil {
		h.log.Warnf("reset-password: failed: %v", err)
		return err
	}

	return c.JSON(model.Response[any]{
		Status:  fiber.StatusOK,
		Message: "password reset success, user session invalidated",
	})
}

func (h *UserHandler) GetUsersStats(c *fiber.Ctx) error {
	response, err := h.userService.GetUsersStats(c.Context())
	if err != nil {
		h.log.Warnf("failed to get users statistics: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.UsersStatsResponse]{
		Status:  fiber.StatusOK,
		Message: "get users statistics success",
		Data:    response,
	})
}
