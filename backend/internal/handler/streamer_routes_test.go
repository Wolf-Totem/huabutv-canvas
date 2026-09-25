package handler

import (
	"testing"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestStreamerRoutesAreRegisteredAndHaveNoCreditWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterCanvasAPI(router.Group("/api"), &service.Service{})
	wanted := map[string]bool{
		"GET /api/admin/streamers":                  false,
		"POST /api/admin/streamers":                 false,
		"POST /api/admin/streamers/:id/disable":     false,
		"POST /api/admin/streamers/:id/enable":      false,
		"GET /api/public/site-skin":                 false,
		"GET /api/public/site-skin/hero-video":      false,
		"GET /api/public/site-skin/hero-poster":     false,
		"GET /api/admin/streamers/:id/hero-video":   false,
		"GET /api/admin/streamers/:id/hero-poster":  false,
		"POST /api/admin/streamers/:id/hero-video":  false,
		"POST /api/admin/streamers/:id/hero-poster": false,
		"GET /api/agent/me":                         false,
		"GET /api/agent/summary":                    false,
		"GET /api/agent/users":                      false,
		"GET /api/agent/rebates":                    false,
		"GET /api/agent/payouts":                    false,
		"POST /api/agent/payouts":                   false,
		"GET /api/admin/payouts":                    false,
		"POST /api/admin/payouts/:id/approve":       false,
		"POST /api/admin/payouts/:id/reject":        false,
		"GET /api/admin/agent-share-models":         false,
		"PATCH /api/admin/agent-share-models/:id":   false,
		"PATCH /api/admin/streamers/:id":            false,
		"PUT /api/admin/streamers/:id/skin":         false,
		"GET /api/admin/roles":                      false,
		"PUT /api/admin/roles/:role/permissions":    false,
	}
	var agentWrites []string
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := wanted[key]; exists {
			wanted[key] = true
		}
		if route.Method != "GET" && route.Method != "HEAD" && route.Method != "OPTIONS" {
			switch route.Path {
			case "/api/agent/me", "/api/agent/summary", "/api/agent/users":
				agentWrites = append(agentWrites, key)
			}
		}
	}
	for route, found := range wanted {
		if !found {
			t.Errorf("route %s is not registered", route)
		}
	}
	if len(agentWrites) > 0 {
		t.Fatalf("streamer console must stay read-only except payouts, found writes: %v", agentWrites)
	}
}

func TestCloudAgentHandlerFileWasNotReplaced(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAgentRoutes(router.Group("/api"), &service.Service{})
	found := false
	for _, route := range router.Routes() {
		if route.Path == "/api/agent/capabilities" {
			found = true
		}
	}
	if !found {
		t.Fatal("cloud agent capabilities route missing; do not overwrite handler/agent.go")
	}
}
