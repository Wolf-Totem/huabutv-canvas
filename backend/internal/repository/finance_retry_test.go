package repository

import (
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRetryTaskWithBillingResetsAttempts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+newRepositoryID()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Task{}, &model.CreditAccount{}, &model.BillingOrder{}, &model.TaskTextDelta{}); err != nil {
		t.Fatal(err)
	}
	repo := &Repository{db: db}
	now := time.Now()
	task := model.Task{
		ID: "task-1", UserID: "user-1", Status: model.TaskStatusFailed, Attempts: 3,
		ProviderRequestID: "old-upstream", RouteRun: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}

	retried, err := repo.RetryTaskWithBilling("user-1", &task, nil, 8)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Attempts != 0 {
		t.Fatalf("attempts after retry = %d, want 0", retried.Attempts)
	}
	if retried.Status != model.TaskStatusQueued || retried.ProviderRequestID != "" || retried.RouteRun != 2 {
		t.Fatalf("retried task = %#v", retried)
	}
}
