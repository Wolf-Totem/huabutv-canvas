package plaza

import (
	"encoding/json"
	"strings"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
)

type ExternalSeedItem struct {
	UUID       string
	Slug       string
	Title      string
	Subtitle   string
	CategoryID string
	AuthorID   string
	CoverURL   string
	WatchURL   string
}

func (s *Service) SaveImportedDocument(item ExternalSeedItem, doc map[string]any) error {
	if s == nil || s.repo == nil {
		return kernel.NewAppError(500, "广场服务未初始化")
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	now := time.Now()
	source := "ext:" + strings.TrimSpace(item.UUID)
	work, err := s.repo.PlazaWorkBySourceProject(source)
	if err != nil {
		if existing, slugErr := s.repo.PlazaWorkBySlug(item.Slug); slugErr == nil {
			work = existing
			work.SourceProjectID = source
		} else {
			work = &model.PlazaWork{
				ID:              kernel.NewID(),
				Slug:            item.Slug,
				AuthorID:        item.AuthorID,
				SourceProjectID: source,
				CreatedAt:       now,
			}
		}
	}
	snapshot := &model.PlazaSnapshot{
		ID:          kernel.NewID(),
		WorkID:      work.ID,
		PayloadJSON: string(payload),
		PayloadHash: hashPayload(payload),
		NodeCount:   nodeCount(doc),
		MediaCount:  importedMediaCount(doc),
		CreatedAt:   now,
	}
	listed := now
	work.SnapshotID = snapshot.ID
	work.Title = item.Title
	work.Subtitle = item.Subtitle
	work.CategoryID = item.CategoryID
	work.Status = model.PlazaWorkListed
	work.AllowWatch = true
	work.AllowProcessView = true
	work.AllowCopy = true
	if work.BadgesJSON == "" {
		work.BadgesJSON = "[]"
	}
	work.ListedAt = &listed
	work.UpdatedAt = now
	work.Score = workScore(work)
	cover, watch := importedCoverWatch(item, doc)
	if cover != "" {
		work.CoverExternalURL = cover
	}
	if watch != "" {
		work.WatchExternalURL = watch
	}
	return s.repo.Transaction(func(tx *repository.Repository) error {
		if err := tx.Save(work); err != nil {
			return err
		}
		if err := tx.Create(snapshot); err != nil {
			return err
		}
		return tx.ReplacePlazaWorkTags(work.ID, []string{item.CategoryID})
	})
}

func importedCoverWatch(item ExternalSeedItem, doc map[string]any) (string, string) {
	cover := strings.TrimSpace(item.CoverURL)
	watch := strings.TrimSpace(item.WatchURL)
	rawNodes, _ := doc["nodes"].([]any)
	for _, raw := range rawNodes {
		node, _ := raw.(map[string]any)
		meta, _ := node["metadata"].(map[string]any)
		content := strings.TrimSpace(kernel.StringValue(meta["content"]))
		if content == "" || !strings.HasPrefix(content, "http") {
			continue
		}
		switch kernel.StringValue(node["type"]) {
		case "image":
			if cover == "" {
				cover = content
			}
		case "video":
			if watch == "" {
				watch = content
			}
			if cover == "" {
				cover = content
			}
		}
	}
	return cover, watch
}

func importedMediaCount(doc map[string]any) int {
	count := 0
	rawNodes, _ := doc["nodes"].([]any)
	for _, raw := range rawNodes {
		node, _ := raw.(map[string]any)
		switch kernel.StringValue(node["type"]) {
		case "image", "video", "audio":
			count++
		}
	}
	return count
}
