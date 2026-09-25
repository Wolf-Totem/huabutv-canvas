package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// 主题库最多 16 套完整 token，JSON 远超早期 32KB 上限。
const appearanceAdminPatchMaxBytes = 1 << 20

func RegisterAppearanceRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/public/appearance", func(c *gin.Context) {
		setting, err := svc.Appearance()
		if err != nil {
			failService(c, err)
			return
		}
		c.Header("Cache-Control", "no-store")
		ok(c, gin.H{"appearance": setting})
	})

	r.GET("/public/appearance/assets/:slot", func(c *gin.Context) {
		stream, err := svc.OpenAppearanceAsset(c.Param("slot"), c.GetHeader("Range"))
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
		if c.Query("v") != "" {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "public, no-cache")
		}
		c.Header("Accept-Ranges", "bytes")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Content-Type-Options", "nosniff")
		if stream.ContentRange != "" {
			c.Header("Content-Range", stream.ContentRange)
		}
		if seeker, available := stream.Body.(io.ReadSeeker); available {
			c.Header("Content-Type", mimeType)
			http.ServeContent(c.Writer, c.Request, resource.ID, resource.UpdatedAt, seeker)
			return
		}
		c.DataFromReader(stream.StatusCode, stream.ContentLength, mimeType, stream.Body, nil)
	})

	r.GET("/admin/settings/appearance", func(c *gin.Context) {
		actor, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		setting, err := svc.AdminAppearance(actor)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"setting": setting})
	})

	r.PATCH("/admin/settings/appearance", func(c *gin.Context) {
		actor, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		current, err := svc.AdminAppearance(actor)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, appearanceAdminPatchMaxBytes)
		req := current.AppearanceSetting
		if err := c.ShouldBindJSON(&req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				fail(c, http.StatusRequestEntityTooLarge, errors.New("外观配置过大，请减少主题套数后再保存"))
				return
			}
			fail(c, http.StatusBadRequest, err)
			return
		}
		setting, err := svc.UpdateAppearance(actor, req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"setting": setting})
	})

	r.DELETE("/admin/settings/appearance", func(c *gin.Context) {
		actor, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		setting, err := svc.ResetAppearance(actor)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"setting": setting})
	})

	r.POST("/admin/settings/appearance/assets/:slot", func(c *gin.Context) {
		actor, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		policy, available := loadRuntimePolicy(c, svc)
		if !available || !enforceRateLimit(c, "admin-appearance-upload:"+actor.ID, policy.Request.ResourceUploadPerMinute, time.Minute) {
			return
		}
		maxBytes, err := service.AppearanceAssetMaxBytes(c.Param("slot"))
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
		resource, err := svc.UploadAppearanceAsset(actor, c.Param("slot"), file)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"resource": resource})
	})

	r.POST("/admin/settings/appearance/media", func(c *gin.Context) {
		actor, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		policy, available := loadRuntimePolicy(c, svc)
		if !available || !enforceRateLimit(c, "admin-appearance-media:"+actor.ID, policy.Request.ResourceUploadPerMinute, time.Minute) {
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, (256<<20)+(1<<20))
		file, err := c.FormFile("file")
		if err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		uploaded, err := svc.UploadAppearanceMedia(actor, file)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"resource": uploaded.Resource, "displayUrl": uploaded.DisplayURL, "originalUrl": uploaded.OriginalURL, "compression": uploaded.Compression})
	})

	r.GET("/public/appearance/media/:id", func(c *gin.Context) {
		width, _ := strconv.Atoi(c.Query("w"))
		delivery, err := svc.AppearanceMedia(c.Param("id"), c.Query("variant"), width)
		if err != nil {
			failService(c, err)
			return
		}
		c.Header("Cache-Control", "public, max-age=300")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Content-Type-Options", "nosniff")
		if delivery.RedirectURL != "" {
			c.Redirect(http.StatusFound, delivery.RedirectURL)
			return
		}
		if delivery.Stream == nil {
			fail(c, http.StatusNotFound, errors.New("外观资源不存在"))
			return
		}
		stream := delivery.Stream
		defer stream.Body.Close()
		resource := stream.Resource
		mimeType := resource.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		c.Header("Accept-Ranges", "bytes")
		if seeker, available := stream.Body.(io.ReadSeeker); available {
			c.Header("Content-Type", mimeType)
			http.ServeContent(c.Writer, c.Request, resource.ID, resource.UpdatedAt, seeker)
			return
		}
		c.DataFromReader(stream.StatusCode, stream.ContentLength, mimeType, stream.Body, nil)
	})
	registerLive2DRoutes(r, svc)
}
