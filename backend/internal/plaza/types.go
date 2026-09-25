package plaza

import (
	"time"

	"infinite-canvas/backend/internal/model"
)

type Settings struct {
	Enabled         bool `json:"enabled"`
	ApplyEnabled    bool `json:"applyEnabled"`
	CopyEnabled     bool `json:"copyEnabled"`
	PublicWatch     bool `json:"publicWatch"`
	ApplyDailyLimit int  `json:"applyDailyLimit"`
}

func DefaultSettings() Settings {
	return Settings{Enabled: false, ApplyEnabled: false, CopyEnabled: false, PublicWatch: true, ApplyDailyLimit: 5}
}

type ApplyRequest struct {
	Title            string   `json:"title"`
	Subtitle         string   `json:"subtitle"`
	CategoryID       string   `json:"categoryId"`
	CampaignIDs      []string `json:"campaignIds"`
	CoverNodeID      string   `json:"coverNodeId"`
	WatchNodeID      string   `json:"watchNodeId"`
	AllowWatch       bool     `json:"allowWatch"`
	AllowProcessView bool     `json:"allowProcessView"`
	AllowCopy        bool     `json:"allowCopy"`
	OriginalityAck   bool     `json:"originalityAck"`
}

type RejectRequest struct {
	Note string `json:"note"`
}

type EventRequest struct {
	Kind string `json:"kind"`
}

type CopyResult struct {
	ProjectID string `json:"projectId"`
	Title     string `json:"title"`
}

type PublicAuthor struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

type PublicCategory struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type PublicWork struct {
	ID               string         `json:"id"`
	Slug             string         `json:"slug"`
	Title            string         `json:"title"`
	Subtitle         string         `json:"subtitle"`
	Status           string         `json:"status"`
	AllowWatch       bool           `json:"allowWatch"`
	AllowProcessView bool           `json:"allowProcessView"`
	AllowCopy        bool           `json:"allowCopy"`
	Badges           []string       `json:"badges"`
	Score            float64        `json:"score"`
	ViewCount        int64          `json:"viewCount"`
	WatchCount       int64          `json:"watchCount"`
	TourCount        int64          `json:"tourCount"`
	LikeCount        int64          `json:"likeCount"`
	CopyCount        int64          `json:"copyCount"`
	Liked            bool           `json:"liked"`
	CoverURL         string         `json:"coverUrl,omitempty"`
	WatchURL         string         `json:"watchUrl,omitempty"`
	Author           PublicAuthor   `json:"author"`
	Category         PublicCategory `json:"category"`
	ListedAt         *time.Time     `json:"listedAt,omitempty"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type WorkList struct {
	Works   []PublicWork `json:"works"`
	Next    string       `json:"next,omitempty"`
	HasMore bool         `json:"hasMore"`
}

type ApplicationView struct {
	ID               string       `json:"id"`
	UserID           string       `json:"userId"`
	ProjectID        string       `json:"projectId"`
	WorkID           string       `json:"workId,omitempty"`
	Title            string       `json:"title"`
	Subtitle         string       `json:"subtitle"`
	CategoryID       string       `json:"categoryId"`
	CampaignIDs      []string     `json:"campaignIds"`
	CoverNodeID      string       `json:"coverNodeId,omitempty"`
	WatchNodeID      string       `json:"watchNodeId,omitempty"`
	AllowWatch       bool         `json:"allowWatch"`
	AllowProcessView bool         `json:"allowProcessView"`
	AllowCopy        bool         `json:"allowCopy"`
	Status           string       `json:"status"`
	ReviewNote       string       `json:"reviewNote,omitempty"`
	NodeCount        int          `json:"nodeCount"`
	MediaCount       int          `json:"mediaCount"`
	SubmittedAt      time.Time    `json:"submittedAt"`
	ReviewedAt       *time.Time   `json:"reviewedAt,omitempty"`
	Author           PublicAuthor `json:"author"`
}

type plazaDraft struct {
	Document        map[string]any `json:"document"`
	ResourceIDs     []string       `json:"resourceIds"`
	CoverNodeID     string         `json:"coverNodeId"`
	WatchNodeID     string         `json:"watchNodeId"`
	CoverResourceID string         `json:"coverResourceId"`
	WatchResourceID string         `json:"watchResourceId"`
}

func plazaAssetURL(workID, assetID string) string {
	return "/api/public/plaza-works/" + workID + "/assets/" + assetID + "/file"
}

func plazaAdminDraftAssetURL(applicationID, resourceID string) string {
	return "/api/admin/plaza/applications/" + applicationID + "/assets/" + resourceID + "/file"
}

func workScore(work *model.PlazaWork) float64 {
	if work == nil {
		return 0
	}
	return float64(work.WatchCount)*3 + float64(work.TourCount)*2 + float64(work.CopyCount)*8 + float64(work.LikeCount) + float64(work.ViewCount)*0.2
}
