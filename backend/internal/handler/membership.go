package handler

import (
	"net/http"
	"time"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterMembershipRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/membership", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		view, err := svc.PublicMembership(user.ID)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, view)
	})
	r.GET("/membership/products", func(c *gin.Context) {
		if _, err := currentUser(c, svc); err != nil {
			failService(c, err)
			return
		}
		products, err := svc.PublicMembershipProducts()
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"products": products})
	})
	r.GET("/admin/membership/products", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		products, err := svc.AdminMembershipProducts(user)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"products": products})
	})
	r.PATCH("/admin/membership/products/:id", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		var req service.UpdateMembershipProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		product, err := svc.UpdateMembershipProduct(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"product": product})
	})
	r.GET("/admin/settings/support-contact", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		if err := svc.RequireAdmin(user); err != nil {
			failService(c, err)
			return
		}
		setting, err := svc.SupportContact()
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, setting)
	})
	r.PATCH("/admin/settings/support-contact", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		var req service.SupportContactSetting
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		setting, err := svc.UpdateSupportContact(user, req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, setting)
	})
	r.GET("/admin/settings/commerce-methods", handleGetCommerceMethods(svc))
	r.PATCH("/admin/settings/commerce-methods", handlePatchCommerceMethods(svc))
	r.GET("/admin/commerce-methods", handleGetCommerceMethods(svc))
	r.PATCH("/admin/commerce-methods", handlePatchCommerceMethods(svc))
	r.PATCH("/admin/users/:id/storage-quota", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		var req service.AdminStorageQuotaRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		if err := svc.UpdateUserStorageQuota(user, c.Param("id"), req.StorageOverrideBytes); err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"ok": true})
	})
	r.POST("/admin/users/:id/membership/grant", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		if !enforceRateLimit(c, "admin-membership-grant:"+user.ID, 60, time.Hour) {
			return
		}
		var req service.AdminMembershipGrantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		view, err := svc.AdminGrantMembership(user, c.Param("id"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"membership": view})
	})
}

func handleGetCommerceMethods(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		if err := svc.RequireAdmin(user); err != nil {
			failService(c, err)
			return
		}
		methods, err := svc.CommerceMethods()
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, methods)
	}
}

func handlePatchCommerceMethods(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		var req service.CommerceMethods
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		methods, err := svc.UpdateCommerceMethods(user, req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, methods)
	}
}
