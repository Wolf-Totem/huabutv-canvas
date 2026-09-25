package app

import (
	"encoding/json"
	"strings"
	"time"

	"infinite-canvas/backend/internal/auth"
	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/plaza"
)

type PlazaExternalSeedItem struct {
	UUID        string `json:"uuid"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	CategoryID  string `json:"categoryId"`
	AuthorID    string `json:"authorId"`
	CoverURL    string `json:"coverUrl"`
	WatchURL    string `json:"watchUrl"`
	ProjectUUID string `json:"projectUuid"`
	DisplayOnly bool   `json:"displayOnly"`
}

type PlazaSeedReport struct {
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Reset    int      `json:"reset,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

func (s *Service) ResetImportedPlazaWorks() (int, error) {
	return s.repo.DeleteImportedPlazaWorks()
}

func (s *Service) WipeCanvasAndPlaza() error {
	return s.repo.WipeCanvasAndPlaza()
}

func (s *Service) AdminSeedPlazaExternal(actor *model.User, items []PlazaExternalSeedItem) (*PlazaSeedReport, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.SeedPlazaExternal(items)
}

func (s *Service) SeedPlazaExternal(items []PlazaExternalSeedItem) (*PlazaSeedReport, error) {
	report := &PlazaSeedReport{}
	adminID := ""
	if admin, err := s.repo.FirstAdmin(); err == nil && admin != nil {
		adminID = admin.ID
	}
	for _, item := range items {
		if report.Imported >= 80 {
			break
		}
		item.UUID = strings.TrimSpace(item.UUID)
		item.Slug = strings.TrimSpace(item.Slug)
		item.Title = strings.TrimSpace(item.Title)
		item.AuthorID = strings.TrimSpace(item.AuthorID)
		if item.AuthorID == "" || item.AuthorID == "plaza-demo" {
			item.AuthorID = adminID
		}
		if item.AuthorID == "" {
			item.AuthorID = "plaza-demo"
		}
		if item.CategoryID == "" {
			item.CategoryID = "plaza-cat-featured"
		}
		if item.UUID == "" || item.Slug == "" || item.Title == "" {
			report.Failed++
			report.Errors = append(report.Errors, item.Slug+": 缺少 uuid/slug/title")
			continue
		}
		imported, err := s.fetchSeedCanvas(item)
		if err != nil || imported == nil || imported.ImportedConnectionCount < 1 || imported.ImportedNodeCount < 3 {
			report.Failed++
			if err != nil {
				report.Errors = append(report.Errors, item.Slug+": "+err.Error())
			} else {
				report.Errors = append(report.Errors, item.Slug+": 没有制作过程")
			}
			continue
		}
		if name := strings.TrimSpace(imported.ProjectName); name != "" && (item.Title == "" || item.Title == item.UUID || len([]rune(item.Title)) <= 8) {
			item.Title = name
		}
		doc := canvasDocumentFromLibTV(item.Title, imported)
		if err := s.saveImportedCanvasProject(item.AuthorID, item.Title, doc); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, item.Slug+": "+err.Error())
			continue
		}
		process := true
		if err := s.plazaDomain().SaveImportedDocument(plaza.ExternalSeedItem{
			UUID: item.UUID, Slug: item.Slug, Title: item.Title, Subtitle: item.Subtitle,
			CategoryID: item.CategoryID, AuthorID: item.AuthorID,
			CoverURL: item.CoverURL, WatchURL: item.WatchURL,
			AllowProcessView: process, AllowCopy: process,
		}, doc); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, item.Slug+": "+err.Error())
			continue
		}
		report.Imported++
	}
	return report, nil
}

func (s *Service) fetchSeedCanvas(item PlazaExternalSeedItem) (*auth.LibTVImportResult, error) {
	tried := map[string]struct{}{}
	for _, uuid := range []string{item.UUID, item.ProjectUUID} {
		uuid = strings.TrimSpace(uuid)
		if uuid == "" {
			continue
		}
		if _, ok := tried[uuid]; ok {
			continue
		}
		tried[uuid] = struct{}{}
		imported, err := s.auth.FetchPublicGraph(uuid)
		if err == nil && imported != nil && imported.ImportedNodeCount >= 3 && imported.ImportedConnectionCount >= 1 {
			return imported, nil
		}
	}
	return nil, kernel.NotFound("画布不可导入")
}

func (s *Service) saveImportedCanvasProject(userID, title string, doc map[string]any) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "plaza-demo" {
		return nil
	}
	now := time.Now()
	id := kernel.StringValue(doc["id"])
	if id == "" {
		id = kernel.NewID()
		doc["id"] = id
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return s.repo.UpsertCanvasProject(&model.CanvasProject{
		ID:          id,
		UserID:      userID,
		Title:       title,
		PayloadJSON: string(payload),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func canvasDocumentFromLibTV(title string, imported *auth.LibTVImportResult) map[string]any {
	nodes := make([]any, 0, len(imported.Nodes))
	for _, node := range imported.Nodes {
		kind := "image"
		if strings.EqualFold(node.Type, "video") {
			kind = "video"
		} else if strings.EqualFold(node.Type, "text") {
			kind = "text"
		}
		nodes = append(nodes, map[string]any{
			"id":       node.ID,
			"type":     kind,
			"title":    node.Title,
			"position": map[string]any{"x": node.X, "y": node.Y},
			"width":    node.Width,
			"height":   node.Height,
			"metadata": map[string]any{
				"content":      node.Content,
				"prompt":       node.Prompt,
				"model":        node.Model,
				"status":       node.Status,
				"errorDetails": node.ErrorDetails,
				"importSource": node.Metadata,
			},
		})
	}
	connections := make([]any, 0, len(imported.Connections))
	for _, conn := range imported.Connections {
		connections = append(connections, map[string]any{
			"id":         conn.ID,
			"fromNodeId": conn.FromNodeID,
			"toNodeId":   conn.ToNodeID,
		})
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"id":             "plaza-" + imported.ProjectUUID,
		"title":          title,
		"createdAt":      now,
		"updatedAt":      now,
		"nodes":          nodes,
		"connections":    connections,
		"chatSessions":   []any{},
		"activeChatId":   nil,
		"backgroundMode": "lines",
		"showImageInfo":  false,
		"viewport":       map[string]any{"x": 40, "y": 20, "k": 0.35},
		"directorScenes": []any{},
	}
}
