package controller

import (
	"backend/internal/service"
	kafkapb "backend/proto/kafka"
	"errors"
	"net/http"
)

func serviceStatus(err error, fallback int) int {
	if errors.Is(err, service.ErrInvalidCursor) {
		return http.StatusBadRequest
	}
	if errors.Is(err, service.ErrCursorReset) {
		return http.StatusConflict
	}
	if errors.Is(err, kafkapb.ErrMessageTooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	if errors.Is(err, kafkapb.ErrInvalidMessage) {
		return http.StatusBadRequest
	}
	if errors.Is(err, service.ErrRequestConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, service.ErrSendBusy) {
		return http.StatusServiceUnavailable
	}
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
