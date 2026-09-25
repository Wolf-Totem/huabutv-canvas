package app

import (
	"errors"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

func resolvedStreamerHeroURL(skin *model.SiteSkin, kind string) string {
	if skin == nil {
		return ""
	}
	if kind == "poster" {
		if url := sanitizeHomeURL(skin.HeroPosterURL); url != "" {
			return url
		}
		if strings.TrimSpace(skin.HeroPosterResourceID) == "" {
			return ""
		}
		return "/api/public/site-skin/hero-poster?v=" + strconv.FormatInt(skin.UpdatedAt.Unix(), 10)
	}
	if url := sanitizeHomeURL(skin.HeroVideoURL); url != "" {
		return url
	}
	if strings.TrimSpace(skin.HeroVideoResourceID) == "" {
		return ""
	}
	return "/api/public/site-skin/hero-video?v=" + strconv.FormatInt(skin.UpdatedAt.Unix(), 10)
}

func (s *Service) UploadStreamerHeroAsset(actor *model.User, streamerID, kind string, header *multipart.FileHeader) (*model.SiteSkin, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	slot := AppearanceAssetLandingVideo
	if kind == "poster" {
		slot = AppearanceAssetPoster
	} else if kind != "video" {
		return nil, BadAuthRequest("只支持 video 或 poster")
	}
	mimeType, err := validateAppearanceUpload(slot, header)
	if err != nil {
		return nil, err
	}
	header.Header.Set("Content-Type", mimeType)
	resourceKind := "video"
	if kind == "poster" {
		resourceKind = "image"
	}
	resource, err := s.UploadResource(actor.ID, header, resourceKind, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	skin, streamer, err := s.AdminGetStreamerSkin(actor, streamerID)
	if err != nil {
		return nil, err
	}
	if skin.StreamerID == "" {
		skin.StreamerID = streamer.ID
	}
	if kind == "poster" {
		skin.HeroPosterResourceID = resource.ID
		skin.HeroPosterURL = ""
	} else {
		skin.HeroVideoResourceID = resource.ID
		skin.HeroVideoURL = ""
	}
	skin.UpdatedAt = time.Now()
	if err := s.repo.SaveSiteSkin(skin); err != nil {
		return nil, err
	}
	return skin, nil
}

func (s *Service) OpenStreamerHeroAsset(host, kind, rangeHeader string) (*ResourceStream, error) {
	streamer, err := s.ResolveStreamerByHost(host)
	if err != nil {
		return nil, err
	}
	if streamer == nil || streamer.Status != model.StreamerStatusActive {
		return nil, NotFound("未配置专属背景")
	}
	return s.openStreamerHeroByID(streamer.ID, kind, rangeHeader)
}

func (s *Service) OpenStreamerHeroAssetByID(actor *model.User, streamerID, kind, rangeHeader string) (*ResourceStream, error) {
	if err := s.RequireAdmin(actor); err != nil {
		return nil, err
	}
	return s.openStreamerHeroByID(streamerID, kind, rangeHeader)
}

func (s *Service) openStreamerHeroByID(streamerID, kind, rangeHeader string) (*ResourceStream, error) {
	skin, err := s.repo.SiteSkin(streamerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFound("未配置专属背景")
		}
		return nil, err
	}
	resourceID := strings.TrimSpace(skin.HeroVideoResourceID)
	if kind == "poster" {
		resourceID = strings.TrimSpace(skin.HeroPosterResourceID)
	} else if kind != "video" {
		return nil, BadAuthRequest("只支持 video 或 poster")
	}
	if resourceID == "" {
		return nil, NotFound("未配置专属背景")
	}
	resource, err := s.repo.Resource(resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFound("专属背景不存在")
		}
		return nil, err
	}
	return s.openResourceRange(resource.UserID, resource, rangeHeader)
}
