package controller

import (
	"backend/internal/service"
	"errors"
	"net/http"
)

func serviceStatus(err error, fallback int) int {
	if errors.Is(err, service.ErrInvalidPassword) {
		return http.StatusBadRequest
	}
	if errors.Is(err, service.ErrForbidden) {
		return http.StatusForbidden
	}
	if errors.Is(err, service.ErrUnauthenticated) {
		return http.StatusUnauthorized
	}
	return fallback
}
