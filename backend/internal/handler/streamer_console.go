package handler

import (
	"net/http"
	"time"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterStreamerConsoleRoutes(r *gin.RouterGroup, svc *service.Service) {
	r.GET("/agent/me", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		me, err := svc.StreamerConsoleMe(user, requestHost(c))
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, me)
	})
	r.GET("/agent/summary", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		from, to, err := parseOptionalTimeRange(c.Query("from"), c.Query("to"))
		if err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		summary, err := svc.StreamerConsoleSummary(user, from, to)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, summary)
	})
	r.GET("/agent/users", func(c *gin.Context) {
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
		result, err := svc.StreamerConsoleUsers(user, page, size)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, result)
	})
	r.GET("/agent/rebates", func(c *gin.Context) {
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
		result, err := svc.StreamerConsoleRebates(user, c.Query("capability"), page, size)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, result)
	})
	r.GET("/agent/payouts", func(c *gin.Context) {
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
		result, err := svc.StreamerConsolePayouts(user, page, size)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, result)
	})
	r.POST("/agent/payouts", func(c *gin.Context) {
		user, err := currentUser(c, svc)
		if err != nil {
			failService(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
		var req service.CreateStreamerPayoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		created, err := svc.CreateStreamerPayout(user, req)
		if err != nil {
			failService(c, err)
			return
		}
		ok(c, gin.H{"payout": created})
	})
}

func parseOptionalTimeRange(fromRaw, toRaw string) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if fromRaw != "" {
		parsed, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return nil, nil, err
		}
		from = &parsed
	}
	if toRaw != "" {
		parsed, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return nil, nil, err
		}
		to = &parsed
	}
	return from, to, nil
}
