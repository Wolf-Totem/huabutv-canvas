package handler

import (
	"io"
	"net/http"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterSiteSkinRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/public/site-skin", func(c *gin.Context) {
		skin, err := svc.PublicSiteSkin(requestHost(c))
		if err != nil {
			failService(c, err)
			return
		}
		c.Header("Cache-Control", "no-store")
		ok(c, skin)
	})
	r.GET("/public/site-skin/hero-video", streamStreamerHero(svc, "video"))
	r.GET("/public/site-skin/hero-poster", streamStreamerHero(svc, "poster"))
}

func streamStreamerHero(svc *service.Service, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		stream, err := svc.OpenStreamerHeroAsset(requestHost(c), kind, c.GetHeader("Range"))
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
	}
}
