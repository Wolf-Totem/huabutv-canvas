package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/app"
)

func TestPlazaSeedLegacyShortFileDoesNotDelete(t *testing.T) {
	ten := make([]app.PlazaExternalSeedItem, 10)
	for i := range ten {
		ten[i] = app.PlazaExternalSeedItem{UUID: "u" + string(rune('a'+i)), Slug: "s", Title: "公开画布导入作品标题"}
	}
	raw, err := json.Marshal(ten)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{string(raw), "[]"} {
		var opened, deleted, seeded int
		var gotLimit int
		var gotSkip string
		code := executePlazaSeed(nil, strings.NewReader(body), &bytes.Buffer{}, &bytes.Buffer{}, plazaSeedDeps{
			open: func() (plazaSeedSession, error) {
				opened++
				return &fakePlazaSeedSession{
					seedFn: func(items []app.PlazaExternalSeedItem, limit int, skip string) (*app.PlazaSeedReport, error) {
						seeded++
						gotLimit = limit
						gotSkip = skip
						return &app.PlazaSeedReport{Imported: len(items)}, nil
					},
					deleteFn: func(string) (int, error) {
						deleted++
						return 1, nil
					},
				}, nil
			},
		})
		if code != 0 || opened != 1 || deleted != 0 || seeded != 1 || gotLimit != 80 || gotSkip != "" {
			t.Fatalf("body %s code=%d opened=%d deleted=%d seeded=%d limit=%d skip=%q", body, code, opened, deleted, seeded, gotLimit, gotSkip)
		}
	}
}

func TestPlazaSeedKeepSlugAloneDoesNotDelete(t *testing.T) {
	const keep = "0251b9ae0e304f7fb96e353eecfe2204"
	items := make([]app.PlazaExternalSeedItem, 0, 6)
	items = append(items, app.PlazaExternalSeedItem{UUID: keep, Slug: keep, Title: "公开画布导入作品标题"})
	for i := 0; i < 5; i++ {
		uuid := "candidate-" + string(rune('a'+i))
		items = append(items, app.PlazaExternalSeedItem{UUID: uuid, Slug: uuid, Title: "公开画布导入作品标题"})
	}
	raw, _ := json.Marshal(items)
	var deleted int
	var gotLimit int
	var gotSkip string
	stdout := &bytes.Buffer{}
	code := executePlazaSeed([]string{"--keep-slug=" + keep, "--limit=2"}, bytes.NewReader(raw), stdout, &bytes.Buffer{}, plazaSeedDeps{
		open: func() (plazaSeedSession, error) {
			return &fakePlazaSeedSession{
				workFn: func(slug string) (string, string, bool, error) {
					if slug != keep {
						t.Fatalf("lookup %s", slug)
					}
					return "keep-id", "listed", true, nil
				},
				deleteFn: func(string) (int, error) {
					deleted++
					return 4, nil
				},
				seedFn: func(_ []app.PlazaExternalSeedItem, limit int, skip string) (*app.PlazaSeedReport, error) {
					gotLimit = limit
					gotSkip = skip
					return &app.PlazaSeedReport{Imported: 1}, nil
				},
			}, nil
		},
	})
	var report app.PlazaSeedReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if code != 0 || deleted != 0 || gotLimit != 2 || gotSkip != keep || report.Deleted != 0 || report.Shortfall != 0 || report.KeptSlug != keep {
		t.Fatalf("code=%d deleted=%d limit=%d skip=%q report=%#v", code, deleted, gotLimit, gotSkip, report)
	}
}

func TestPlazaSeedDeleteOthersMissingOrUnlistedDoesNotDelete(t *testing.T) {
	body := mustSeedJSON(t, []app.PlazaExternalSeedItem{
		{UUID: "other-1", Slug: "other-1", Title: "公开画布导入作品标题"},
	})
	cases := []struct {
		name   string
		status string
		found  bool
	}{
		{name: "missing", found: false},
		{name: "taken down", status: "taken_down", found: true},
		{name: "unlisted", status: "unlisted", found: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var opened, deleted, seeded int
			code := executePlazaSeed([]string{"--delete-others", "--keep-slug=keep-slug", "--limit=1"}, strings.NewReader(body), &bytes.Buffer{}, &bytes.Buffer{}, plazaSeedDeps{
				open: func() (plazaSeedSession, error) {
					opened++
					return &fakePlazaSeedSession{
						workFn: func(string) (string, string, bool, error) {
							return "keep-id", tc.status, tc.found, nil
						},
						deleteFn: func(string) (int, error) {
							deleted++
							return 3, nil
						},
						seedFn: func([]app.PlazaExternalSeedItem, int, string) (*app.PlazaSeedReport, error) {
							seeded++
							return &app.PlazaSeedReport{Imported: 1}, nil
						},
					}, nil
				},
			})
			if code != 1 || opened != 1 || deleted != 0 || seeded != 0 {
				t.Fatalf("code=%d opened=%d deleted=%d seeded=%d", code, opened, deleted, seeded)
			}
		})
	}
}

func TestPlazaSeedDeleteOthersRejectsBeforeOpen(t *testing.T) {
	cases := []struct {
		name string
		args []string
		body string
	}{
		{name: "wipe with keep", args: []string{"--keep-slug=abc", "--wipe-canvas-plaza"}, body: "[]"},
		{name: "reset with keep", args: []string{"--reset-imported", "--keep-slug=abc"}, body: "[]"},
		{name: "wipe with delete", args: []string{"--delete-others", "--keep-slug=abc", "--wipe-canvas-plaza"}, body: "[]"},
		{name: "reset with delete", args: []string{"--delete-others", "--reset-imported", "--keep-slug=abc"}, body: "[]"},
		{name: "delete without keep", args: []string{"--delete-others"}, body: "[]"},
		{name: "empty stdin", args: []string{"--delete-others", "--keep-slug=keep"}, body: ""},
		{name: "empty array", args: []string{"--delete-others", "--keep-slug=keep", "--limit=1"}, body: "[]"},
		{name: "bad json", args: []string{"--delete-others", "--keep-slug=keep"}, body: "{"},
		{name: "too few candidates", args: []string{"--delete-others", "--keep-slug=keep", "--limit=3"}, body: mustSeedJSON(t, []app.PlazaExternalSeedItem{
			{UUID: "keep", Slug: "keep", Title: "公开画布导入作品标题"},
			{UUID: "only-one", Slug: "only-one", Title: "公开画布导入作品标题"},
			{UUID: "ONLY-ONE", Slug: "dup", Title: "后来的完整标题", CoverURL: "https://example.com/a.jpg"},
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var opened, deleted int
			code := executePlazaSeed(tc.args, strings.NewReader(tc.body), &bytes.Buffer{}, &bytes.Buffer{}, plazaSeedDeps{
				open: func() (plazaSeedSession, error) {
					opened++
					return &fakePlazaSeedSession{
						deleteFn: func(string) (int, error) {
							deleted++
							return 1, nil
						},
					}, nil
				},
			})
			if code != 1 || opened != 0 || deleted != 0 {
				t.Fatalf("code=%d opened=%d deleted=%d", code, opened, deleted)
			}
		})
	}
}

func TestPlazaSeedDeleteOthersShortfallExits2(t *testing.T) {
	body := mustSeedJSON(t, []app.PlazaExternalSeedItem{
		{UUID: "keep-slug", Slug: "keep-slug", Title: "公开画布导入作品标题"},
		{UUID: "new-1", Slug: "new-1", Title: "公开画布导入作品标题"},
		{UUID: "new-2", Slug: "new-2", Title: "公开画布导入作品标题"},
		{UUID: "new-3", Slug: "new-3", Title: "公开画布导入作品标题"},
	})
	var deletedID string
	stdout := &bytes.Buffer{}
	code := executePlazaSeed([]string{"--delete-others", "--keep-slug=keep-slug", "--limit=3"}, strings.NewReader(body), stdout, &bytes.Buffer{}, plazaSeedDeps{
		open: func() (plazaSeedSession, error) {
			return &fakePlazaSeedSession{
				workFn: func(string) (string, string, bool, error) {
					return "keep-id", "listed", true, nil
				},
				deleteFn: func(id string) (int, error) {
					deletedID = id
					return 4, nil
				},
				seedFn: func(_ []app.PlazaExternalSeedItem, limit int, skip string) (*app.PlazaSeedReport, error) {
					if limit != 3 || skip != "keep-slug" {
						t.Fatalf("seed limit=%d skip=%s", limit, skip)
					}
					return &app.PlazaSeedReport{Imported: 1, Failed: 2}, nil
				},
			}, nil
		},
	})
	var report app.PlazaSeedReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if code != 2 || deletedID != "keep-id" || report.Deleted != 4 || report.Imported != 1 || report.Shortfall != 2 || report.KeptSlug != "keep-slug" {
		t.Fatalf("code=%d deletedID=%s report=%#v", code, deletedID, report)
	}
}

func TestDedupeSeedItemsPrefersFirstRichRecord(t *testing.T) {
	got := dedupeSeedItems([]app.PlazaExternalSeedItem{
		{UUID: " AbC ", Title: "短"},
		{UUID: "abc", Title: "完整标题", CoverURL: "https://example.com/a.jpg"},
		{UUID: "ABC", Title: "更后的标题", CoverURL: "https://example.com/b.jpg"},
		{UUID: " ", Title: "空"},
		{UUID: "second", Title: "第二条", CoverURL: "https://example.com/c.jpg"},
	})
	if len(got) != 2 || got[0].UUID != "abc" || got[0].Title != "完整标题" || got[1].UUID != "second" {
		t.Fatalf("%#v", got)
	}
}

type fakePlazaSeedSession struct {
	wipeFn   func() error
	resetFn  func() (int, error)
	workFn   func(slug string) (string, string, bool, error)
	deleteFn func(keepID string) (int, error)
	seedFn   func(items []app.PlazaExternalSeedItem, limit int, skip string) (*app.PlazaSeedReport, error)
}

func (f *fakePlazaSeedSession) Close() {}

func (f *fakePlazaSeedSession) Wipe() error {
	if f != nil && f.wipeFn != nil {
		return f.wipeFn()
	}
	return nil
}

func (f *fakePlazaSeedSession) ResetImported() (int, error) {
	if f != nil && f.resetFn != nil {
		return f.resetFn()
	}
	return 0, nil
}

func (f *fakePlazaSeedSession) WorkBySlug(slug string) (string, string, bool, error) {
	if f != nil && f.workFn != nil {
		return f.workFn(slug)
	}
	return "", "", false, nil
}

func (f *fakePlazaSeedSession) DeleteExcept(keepID string) (int, error) {
	if f != nil && f.deleteFn != nil {
		return f.deleteFn(keepID)
	}
	return 0, nil
}

func (f *fakePlazaSeedSession) Seed(items []app.PlazaExternalSeedItem, limit int, skip string) (*app.PlazaSeedReport, error) {
	if f != nil && f.seedFn != nil {
		return f.seedFn(items, limit, skip)
	}
	return &app.PlazaSeedReport{}, nil
}

func mustSeedJSON(t *testing.T, items []app.PlazaExternalSeedItem) string {
	t.Helper()
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
