package handler

import (
	"io"
	"net/http"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterAdminStreamerRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/admin/streamers", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		items, err := svc.AdminListStreamers(user)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"items": items})
	})
	r.POST("/admin/streamers", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		var req service.CreateStreamerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		created, err := svc.AdminCreateStreamer(user, req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"streamer": created})
	})
	r.POST("/admin/streamers/:id/disable", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		updated, err := svc.AdminSetStreamerStatus(user, c.Param("id"), model.StreamerStatusDisabled)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"streamer": updated})
	})
	r.POST("/admin/streamers/:id/enable", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		updated, err := svc.AdminSetStreamerStatus(user, c.Param("id"), model.StreamerStatusActive)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"streamer": updated})
	})
	r.PATCH("/admin/streamers/:id", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		var req service.UpdateStreamerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		updated, err := svc.AdminUpdateStreamer(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"streamer": updated})
	})
	r.POST("/admin/streamers/:id/rotate-code", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		updated, err := svc.AdminRotateStreamerInviteCode(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"streamer": updated})
	})
	r.GET("/admin/streamers/:id/skin", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		skin, streamer, err := svc.AdminGetStreamerSkin(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"skin": skin, "streamer": streamer})
	})
	r.PUT("/admin/streamers/:id/skin", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256<<10)
		var req service.UpdateStreamerSkinRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		skin, err := svc.AdminUpdateStreamerSkin(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"skin": skin})
	})
	r.GET("/admin/streamers/:id/hero-video", previewStreamerHero(svc, "video"))
	r.GET("/admin/streamers/:id/hero-poster", previewStreamerHero(svc, "poster"))
	r.POST("/admin/streamers/:id/hero-video", uploadStreamerHero(svc, "video"))
	r.POST("/admin/streamers/:id/hero-poster", uploadStreamerHero(svc, "poster"))
	r.GET("/admin/payouts", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		page, size, err := parsePaginationQuery(c, 20)
		if err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		result, err := svc.AdminListStreamerPayouts(user, page, size)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, result)
	})
	r.POST("/admin/payouts/:id/approve", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		updated, err := svc.AdminApproveStreamerPayout(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"payout": updated})
	})
	r.POST("/admin/payouts/:id/reject", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
		var req service.RejectStreamerPayoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		updated, err := svc.AdminRejectStreamerPayout(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"payout": updated})
	})
	r.GET("/admin/agent-share-models", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		items, err := svc.AdminListAgentShareModels(user)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"items": items})
	})
	r.PATCH("/admin/agent-share-models/:id", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
		var req service.UpdateAgentShareRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		updated, err := svc.AdminUpdateAgentShareModel(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"model": updated})
	})
}

func previewStreamerHero(svc *service.Service, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		stream, err := svc.OpenStreamerHeroAssetByID(user, c.Param("id"), kind, c.GetHeader("Range"))
		if err != nil {
			failService(c, err)
			return
		}
		defer stream.Body.Close()
		resource := stream.Resource
		mimeType := resource.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		c.Header("Cache-Control", "private, no-store")
		c.Header("Accept-Ranges", "bytes")
		if stream.ContentRange != "" {
			c.Header("Content-Range", stream.ContentRange)
		}
		if seeker, available := stream.Body.(io.ReadSeeker); available {
			c.Header("Content-Type", mimeType)
			http.ServeContent(c.Writer, c.Request, resource.ID, resource.UpdatedAt, seeker)
			return
		}
		c.DataFromReader(stream.StatusCode, stream.ContentLength, mimeType, stream.Body, nil)
	}
}

func uploadStreamerHero(svc *service.Service, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		slot := service.AppearanceAssetLandingVideo
		if kind == "poster" {
			slot = service.AppearanceAssetPoster
		}
		maxBytes, err := service.AppearanceAssetMaxBytes(slot)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+(1<<20))
		file, err := c.FormFile("file")
		if err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		skin, err := svc.UploadStreamerHeroAsset(user, c.Param("id"), kind, file)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"skin": skin})
	}
}
