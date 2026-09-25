package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterPlazaRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/plaza/settings", func(c *gin.Context) {
		settings, err := svc.PlazaSettings()
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"settings": settings})
	})

	r.GET("/plaza/categories", func(c *gin.Context) {
		items, err := svc.PlazaCategories()
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"categories": items})
	})

	r.GET("/plaza/works", func(c *gin.Context) {
		if !enforceRateLimit(c, "plaza-list:"+c.ClientIP(), 120, time.Minute) {
			return
		}
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))
		viewer := optionalUser(c, svc)
		viewerID := ""
		if viewer != nil {
			viewerID = viewer.ID
		}
		result, err := svc.ListPlazaWorks(c.Query("category"), c.Query("sort"), viewerID, page, pageSize)
		if err != nil {
			failService(c, err)
			return
		}
		c.Header("Cache-Control", "public, max-age=30")
		ok(c, result)
	})

	r.GET("/plaza/works/:slug", func(c *gin.Context) {
		if !enforceRateLimit(c, "plaza-detail:"+c.ClientIP(), 120, time.Minute) {
			return
		}
		viewer := optionalUser(c, svc)
		viewerID := ""
		if viewer != nil {
			viewerID = viewer.ID
		}
		work, err := svc.PublicPlazaWork(c.Param("slug"), viewerID)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"work": work})
	})

	r.GET("/public/plaza-works/:id/snapshot", func(c *gin.Context) {
		if !enforceRateLimit(c, "plaza-snapshot:"+c.ClientIP(), 120, time.Minute) {
			return
		}
		project, err := svc.PublicPlazaSnapshot(c.Param("id"), optionalUser(c, svc))
		if err != nil {
			failService(c, err)
			return
		}
		c.Header("Cache-Control", "no-store")
		ok(c, gin.H{"project": project})
	})

	r.GET("/public/plaza-works/:id/assets/:assetId/file", func(c *gin.Context) {
		if !enforceRateLimit(c, "plaza-asset:"+c.ClientIP(), 300, time.Minute) {
			return
		}
		delivery, err := svc.PreparePlazaWorkAssetDelivery(c.Param("id"), c.Param("assetId"), c.GetHeader("Range"), optionalUser(c, svc))
		if err != nil {
			fail(c, http.StatusNotFound, errors.New("广场资源不存在"))
			return
		}
		writePlazaResourceDelivery(c, delivery)
	})

	r.POST("/plaza/works/:id/events", func(c *gin.Context) {
		if !enforceRateLimit(c, "plaza-event:"+c.ClientIP(), 120, time.Minute) {
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<10)
		var req service.PlazaEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		userID := ""
		if user := optionalUser(c, svc); user != nil {
			userID = user.ID
		}
		if err := svc.RecordPlazaEvent(c.Param("id"), req.Kind, userID); err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"ok": true})
	})

	r.POST("/plaza/works/:id/like", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		work, err := svc.LikePlazaWork(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"work": work})
	})

	r.DELETE("/plaza/works/:id/like", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		work, err := svc.UnlikePlazaWork(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"work": work})
	})

	r.POST("/plaza/works/:id/copy", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		if !enforceRateLimit(c, "plaza-copy:"+user.ID+":"+c.Param("id"), 1, time.Minute) {
			return
		}
		result, err := svc.CopyPlazaWork(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"code": service.CodeOK, "data": result, "msg": "ok"})
	})

	r.POST("/canvas-projects/:id/plaza-applications", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		if !enforceRateLimit(c, "plaza-apply:"+user.ID, 10, time.Minute) {
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
		var req service.PlazaApplyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		application, err := svc.ApplyPlazaWork(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"application": application})
	})

	r.GET("/me/plaza-applications", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		items, err := svc.MyPlazaApplications(user)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"applications": items})
	})

	r.POST("/me/plaza-applications/:id/withdraw", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		application, err := svc.WithdrawPlazaApplication(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"application": application})
	})

	r.GET("/me/plaza-works", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		items, err := svc.MyPlazaWorks(user)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"works": items})
	})

	r.POST("/me/plaza-works/:id/unpublish", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		work, err := svc.UnpublishPlazaWork(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"work": work})
	})

	r.POST("/me/plaza-works/:id/publish", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		work, err := svc.RepublishPlazaWork(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"work": work})
	})
}

func optionalUser(c *gin.Context, svc *service.Service) *model.User {
	user, err := currentUser(c, svc)
	if err != nil {
		return nil
	}
	return user
}

func writePlazaResourceDelivery(c *gin.Context, delivery *service.ResourceDelivery) {
	if delivery == nil {
		fail(c, http.StatusNotFound, errors.New("广场资源不存在"))
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Security-Policy", "sandbox")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Robots-Tag", "noindex, nofollow")
	if delivery.RedirectURL != "" {
		c.Redirect(http.StatusTemporaryRedirect, delivery.RedirectURL)
		return
	}
	if delivery.Stream == nil {
		fail(c, http.StatusNotFound, errors.New("广场资源不存在"))
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
	if resource.Provider == "local" {
		if seeker, ok := stream.Body.(io.ReadSeeker); ok {
			c.Header("Content-Type", mimeType)
			http.ServeContent(c.Writer, c.Request, resource.ID, resource.UpdatedAt, seeker)
			return
		}
	}
	if stream.ContentRange != "" {
		c.Header("Content-Range", stream.ContentRange)
	}
	c.DataFromReader(stream.StatusCode, stream.ContentLength, mimeType, stream.Body, nil)
}
