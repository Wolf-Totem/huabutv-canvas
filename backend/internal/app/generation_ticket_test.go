package app

import (
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConsumeTaskTicketRejectsSecondSubmit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(10 * time.Minute)
	task := model.Task{ID: "task-1", UserID: "user-1", Type: "canvas_video", Status: model.TaskStatusQueued, TicketJTI: "jti-1", TicketExpiresAt: &expires, ClientSubmit: true}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.New(db)
	svc := &Service{repo: repo}
	if _, err := svc.ReportTaskProviderRequest("user-1", "task-1", "provider-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReportTaskProviderRequest("user-1", "task-1", "provider-2"); err == nil {
		t.Fatal("second submit should fail")
	}
}

func TestConsumeTaskTicketRejectsExpired(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(-time.Minute)
	task := model.Task{ID: "task-2", UserID: "user-1", Type: "canvas_video", Status: model.TaskStatusQueued, TicketJTI: "jti-2", TicketExpiresAt: &expires, ClientSubmit: true}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	svc := &Service{repo: repository.New(db)}
	if _, err := svc.ReportTaskProviderRequest("user-1", "task-2", "provider-1"); err == nil {
		t.Fatal("expired ticket should fail")
	}
}
