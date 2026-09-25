package repository

import (
	"errors"
	"strings"

	"infinite-canvas/backend/internal/model"
)

var errMissingPlazaKeepID = errors.New("缺少保留作品")

// PlazaDeletedWork is one plaza work removed by DeletePlazaWorks.
// SystemAssetIDs are plaza-system resources referenced by the work or its snapshots.
// Object bytes are left for cleanupDetachedUserResources.
type PlazaDeletedWork struct {
	ID              string
	Slug            string
	Title           string
	SourceProjectID string
	SystemAssetIDs  []string
}

// PlazaWorksDeleteResult is the outcome of one deletePlazaWorks transaction.
type PlazaWorksDeleteResult struct {
	Deleted int
	Works   []PlazaDeletedWork
}

// PlazaWorkIDsExcept returns every plaza work id except keepID.
// An empty keepID is refused so a caller cannot wipe the table by accident.
func (r *Repository) PlazaWorkIDsExcept(keepID string) ([]string, error) {
	keepID = strings.TrimSpace(keepID)
	if keepID == "" {
		return nil, errMissingPlazaKeepID
	}
	var ids []string
	err := r.db.Model(&model.PlazaWork{}).Where("id <> ?", keepID).Pluck("id", &ids).Error
	return ids, err
}

// DeletePlazaWorks removes the given works in one transaction: likes, events,
// tags, snapshot assets, snapshots, blanked application work ids, the work rows,
// and the author's plaza-{uuid} canvas for source_project_id ext:{uuid}.
// It does not delete resource bytes or any other canvas project.
func (r *Repository) DeletePlazaWorks(ids []string) (PlazaWorksDeleteResult, error) {
	var result PlazaWorksDeleteResult
	ids = uniqueNonEmpty(ids)
	if len(ids) == 0 {
		return result, nil
	}
	err := r.Transaction(func(tx *Repository) error {
		return deletePlazaWorks(tx, ids, &result)
	})
	if err != nil {
		return PlazaWorksDeleteResult{}, err
	}
	return result, nil
}

func deletePlazaWorks(tx *Repository, ids []string, result *PlazaWorksDeleteResult) error {
	var works []model.PlazaWork
	if err := tx.db.Where("id IN ?", ids).Find(&works).Error; err != nil {
		return err
	}
	if len(works) == 0 {
		return nil
	}
	foundIDs := make([]string, 0, len(works))
	perWork := make(map[string]*stringSet, len(works))
	allAssets := newStringSet()
	for _, work := range works {
		foundIDs = append(foundIDs, work.ID)
		set := newStringSet()
		set.add(work.CoverAssetID)
		set.add(work.WatchAssetID)
		perWork[work.ID] = set
		allAssets.add(work.CoverAssetID)
		allAssets.add(work.WatchAssetID)
	}

	var snapshots []model.PlazaSnapshot
	if err := tx.db.Where("work_id IN ?", foundIDs).Find(&snapshots).Error; err != nil {
		return err
	}
	snapshotIDs := make([]string, 0, len(snapshots))
	snapshotWork := make(map[string]string, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotIDs = append(snapshotIDs, snapshot.ID)
		snapshotWork[snapshot.ID] = snapshot.WorkID
		if set := perWork[snapshot.WorkID]; set != nil {
			set.add(snapshot.CoverAssetID)
			set.add(snapshot.WatchAssetID)
		}
		allAssets.add(snapshot.CoverAssetID)
		allAssets.add(snapshot.WatchAssetID)
	}
	if len(snapshotIDs) > 0 {
		var assets []model.PlazaSnapshotAsset
		if err := tx.db.Where("snapshot_id IN ?", snapshotIDs).Find(&assets).Error; err != nil {
			return err
		}
		for _, asset := range assets {
			workID := snapshotWork[asset.SnapshotID]
			if set := perWork[workID]; set != nil {
				set.add(asset.AssetID)
			}
			allAssets.add(asset.AssetID)
		}
	}

	systemIDs := map[string]struct{}{}
	if assetIDs := allAssets.items(); len(assetIDs) > 0 {
		var resources []model.Resource
		if err := tx.db.Where("id IN ? AND user_id = ?", assetIDs, model.PlazaSystemUserID).Find(&resources).Error; err != nil {
			return err
		}
		for _, resource := range resources {
			systemIDs[resource.ID] = struct{}{}
		}
	}

	if err := tx.db.Where("work_id IN ?", foundIDs).Delete(&model.PlazaLike{}).Error; err != nil {
		return err
	}
	if err := tx.db.Where("work_id IN ?", foundIDs).Delete(&model.PlazaEvent{}).Error; err != nil {
		return err
	}
	if err := tx.db.Where("work_id IN ?", foundIDs).Delete(&model.PlazaWorkTag{}).Error; err != nil {
		return err
	}
	if len(snapshotIDs) > 0 {
		if err := tx.db.Where("snapshot_id IN ?", snapshotIDs).Delete(&model.PlazaSnapshotAsset{}).Error; err != nil {
			return err
		}
	}
	if err := tx.db.Where("work_id IN ?", foundIDs).Delete(&model.PlazaSnapshot{}).Error; err != nil {
		return err
	}
	for _, work := range works {
		canvasID, ok := plazaImportedCanvasID(work.SourceProjectID)
		if !ok || strings.TrimSpace(work.AuthorID) == "" {
			continue
		}
		if err := deleteOwnedPlazaCanvas(tx, work.AuthorID, canvasID); err != nil {
			return err
		}
	}
	if err := tx.db.Model(&model.PlazaApplication{}).Where("work_id IN ?", foundIDs).Update("work_id", "").Error; err != nil {
		return err
	}
	if err := tx.db.Where("id IN ?", foundIDs).Delete(&model.PlazaWork{}).Error; err != nil {
		return err
	}

	result.Deleted = len(works)
	result.Works = make([]PlazaDeletedWork, 0, len(works))
	for _, work := range works {
		item := PlazaDeletedWork{
			ID:              work.ID,
			Slug:            work.Slug,
			Title:           work.Title,
			SourceProjectID: work.SourceProjectID,
		}
		if set := perWork[work.ID]; set != nil {
			for _, assetID := range set.items() {
				if _, ok := systemIDs[assetID]; ok {
					item.SystemAssetIDs = append(item.SystemAssetIDs, assetID)
				}
			}
		}
		result.Works = append(result.Works, item)
	}
	return nil
}

// deleteOwnedPlazaCanvas runs the same four statements as Repository.DeleteCanvasProject
// on the current transaction, limited to this author and this canvas id.
func deleteOwnedPlazaCanvas(tx *Repository, userID, canvasID string) error {
	if err := tx.db.Where("user_id = ? AND project_id = ?", userID, canvasID).Delete(&model.CanvasShare{}).Error; err != nil {
		return err
	}
	if err := tx.db.Where("canvas_id = ?", canvasID).Delete(&model.CanvasUnitLink{}).Error; err != nil {
		return err
	}
	if err := tx.db.Model(&model.Task{}).Where("user_id = ? AND project_id = ?", userID, canvasID).Update("project_id", "").Error; err != nil {
		return err
	}
	return tx.db.Delete(&model.CanvasProject{}, "id = ? AND user_id = ?", canvasID, userID).Error
}

func plazaImportedCanvasID(sourceProjectID string) (string, bool) {
	source := strings.TrimSpace(sourceProjectID)
	const prefix = "ext:"
	if !strings.HasPrefix(source, prefix) {
		return "", false
	}
	uuid := strings.TrimSpace(strings.TrimPrefix(source, prefix))
	if uuid == "" || strings.ContainsAny(uuid, " \t\r\n/\\") {
		return "", false
	}
	return "plaza-" + uuid, true
}

type stringSet struct {
	order []string
	seen  map[string]struct{}
}

func newStringSet() *stringSet {
	return &stringSet{seen: map[string]struct{}{}}
}

func (s *stringSet) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := s.seen[value]; ok {
		return
	}
	s.seen[value] = struct{}{}
	s.order = append(s.order, value)
}

func (s *stringSet) items() []string {
	if s == nil {
		return nil
	}
	return s.order
}

func uniqueNonEmpty(values []string) []string {
	set := newStringSet()
	for _, value := range values {
		set.add(value)
	}
	return set.items()
}
