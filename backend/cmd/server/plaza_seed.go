package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"infinite-canvas/backend/internal/app"
	"infinite-canvas/backend/internal/database"
	"infinite-canvas/backend/internal/repository"
	"infinite-canvas/backend/internal/service"
)

func runPlazaSeed(_ context.Context) error {
	dataDir := env("CANVAS_BACKEND_DATA_DIR", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	db, err := database.Open(database.Config{
		Driver:  env("CANVAS_DATABASE_DRIVER", "sqlite"),
		DSN:     os.Getenv("DATABASE_URL"),
		DataDir: dataDir,
	})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	repo := repository.New(db)
	svc := service.New(repo, dataDir)
	defer svc.Close()
	decoder := json.NewDecoder(os.Stdin)
	var items []app.PlazaExternalSeedItem
	if err := decoder.Decode(&items); err != nil {
		return fmt.Errorf("读取导入清单失败: %w", err)
	}
	report, err := svc.SeedPlazaExternal(items)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
