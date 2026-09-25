package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterPlazaAdminRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/admin/plaza/settings", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		settings, err := svc.AdminPlazaSettings(user)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"settings": settings})
	})

	r.PATCH("/admin/plaza/settings", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		current, err := svc.PlazaSettings()
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
		req := current
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		settings, err := svc.UpdatePlazaSettings(user, req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"settings": settings})
	})

	r.GET("/admin/plaza/applications", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))
		items, total, err := svc.AdminPlazaApplications(user, c.Query("status"), page, pageSize)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"applications": items, "total": total, "page": page, "pageSize": pageSize})
	})

	r.GET("/admin/plaza/applications/:id", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		item, err := svc.AdminPlazaApplication(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"application": item})
	})

	r.GET("/admin/plaza/applications/:id/snapshot", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		project, err := svc.AdminPlazaApplicationSnapshot(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		c.Header("Cache-Control", "no-store")
		ok(c, gin.H{"project": project})
	})

	r.GET("/admin/plaza/applications/:id/assets/:assetId/file", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		if !enforceRateLimit(c, "plaza-admin-asset:"+user.ID, 300, time.Minute) {
			return
		}
		delivery, err := svc.PreparePlazaApplicationAssetDelivery(user, c.Param("id"), c.Param("assetId"), c.GetHeader("Range"))
		if err != nil {
			fail(c, http.StatusNotFound, errors.New("申请资源不存在"))
			return
		}
		writePlazaResourceDelivery(c, delivery)
	})

	r.POST("/admin/plaza/applications/:id/approve", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		work, err := svc.ApprovePlazaApplication(user, c.Param("id"))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"work": work})
	})

	r.POST("/admin/plaza/applications/:id/reject", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
		var req service.PlazaRejectRequest
		_ = c.ShouldBindJSON(&req)
		application, err := svc.RejectPlazaApplication(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"application": application})
	})

	r.GET("/admin/plaza/works", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))
		items, total, err := svc.AdminPlazaWorks(user, c.Query("status"), page, pageSize)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"works": items, "total": total, "page": page, "pageSize": pageSize})
	})

	r.POST("/admin/plaza/works/:id/take-down", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
		var req struct {
			Note string `json:"note"`
		}
		_ = c.ShouldBindJSON(&req)
		work, err := svc.TakeDownPlazaWork(user, c.Param("id"), req.Note)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"work": work})
	})

	r.POST("/admin/plaza/seed-external", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		var items []service.PlazaExternalSeedItem
		if err := c.ShouldBindJSON(&items); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		report, err := svc.AdminSeedPlazaExternal(user, items)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"report": report})
	})
}
