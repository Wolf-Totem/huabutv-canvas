package plaza

import (
	"io"

	"infinite-canvas/backend/internal/assets"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
)

type Host interface {
	OpenResource(userID, resourceID string) (*model.Resource, io.ReadCloser, error)
	OpenResourceRange(userID string, resource *model.Resource, rangeHeader string) (*assets.ResourceStream, error)
	StoreResource(userID, kind, fileName, mimeType string, size int64, width, height int, durationMs int64, body io.Reader) (*model.Resource, error)
	PrepareResourceDelivery(userID string, resource *model.Resource, options assets.ResourceDeliveryOptions) (*assets.ResourceDelivery, error)
	RecordAdminAudit(actor *model.User, action, targetType, targetID, summary string, metadata any) error
}

type nopHost struct{}

func (nopHost) OpenResource(string, string) (*model.Resource, io.ReadCloser, error) {
	return nil, nil, nil
}
func (nopHost) OpenResourceRange(string, *model.Resource, string) (*assets.ResourceStream, error) {
	return nil, nil
}
func (nopHost) StoreResource(string, string, string, string, int64, int, int, int64, io.Reader) (*model.Resource, error) {
	return nil, nil
}
func (nopHost) PrepareResourceDelivery(string, *model.Resource, assets.ResourceDeliveryOptions) (*assets.ResourceDelivery, error) {
	return nil, nil
}
func (nopHost) RecordAdminAudit(*model.User, string, string, string, string, any) error { return nil }

type Service struct {
	repo *repository.Repository
	host Host
}

func New(repo *repository.Repository, host Host) *Service {
	if host == nil {
		host = nopHost{}
	}
	return &Service{repo: repo, host: host}
}
