package model

import "time"

const (
	PlazaSystemUserID = "plaza-system"
	PlazaSettingKey   = "plaza"

	PlazaApplicationPending   = "pending"
	PlazaApplicationApproved  = "approved"
	PlazaApplicationRejected  = "rejected"
	PlazaApplicationWithdrawn = "withdrawn"

	PlazaWorkListed    = "listed"
	PlazaWorkUnlisted  = "unlisted"
	PlazaWorkTakenDown = "taken_down"

	PlazaCategoryGenre    = "genre"
	PlazaCategoryCampaign = "campaign"
	PlazaCategoryFeatured = "featured"

	PlazaEventView   = "view"
	PlazaEventWatch  = "watch"
	PlazaEventTour   = "tour"
	PlazaEventLike   = "like"
	PlazaEventUnlike = "unlike"
	PlazaEventCopy   = "copy"

	PlazaAssetImage = "image"
	PlazaAssetVideo = "video"
	PlazaAssetAudio = "audio"
	PlazaAssetCover = "cover"
	PlazaAssetWatch = "watch"
)

type PlazaCategory struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	Slug      string    `json:"slug" gorm:"uniqueIndex;size:64"`
	Name      string    `json:"name" gorm:"size:80"`
	Kind      string    `json:"kind" gorm:"size:24"`
	Sort      int       `json:"sort" gorm:"index"`
	Enabled   bool      `json:"enabled" gorm:"index"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (PlazaCategory) TableName() string { return "plaza_categories" }

type PlazaApplication struct {
	ID               string     `json:"id" gorm:"primaryKey;size:36"`
	UserID           string     `json:"userId" gorm:"index;size:36"`
	ProjectID        string     `json:"projectId" gorm:"index;size:80"`
	WorkID           string     `json:"workId" gorm:"index;size:36"`
	Title            string     `json:"title" gorm:"size:80"`
	Subtitle         string     `json:"subtitle" gorm:"size:160"`
	CategoryID       string     `json:"categoryId" gorm:"size:36"`
	CampaignIDsJSON  string     `json:"campaignIdsJson" gorm:"type:text"`
	CoverResourceID  string     `json:"coverResourceId" gorm:"size:36"`
	WatchResourceID  string     `json:"watchResourceId" gorm:"size:36"`
	AllowWatch       bool       `json:"allowWatch"`
	AllowProcessView bool       `json:"allowProcessView"`
	AllowCopy        bool       `json:"allowCopy"`
	OriginalityAck   bool       `json:"originalityAck"`
	Status           string     `json:"status" gorm:"index;size:24"`
	ReviewerID       string     `json:"reviewerId" gorm:"size:36"`
	ReviewNote       string     `json:"reviewNote" gorm:"size:500"`
	DraftPayloadJSON string     `json:"draftPayloadJson" gorm:"type:text"`
	NodeCount        int        `json:"nodeCount"`
	MediaCount       int        `json:"mediaCount"`
	SubmittedAt      time.Time  `json:"submittedAt" gorm:"index"`
	ReviewedAt       *time.Time `json:"reviewedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (PlazaApplication) TableName() string { return "plaza_applications" }

type PlazaWork struct {
	ID               string     `json:"id" gorm:"primaryKey;size:36"`
	Slug             string     `json:"slug" gorm:"uniqueIndex;size:80"`
	AuthorID         string     `json:"authorId" gorm:"index;size:36"`
	SourceProjectID  string     `json:"sourceProjectId" gorm:"uniqueIndex;size:80"`
	SnapshotID       string     `json:"snapshotId" gorm:"index;size:36"`
	Title            string     `json:"title" gorm:"size:80"`
	Subtitle         string     `json:"subtitle" gorm:"size:160"`
	CategoryID       string     `json:"categoryId" gorm:"index;size:36"`
	CoverAssetID     string     `json:"coverAssetId" gorm:"size:36"`
	WatchAssetID     string     `json:"watchAssetId" gorm:"size:36"`
	CoverExternalURL string     `json:"coverExternalUrl,omitempty" gorm:"size:1000"`
	WatchExternalURL string     `json:"watchExternalUrl,omitempty" gorm:"size:1000"`
	Status           string     `json:"status" gorm:"index;size:24"`
	AllowWatch       bool       `json:"allowWatch"`
	AllowProcessView bool       `json:"allowProcessView"`
	AllowCopy        bool       `json:"allowCopy"`
	BadgesJSON       string     `json:"badgesJson" gorm:"type:text"`
	Score            float64    `json:"score" gorm:"index"`
	ViewCount        int64      `json:"viewCount"`
	WatchCount       int64      `json:"watchCount"`
	TourCount        int64      `json:"tourCount"`
	LikeCount        int64      `json:"likeCount"`
	CopyCount        int64      `json:"copyCount"`
	PinnedAt         *time.Time `json:"pinnedAt" gorm:"index"`
	ListedAt         *time.Time `json:"listedAt" gorm:"index"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (PlazaWork) TableName() string { return "plaza_works" }

type PlazaWorkTag struct {
	WorkID     string `json:"workId" gorm:"primaryKey;size:36"`
	CategoryID string `json:"categoryId" gorm:"primaryKey;size:36;index"`
}

func (PlazaWorkTag) TableName() string { return "plaza_work_tags" }

type PlazaSnapshot struct {
	ID            string    `json:"id" gorm:"primaryKey;size:36"`
	WorkID        string    `json:"workId" gorm:"index;size:36"`
	ApplicationID string    `json:"applicationId" gorm:"index;size:36"`
	PayloadJSON   string    `json:"payloadJson" gorm:"type:text;not null"`
	PayloadHash   string    `json:"payloadHash" gorm:"size:64;index"`
	NodeCount     int       `json:"nodeCount"`
	MediaCount    int       `json:"mediaCount"`
	CoverAssetID  string    `json:"coverAssetId" gorm:"size:36"`
	WatchAssetID  string    `json:"watchAssetId" gorm:"size:36"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (PlazaSnapshot) TableName() string { return "plaza_snapshots" }

type PlazaSnapshotAsset struct {
	SnapshotID       string `json:"snapshotId" gorm:"primaryKey;size:36"`
	AssetID          string `json:"assetId" gorm:"primaryKey;size:36"`
	SourceResourceID string `json:"sourceResourceId" gorm:"size:36"`
	Kind             string `json:"kind" gorm:"size:24"`
}

func (PlazaSnapshotAsset) TableName() string { return "plaza_snapshot_assets" }

type PlazaEvent struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	WorkID    string    `json:"workId" gorm:"index;size:36;index:idx_plaza_events_work_kind_day,priority:1"`
	UserID    string    `json:"userId" gorm:"index;size:36"`
	Kind      string    `json:"kind" gorm:"size:24;index:idx_plaza_events_work_kind_day,priority:2"`
	Day       string    `json:"day" gorm:"size:10;index:idx_plaza_events_work_kind_day,priority:3"`
	CreatedAt time.Time `json:"createdAt"`
}

func (PlazaEvent) TableName() string { return "plaza_events" }

type PlazaLike struct {
	WorkID    string    `json:"workId" gorm:"primaryKey;size:36"`
	UserID    string    `json:"userId" gorm:"primaryKey;size:36"`
	CreatedAt time.Time `json:"createdAt"`
}

func (PlazaLike) TableName() string { return "plaza_likes" }
