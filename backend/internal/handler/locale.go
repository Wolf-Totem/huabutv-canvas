package handler

import (
	"net/http"
	"strings"

	"infinite-canvas/backend/internal/locale"
	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterLocaleRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/locale/recommend", func(c *gin.Context) {
		result, err := svc.RecommendLocale(service.RequestCountryCode(c.Request.Header))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, result)
	})
	r.GET("/locales/:lng/:ns", func(c *gin.Context) {
		ns := strings.TrimSuffix(c.Param("ns"), ".json")
		data, err := svc.LocaleNamespaceJSON(c.Param("lng"), ns)
		if err != nil {
			fail(c, http.StatusNotFound, err)
			return
		}
		c.Header("Cache-Control", "public, max-age=60")
		c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	})
	r.PATCH("/me/locale", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		var req struct {
			Locale string `json:"locale"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		updated, err := svc.UpdateUserLocale(user, req.Locale)
		if err != nil {
			failService(c, err)
			return
		}
		publicUser, err := svc.PublicAuthUser(updated)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"user": publicUser, "locale": locale.Normalize(updated.Locale)})
	})
}
