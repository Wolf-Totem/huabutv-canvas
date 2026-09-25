package repository

import (
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) PlazaCategory(id string) (*model.PlazaCategory, error) {
	var item model.PlazaCategory
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PlazaCategoryBySlug(slug string) (*model.PlazaCategory, error) {
	var item model.PlazaCategory
	if err := r.db.First(&item, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PlazaCategories(enabledOnly bool) ([]model.PlazaCategory, error) {
	var items []model.PlazaCategory
	query := r.db.Order("sort asc, created_at asc")
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	err := query.Find(&items).Error
	return items, err
}

func (r *Repository) SavePlazaCategory(item *model.PlazaCategory) error {
	return r.db.Save(item).Error
}

func (r *Repository) PlazaApplication(id string) (*model.PlazaApplication, error) {
	var item model.PlazaApplication
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PlazaApplicationForUser(userID, id string) (*model.PlazaApplication, error) {
	var item model.PlazaApplication
	if err := r.db.First(&item, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PendingPlazaApplication(userID, projectID string) (*model.PlazaApplication, error) {
	var item model.PlazaApplication
	if err := r.db.First(&item, "user_id = ? AND project_id = ? AND status = ?", userID, projectID, model.PlazaApplicationPending).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) CountPlazaApplicationsSince(userID string, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.PlazaApplication{}).Where("user_id = ? AND submitted_at >= ?", userID, since).Count(&count).Error
	return count, err
}

func (r *Repository) ListPlazaApplications(userID, status string, limit, offset int) ([]model.PlazaApplication, int64, error) {
	query := r.db.Model(&model.PlazaApplication{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	var items []model.PlazaApplication
	err := query.Order("submitted_at desc").Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

func (r *Repository) PlazaWork(id string) (*model.PlazaWork, error) {
	var item model.PlazaWork
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PlazaWorkBySlug(slug string) (*model.PlazaWork, error) {
	var item model.PlazaWork
	if err := r.db.First(&item, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PlazaWorkBySourceProject(projectID string) (*model.PlazaWork, error) {
	var item model.PlazaWork
	if err := r.db.First(&item, "source_project_id = ?", projectID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) ListPlazaWorks(authorID, status, categoryID, sort string, limit, offset int) ([]model.PlazaWork, int64, error) {
	query := r.db.Model(&model.PlazaWork{})
	if authorID != "" {
		query = query.Where("author_id = ?", authorID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if categoryID != "" {
		query = query.Where(
			"category_id = ? OR id IN (SELECT work_id FROM plaza_work_tags WHERE category_id = ?)",
			categoryID, categoryID,
		)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 24
	}
	order := "CASE WHEN pinned_at IS NULL THEN 1 ELSE 0 END, pinned_at DESC, listed_at DESC, id DESC"
	if strings.TrimSpace(sort) == "hot" {
		order = "CASE WHEN pinned_at IS NULL THEN 1 ELSE 0 END, pinned_at DESC, score DESC, listed_at DESC, id DESC"
	}
	var items []model.PlazaWork
	err := query.Order(order).Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

func (r *Repository) PlazaSnapshot(id string) (*model.PlazaSnapshot, error) {
	var item model.PlazaSnapshot
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PlazaSnapshotAssets(snapshotID string) ([]model.PlazaSnapshotAsset, error) {
	var items []model.PlazaSnapshotAsset
	err := r.db.Find(&items, "snapshot_id = ?", snapshotID).Error
	return items, err
}

func (r *Repository) PlazaSnapshotAsset(snapshotID, assetID string) (*model.PlazaSnapshotAsset, error) {
	var item model.PlazaSnapshotAsset
	if err := r.db.First(&item, "snapshot_id = ? AND asset_id = ?", snapshotID, assetID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) PlazaLike(workID, userID string) (*model.PlazaLike, error) {
	var item model.PlazaLike
	if err := r.db.First(&item, "work_id = ? AND user_id = ?", workID, userID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) ReplacePlazaWorkTags(workID string, categoryIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("work_id = ?", workID).Delete(&model.PlazaWorkTag{}).Error; err != nil {
			return err
		}
		if len(categoryIDs) == 0 {
			return nil
		}
		rows := make([]model.PlazaWorkTag, 0, len(categoryIDs))
		seen := map[string]bool{}
		for _, categoryID := range categoryIDs {
			categoryID = strings.TrimSpace(categoryID)
			if categoryID == "" || seen[categoryID] {
				continue
			}
			seen[categoryID] = true
			rows = append(rows, model.PlazaWorkTag{WorkID: workID, CategoryID: categoryID})
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

func (r *Repository) UserIdentitiesByUserIDs(ids []string) (map[string]model.UserIdentity, error) {
	out := map[string]model.UserIdentity{}
	if len(ids) == 0 {
		return out, nil
	}
	var identities []model.UserIdentity
	if err := r.db.Where("user_id IN ?", ids).Order("updated_at desc").Find(&identities).Error; err != nil {
		return nil, err
	}
	for _, identity := range identities {
		if _, exists := out[identity.UserID]; exists {
			continue
		}
		out[identity.UserID] = identity
	}
	return out, nil
}

func (r *Repository) PlazaCategoriesByIDs(ids []string) (map[string]model.PlazaCategory, error) {
	out := map[string]model.PlazaCategory{}
	if len(ids) == 0 {
		return out, nil
	}
	var items []model.PlazaCategory
	if err := r.db.Find(&items, "id IN ?", ids).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		out[item.ID] = item
	}
	return out, nil
}

func (r *Repository) LastPlazaEvent(workID, userID, kind string, since time.Time) (*model.PlazaEvent, error) {
	var item model.PlazaEvent
	query := r.db.Where("work_id = ? AND kind = ?", workID, kind)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}
	if err := query.Order("created_at desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) CreatePlazaLike(like *model.PlazaLike) (bool, error) {
	result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like)
	return result.RowsAffected > 0, result.Error
}

func (r *Repository) DeletePlazaLike(workID, userID string) error {
	return r.db.Delete(&model.PlazaLike{}, "work_id = ? AND user_id = ?", workID, userID).Error
}

func (r *Repository) FirstAdmin() (*model.User, error) {
	var user model.User
	if err := r.db.Where("role = ?", model.UserRoleAdmin).Order("created_at asc").First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) DeleteImportedPlazaWorks() (int, error) {
	var works []model.PlazaWork
	if err := r.db.Where("source_project_id LIKE ?", "ext:%").Find(&works).Error; err != nil {
		return 0, err
	}
	if len(works) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(works))
	for _, work := range works {
		ids = append(ids, work.ID)
	}
	err := r.Transaction(func(tx *Repository) error {
		if err := tx.db.Where("work_id IN ?", ids).Delete(&model.PlazaLike{}).Error; err != nil {
			return err
		}
		if err := tx.db.Where("work_id IN ?", ids).Delete(&model.PlazaEvent{}).Error; err != nil {
			return err
		}
		if err := tx.db.Where("work_id IN ?", ids).Delete(&model.PlazaWorkTag{}).Error; err != nil {
			return err
		}
		var snapshots []model.PlazaSnapshot
		if err := tx.db.Where("work_id IN ?", ids).Find(&snapshots).Error; err != nil {
			return err
		}
		snapshotIDs := make([]string, 0, len(snapshots))
		for _, snapshot := range snapshots {
			snapshotIDs = append(snapshotIDs, snapshot.ID)
		}
		if len(snapshotIDs) > 0 {
			if err := tx.db.Where("snapshot_id IN ?", snapshotIDs).Delete(&model.PlazaSnapshotAsset{}).Error; err != nil {
				return err
			}
		}
		if err := tx.db.Where("work_id IN ?", ids).Delete(&model.PlazaSnapshot{}).Error; err != nil {
			return err
		}
		return tx.db.Where("id IN ?", ids).Delete(&model.PlazaWork{}).Error
	})
	return len(works), err
}
