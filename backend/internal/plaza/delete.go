package plaza

import (
	"strings"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
)

const hardDeleteAuditAssetLimit = 20

// HardDeleteResult is the admin hard-delete payload.
type HardDeleteResult struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
	Slug    string `json:"slug"`
}

// HardDelete removes one plaza work through deletePlazaWorks and records an admin audit.
// Callers that expose this over HTTP must RequireAdmin first. Object bytes are not deleted.
func (s *Service) HardDelete(actor *model.User, workID string) (*HardDeleteResult, error) {
	if s == nil || s.repo == nil {
		return nil, kernel.NewAppError(500, "广场服务未初始化")
	}
	if actor == nil {
		return nil, kernel.Unauthorized("请先登录")
	}
	workID = strings.TrimSpace(workID)
	work, err := s.repo.PlazaWork(workID)
	if err != nil {
		return nil, kernel.NotFound("作品不存在")
	}
	deleted, err := s.repo.DeletePlazaWorks([]string{work.ID})
	if err != nil {
		return nil, err
	}
	if deleted.Deleted == 0 {
		return nil, kernel.NotFound("作品不存在")
	}
	var assetIDs []string
	for _, item := range deleted.Works {
		if item.ID == work.ID {
			assetIDs = item.SystemAssetIDs
			break
		}
	}
	if err := s.host.RecordAdminAudit(actor, "plaza.work.hard_delete", "plaza_work", work.ID, "彻底删除广场作品", hardDeleteAuditMetadata(work, assetIDs)); err != nil {
		return nil, err
	}
	return &HardDeleteResult{Deleted: true, ID: work.ID, Slug: work.Slug}, nil
}

// DeleteExcept deletes every plaza work except keepID. keepID must be non-empty.
// This is the only bulk entry used by plaza-seed --delete-others.
func (s *Service) DeleteExcept(keepID string) (int, error) {
	if s == nil || s.repo == nil {
		return 0, kernel.NewAppError(500, "广场服务未初始化")
	}
	keepID = strings.TrimSpace(keepID)
	if keepID == "" {
		return 0, kernel.NewAppError(400, "缺少保留作品")
	}
	ids, err := s.repo.PlazaWorkIDsExcept(keepID)
	if err != nil {
		return 0, err
	}
	deleted, err := s.repo.DeletePlazaWorks(ids)
	if err != nil {
		return 0, err
	}
	return deleted.Deleted, nil
}

func hardDeleteAuditMetadata(work *model.PlazaWork, assetIDs []string) map[string]any {
	meta := map[string]any{
		"slug":            work.Slug,
		"title":           work.Title,
		"sourceProjectId": work.SourceProjectID,
	}
	if len(assetIDs) == 0 {
		return meta
	}
	total := len(assetIDs)
	shown := assetIDs
	if total > hardDeleteAuditAssetLimit {
		shown = append([]string(nil), assetIDs[:hardDeleteAuditAssetLimit]...)
	}
	meta["systemAssetIds"] = shown
	meta["systemAssetCount"] = total
	return meta
}
