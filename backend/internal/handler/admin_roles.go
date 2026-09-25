package handler

import (
	"net/http"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterAdminRoleRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/admin/roles", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		catalog, err := svc.AdminRoleCatalog(user)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, catalog)
	})
	r.PUT("/admin/roles/:role/permissions", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		var req service.UpdateRolePermissionsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		catalog, err := svc.AdminUpdateRolePermissions(user, c.Param("role"), req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, catalog)
	})
}
