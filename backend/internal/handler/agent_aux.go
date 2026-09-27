package handler

import (
	"github.com/gin-gonic/gin"
	"infinite-canvas/backend/internal/service"
)

// RegisterAgentAuxRoutes adds Agent HTTP surfaces that must not edit agent.go.
func RegisterAgentAuxRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/agent/skills/usage", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		view, err := svc.CloudAgentSkillUsage(user.ID)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, view)
	})
}
