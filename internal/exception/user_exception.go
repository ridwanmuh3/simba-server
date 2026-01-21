package exception

import "github.com/gofiber/fiber/v2"

var (
	UserNotFoundError               = fiber.NewError(fiber.StatusNotFound, "user not found")
	UserAlreadyExistsError          = fiber.NewError(fiber.StatusConflict, "user already exists")
	InvalidUserRoleError            = fiber.NewError(fiber.StatusBadRequest, "invalid user role. only accept 'Admin' or 'Super Admin'")
	UserPasswordNotMatch            = fiber.NewError(fiber.StatusBadRequest, "user password not match")
	UserCannotSelfDeleteError       = fiber.NewError(fiber.StatusBadRequest, "cannot self delete user")
	UserDeleteSingleSuperAdminError = fiber.NewError(fiber.StatusForbidden, "cannot delete super admin user if exists only one")
	UserDeleteActiveUserError       = fiber.NewError(fiber.StatusForbidden, "cannot delete currently active user")
)
