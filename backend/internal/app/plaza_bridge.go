package app

import (
	"io"

	"infinite-canvas/backend/internal/assets"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/plaza"
)

type (
	PlazaSettings        = plaza.Settings
	PlazaApplyRequest    = plaza.ApplyRequest
	PlazaRejectRequest   = plaza.RejectRequest
	PlazaEventRequest    = plaza.EventRequest
	PlazaCopyResult      = plaza.CopyResult
	PlazaPublicWork      = plaza.PublicWork
	PlazaWorkList        = plaza.WorkList
	PlazaApplicationView = plaza.ApplicationView
	PlazaPublicCategory  = plaza.PublicCategory
)

type plazaHost struct {
	svc *Service
}

func (h plazaHost) OpenResource(userID, resourceID string) (*model.Resource, io.ReadCloser, error) {
	if h.svc == nil {
		return nil, nil, nil
	}
	return h.svc.OpenResource(userID, resourceID)
}

func (h plazaHost) OpenResourceRange(userID string, resource *model.Resource, rangeHeader string) (*assets.ResourceStream, error) {
	if h.svc == nil {
		return nil, nil
	}
	return h.svc.openResourceRange(userID, resource, rangeHeader)
}

func (h plazaHost) StoreResource(userID, kind, fileName, mimeType string, size int64, width, height int, durationMs int64, body io.Reader) (*model.Resource, error) {
	if h.svc == nil {
		return nil, nil
	}
	resource, _, err := h.svc.storeResource(userID, kind, fileName, mimeType, size, width, height, durationMs, body, nil, false)
	return resource, err
}

func (h plazaHost) PrepareResourceDelivery(userID string, resource *model.Resource, options assets.ResourceDeliveryOptions) (*assets.ResourceDelivery, error) {
	if h.svc == nil {
		return nil, nil
	}
	return h.svc.prepareResourceDelivery(userID, resource, options)
}

func (h plazaHost) RecordAdminAudit(actor *model.User, action, targetType, targetID, summary string, metadata any) error {
	if h.svc == nil {
		return nil
	}
	return h.svc.appendAdminAudit(actor, action, targetType, targetID, summary, metadata)
}

func (s *Service) plazaDomain() *plaza.Service {
	if s == nil {
		return plaza.New(nil, nil)
	}
	if s.plaza != nil {
		return s.plaza
	}
	return plaza.New(s.repo, plazaHost{svc: s})
}

func (s *Service) PlazaSettings() (PlazaSettings, error) {
	return s.plazaDomain().Settings()
}

func (s *Service) AdminPlazaSettings(actor *model.User) (PlazaSettings, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return PlazaSettings{}, err
	}
	return s.plazaDomain().Settings()
}

func (s *Service) UpdatePlazaSettings(actor *model.User, req PlazaSettings) (PlazaSettings, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return PlazaSettings{}, err
	}
	return s.plazaDomain().UpdateSettings(actor, req)
}

func (s *Service) PlazaCategories() ([]model.PlazaCategory, error) {
	return s.plazaDomain().Categories()
}

func (s *Service) ApplyPlazaWork(user *model.User, projectID string, req PlazaApplyRequest) (*PlazaApplicationView, error) {
	return s.plazaDomain().Apply(user, projectID, req)
}

func (s *Service) MyPlazaApplications(user *model.User) ([]PlazaApplicationView, error) {
	return s.plazaDomain().MyApplications(user)
}

func (s *Service) WithdrawPlazaApplication(user *model.User, id string) (*PlazaApplicationView, error) {
	return s.plazaDomain().Withdraw(user, id)
}

func (s *Service) MyPlazaWorks(user *model.User) ([]PlazaPublicWork, error) {
	return s.plazaDomain().MyWorks(user)
}

func (s *Service) UnpublishPlazaWork(user *model.User, id string) (*PlazaPublicWork, error) {
	return s.plazaDomain().Unpublish(user, id)
}

func (s *Service) RepublishPlazaWork(user *model.User, id string) (*PlazaPublicWork, error) {
	return s.plazaDomain().Republish(user, id)
}

func (s *Service) ListPlazaWorks(category, sort, viewerID string, page, pageSize int) (*PlazaWorkList, error) {
	return s.plazaDomain().ListWorks(category, sort, viewerID, page, pageSize)
}

func (s *Service) PublicPlazaWork(slug, viewerID string) (*PlazaPublicWork, error) {
	return s.plazaDomain().PublicWork(slug, viewerID)
}

func (s *Service) PublicPlazaSnapshot(slug string, viewer *model.User) (map[string]any, error) {
	return s.plazaDomain().PublicSnapshot(slug, viewer)
}

func (s *Service) PreparePlazaWorkAssetDelivery(slug, assetID, rangeHeader string, viewer *model.User) (*ResourceDelivery, error) {
	return s.plazaDomain().OpenWorkAsset(slug, assetID, rangeHeader, viewer)
}

func (s *Service) RecordPlazaEvent(slug, kind, userID string) error {
	return s.plazaDomain().RecordEvent(slug, kind, userID)
}

func (s *Service) LikePlazaWork(user *model.User, id string) (*PlazaPublicWork, error) {
	return s.plazaDomain().Like(user, id)
}

func (s *Service) UnlikePlazaWork(user *model.User, id string) (*PlazaPublicWork, error) {
	return s.plazaDomain().Unlike(user, id)
}

func (s *Service) CopyPlazaWork(user *model.User, id string) (*PlazaCopyResult, error) {
	return s.plazaDomain().Copy(user, id)
}

func (s *Service) AdminPlazaApplications(actor *model.User, status string, page, pageSize int) ([]PlazaApplicationView, int64, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, 0, err
	}
	return s.plazaDomain().AdminApplications(status, page, pageSize)
}

func (s *Service) AdminPlazaApplication(actor *model.User, id string) (*PlazaApplicationView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.plazaDomain().AdminApplication(id)
}

func (s *Service) AdminPlazaApplicationSnapshot(actor *model.User, id string) (map[string]any, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.plazaDomain().AdminApplicationSnapshot(id)
}

func (s *Service) PreparePlazaApplicationAssetDelivery(actor *model.User, applicationID, resourceID, rangeHeader string) (*ResourceDelivery, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.plazaDomain().OpenApplicationDraftAsset(applicationID, resourceID, rangeHeader)
}

func (s *Service) ApprovePlazaApplication(actor *model.User, id string) (*PlazaPublicWork, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.plazaDomain().Approve(actor, id)
}

func (s *Service) RejectPlazaApplication(actor *model.User, id string, req PlazaRejectRequest) (*PlazaApplicationView, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.plazaDomain().Reject(actor, id, req.Note)
}

func (s *Service) AdminPlazaWorks(actor *model.User, status string, page, pageSize int) ([]PlazaPublicWork, int64, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, 0, err
	}
	return s.plazaDomain().AdminWorks(status, page, pageSize)
}

func (s *Service) TakeDownPlazaWork(actor *model.User, id, note string) (*PlazaPublicWork, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.plazaDomain().TakeDown(actor, id, note)
}
