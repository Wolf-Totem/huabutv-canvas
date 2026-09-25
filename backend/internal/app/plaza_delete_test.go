package app

import (
	"errors"
	"testing"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
)

func TestHardDeletePlazaWorkRejectsNonAdmin(t *testing.T) {
	svc := &Service{}
	_, err := svc.HardDeletePlazaWork(&model.User{ID: "user-1", Role: model.UserRoleUser}, "work-1")
	var appErr *kernel.AppError
	if !errors.As(err, &appErr) || appErr.Status != 403 {
		t.Fatalf("non-admin err = %v", err)
	}
	_, err = svc.HardDeletePlazaWork(nil, "work-1")
	if !errors.As(err, &appErr) || appErr.Status != 401 {
		t.Fatalf("anonymous err = %v", err)
	}
}
