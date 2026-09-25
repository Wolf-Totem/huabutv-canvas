package app

import (
	"strings"
	"time"

	"infinite-canvas/backend/internal/auth"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/plaza"
)

type PlazaExternalSeedItem struct {
	UUID       string `json:"uuid"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	CategoryID string `json:"categoryId"`
	AuthorID   string `json:"authorId"`
}

type PlazaSeedReport struct {
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

func (s *Service) AdminSeedPlazaExternal(actor *model.User, items []PlazaExternalSeedItem) (*PlazaSeedReport, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.SeedPlazaExternal(items)
}

func (s *Service) SeedPlazaExternal(items []PlazaExternalSeedItem) (*PlazaSeedReport, error) {
	report := &PlazaSeedReport{}
	for _, item := range items {
		item.UUID = strings.TrimSpace(item.UUID)
		item.Slug = strings.TrimSpace(item.Slug)
		item.Title = strings.TrimSpace(item.Title)
		item.AuthorID = strings.TrimSpace(item.AuthorID)
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
		imported, err := s.auth.FetchLibTV(item.UUID)
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, item.Slug+": "+err.Error())
			continue
		}
		if name := strings.TrimSpace(imported.ProjectName); name != "" && (item.Title == "" || item.Title == item.UUID || len([]rune(item.Title)) <= 8) {
			item.Title = name
		}
		doc := canvasDocumentFromLibTV(item.Title, imported)
		if err := s.plazaDomain().SaveImportedDocument(plaza.ExternalSeedItem{
			UUID: item.UUID, Slug: item.Slug, Title: item.Title, Subtitle: item.Subtitle,
			CategoryID: item.CategoryID, AuthorID: item.AuthorID,
		}, doc); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, item.Slug+": "+err.Error())
			continue
		}
		report.Imported++
	}
	return report, nil
}

func canvasDocumentFromLibTV(title string, imported *auth.LibTVImportResult) map[string]any {
	nodes := make([]any, 0, len(imported.Nodes))
	for _, node := range imported.Nodes {
		kind := "image"
		if strings.EqualFold(node.Type, "video") {
			kind = "video"
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
