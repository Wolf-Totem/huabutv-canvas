package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"infinite-canvas/backend/internal/app"
	"infinite-canvas/backend/internal/database"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
	"infinite-canvas/backend/internal/service"

	"gorm.io/gorm"
)

// --wipe-canvas-plaza 会删除全部画布项目（canvas_projects）、分享和单元链接，不限用户。
// 它和 --reset-imported 只能在没有 --keep-slug / --delete-others 时执行。
// 同时出现时必须在 database.Open 之前失败，不能进入清库。

const plazaSeedDefaultLimit = 80

type plazaSeedFlags struct {
	keepSlug      string
	keepSlugSet   bool
	deleteOthers  bool
	limit         int
	resetImported bool
	wipe          bool
}

type plazaSeedSession interface {
	Close()
	Wipe() error
	ResetImported() (int, error)
	WorkBySlug(slug string) (id string, status string, found bool, err error)
	DeleteExcept(keepID string) (int, error)
	Seed(items []app.PlazaExternalSeedItem, limit int, skip string) (*app.PlazaSeedReport, error)
}

type plazaSeedDeps struct {
	open func() (plazaSeedSession, error)
}

func runPlazaSeed(_ context.Context) int {
	return executePlazaSeed(os.Args[2:], os.Stdin, os.Stdout, os.Stderr, plazaSeedDeps{open: openPlazaSeedSession})
}

func executePlazaSeed(args []string, stdin io.Reader, stdout, stderr io.Writer, deps plazaSeedDeps) int {
	flags, err := parsePlazaSeedFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	// 新旗标和清库互斥，并且在打开数据库之前拒绝。
	if (flags.keepSlugSet || flags.deleteOthers) && (flags.wipe || flags.resetImported) {
		fmt.Fprintln(stderr, "保留或删除旗标不能与清库、重置导入同时使用")
		return 1
	}
	if flags.deleteOthers && flags.keepSlug == "" {
		fmt.Fprintln(stderr, "删除其他作品必须同时指定 --keep-slug")
		return 1
	}
	if flags.keepSlugSet && flags.keepSlug == "" {
		fmt.Fprintln(stderr, "保留 slug 不能为空")
		return 1
	}
	if flags.deleteOthers {
		return runDeleteOthersPlazaSeed(flags, stdin, stdout, stderr, deps)
	}
	if flags.keepSlugSet {
		return runKeepSlugPlazaSeed(flags, stdin, stdout, stderr, deps)
	}
	return runLegacyPlazaSeed(flags, stdin, stdout, stderr, deps)
}

func parsePlazaSeedFlags(args []string) (plazaSeedFlags, error) {
	flags := plazaSeedFlags{limit: plazaSeedDefaultLimit}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--delete-others":
			flags.deleteOthers = true
		case arg == "--reset-imported":
			flags.resetImported = true
		case arg == "--wipe-canvas-plaza":
			flags.wipe = true
		case arg == "--keep-slug" || strings.HasPrefix(arg, "--keep-slug="):
			value, next, err := takeFlagValue(args, i, "--keep-slug")
			if err != nil {
				return flags, err
			}
			flags.keepSlug = value
			flags.keepSlugSet = true
			i = next
		case arg == "--limit" || strings.HasPrefix(arg, "--limit="):
			value, next, err := takeFlagValue(args, i, "--limit")
			if err != nil {
				return flags, err
			}
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed <= 0 {
				return flags, errors.New("--limit 必须是正整数")
			}
			flags.limit = parsed
			i = next
		default:
			return flags, fmt.Errorf("未知参数: %s", arg)
		}
	}
	return flags, nil
}

func takeFlagValue(args []string, index int, name string) (string, int, error) {
	arg := args[index]
	prefix := name + "="
	if strings.HasPrefix(arg, prefix) {
		return strings.TrimSpace(arg[len(prefix):]), index, nil
	}
	if arg == name {
		if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
			return "", index, fmt.Errorf("%s 缺少取值", name)
		}
		return strings.TrimSpace(args[index+1]), index + 1, nil
	}
	return "", index, fmt.Errorf("未知参数: %s", arg)
}

func runLegacyPlazaSeed(flags plazaSeedFlags, stdin io.Reader, stdout, stderr io.Writer, deps plazaSeedDeps) int {
	session, err := deps.open()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer session.Close()
	if flags.wipe {
		if err := session.Wipe(); err != nil {
			fmt.Fprintf(stderr, "清库失败: %v\n", err)
			return 1
		}
		fmt.Fprintln(stderr, "wiped canvas_projects and plaza tables")
	} else if flags.resetImported {
		removed, err := session.ResetImported()
		if err != nil {
			fmt.Fprintf(stderr, "清理导入作品失败: %v\n", err)
			return 1
		}
		fmt.Fprintf(stderr, "reset imported plaza works: %d\n", removed)
	}
	items, err := decodeSeedItems(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "读取导入清单失败: %v\n", err)
		return 1
	}
	report, err := session.Seed(items, flags.limit, "")
	if err != nil {
		fmt.Fprintf(stderr, "导入失败: %v\n", err)
		return 1
	}
	if err := writeSeedReport(stdout, report); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func runKeepSlugPlazaSeed(flags plazaSeedFlags, stdin io.Reader, stdout, stderr io.Writer, deps plazaSeedDeps) int {
	items, err := decodeSeedItems(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "读取导入清单失败: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		report := &app.PlazaSeedReport{KeptSlug: flags.keepSlug}
		if err := writeSeedReport(stdout, report); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		return 0
	}
	session, err := deps.open()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer session.Close()
	report, err := session.Seed(items, flags.limit, flags.keepSlug)
	if err != nil {
		fmt.Fprintf(stderr, "导入失败: %v\n", err)
		return 1
	}
	if report == nil {
		report = &app.PlazaSeedReport{}
	}
	report.Deleted = 0
	report.KeptSlug = flags.keepSlug
	report.Shortfall = 0
	if err := writeSeedReport(stdout, report); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func runDeleteOthersPlazaSeed(flags plazaSeedFlags, stdin io.Reader, stdout, stderr io.Writer, deps plazaSeedDeps) int {
	items, err := decodeSeedItems(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "读取导入清单失败: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(stderr, "导入清单为空")
		return 1
	}
	deduped := dedupeSeedItems(items)
	others := 0
	for _, item := range deduped {
		if !sameSeedUUID(item.UUID, flags.keepSlug) {
			others++
		}
	}
	if others < flags.limit {
		fmt.Fprintf(stderr, "去掉保留作品后候选不足 %d 个\n", flags.limit)
		return 1
	}
	session, err := deps.open()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer session.Close()
	keepID, status, found, err := session.WorkBySlug(flags.keepSlug)
	if err != nil {
		fmt.Fprintf(stderr, "读取保留作品失败: %v\n", err)
		return 1
	}
	if !found {
		fmt.Fprintln(stderr, "保留作品不存在")
		return 1
	}
	if status != model.PlazaWorkListed {
		fmt.Fprintln(stderr, "保留作品不是上架状态")
		return 1
	}
	// 只有 --delete-others 会调用 deletePlazaWorks。再次运行会先删掉上次留下的半成品，这是故意的。
	deleted, err := session.DeleteExcept(keepID)
	if err != nil {
		fmt.Fprintf(stderr, "删除广场作品失败: %v\n", err)
		return 1
	}
	report, err := session.Seed(deduped, flags.limit, flags.keepSlug)
	if err != nil {
		fmt.Fprintf(stderr, "导入失败: %v\n", err)
		return 1
	}
	if report == nil {
		report = &app.PlazaSeedReport{}
	}
	report.Deleted = deleted
	report.KeptSlug = flags.keepSlug
	code := 0
	if report.Imported < flags.limit {
		report.Shortfall = flags.limit - report.Imported
		code = 2
	}
	if err := writeSeedReport(stdout, report); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return code
}

func decodeSeedItems(stdin io.Reader) ([]app.PlazaExternalSeedItem, error) {
	if stdin == nil {
		return nil, errors.New("缺少导入清单")
	}
	var items []app.PlazaExternalSeedItem
	if err := json.NewDecoder(stdin).Decode(&items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []app.PlazaExternalSeedItem{}
	}
	return items, nil
}

func dedupeSeedItems(items []app.PlazaExternalSeedItem) []app.PlazaExternalSeedItem {
	type slot struct {
		index int
		item  app.PlazaExternalSeedItem
		rich  bool
	}
	order := make([]string, 0, len(items))
	slots := map[string]*slot{}
	for _, item := range items {
		uuid := strings.ToLower(strings.TrimSpace(item.UUID))
		if uuid == "" {
			continue
		}
		item.UUID = uuid
		rich := strings.TrimSpace(item.Title) != "" && strings.TrimSpace(item.CoverURL) != ""
		if existing, ok := slots[uuid]; ok {
			if !existing.rich && rich {
				existing.item = item
				existing.rich = true
			}
			continue
		}
		slots[uuid] = &slot{index: len(order), item: item, rich: rich}
		order = append(order, uuid)
	}
	out := make([]app.PlazaExternalSeedItem, len(order))
	for _, uuid := range order {
		out[slots[uuid].index] = slots[uuid].item
	}
	return out
}

func sameSeedUUID(left, right string) bool {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	return left != "" && left == right
}

func writeSeedReport(stdout io.Writer, report *app.PlazaSeedReport) error {
	if report == nil {
		report = &app.PlazaSeedReport{}
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

type dbPlazaSeedSession struct {
	svc   *service.Service
	repo  *repository.Repository
	sqlDB *sql.DB
}

func openPlazaSeedSession() (plazaSeedSession, error) {
	dataDir := env("CANVAS_BACKEND_DATA_DIR", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	db, err := database.Open(database.Config{
		Driver:  env("CANVAS_DATABASE_DRIVER", "sqlite"),
		DSN:     os.Getenv("DATABASE_URL"),
		DataDir: dataDir,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	repo := repository.New(db)
	return &dbPlazaSeedSession{svc: service.New(repo, dataDir), repo: repo, sqlDB: sqlDB}, nil
}

func (s *dbPlazaSeedSession) Close() {
	if s == nil {
		return
	}
	if s.svc != nil {
		_ = s.svc.Close()
	}
	if s.sqlDB != nil {
		_ = s.sqlDB.Close()
	}
}

func (s *dbPlazaSeedSession) Wipe() error {
	return s.svc.WipeCanvasAndPlaza()
}

func (s *dbPlazaSeedSession) ResetImported() (int, error) {
	return s.svc.ResetImportedPlazaWorks()
}

func (s *dbPlazaSeedSession) WorkBySlug(slug string) (string, string, bool, error) {
	work, err := s.repo.PlazaWorkBySlug(strings.TrimSpace(slug))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return work.ID, work.Status, true, nil
}

func (s *dbPlazaSeedSession) DeleteExcept(keepID string) (int, error) {
	return s.svc.DeletePlazaWorksExcept(keepID)
}

func (s *dbPlazaSeedSession) Seed(items []app.PlazaExternalSeedItem, limit int, skip string) (*app.PlazaSeedReport, error) {
	return s.svc.SeedPlazaExternal(items, limit, skip)
}
