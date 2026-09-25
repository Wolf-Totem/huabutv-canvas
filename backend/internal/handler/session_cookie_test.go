package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestSetSessionCookieExpiresHostOnlyDuplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CANVAS_COOKIE_DOMAIN", ".j11.net")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "https://canvas.j11.net/api/auth/login", nil)
	context.Request.Host = "canvas.j11.net"
	context.Request.Header.Set("X-Forwarded-Proto", "https")
	setSessionCookie(context, "new-session", 3600)
	cookies := recorder.Result().Cookies()
	var expiredHostOnly, setDomain bool
	for _, cookie := range cookies {
		if cookie.Name != service.SessionCookieName {
			continue
		}
		if cookie.Domain == "" && cookie.MaxAge < 0 {
			expiredHostOnly = true
		}
		if (cookie.Domain == "j11.net" || cookie.Domain == ".j11.net") && cookie.Value == "new-session" && cookie.MaxAge > 0 {
			setDomain = true
		}
	}
	if !expiredHostOnly {
		t.Fatalf("expected host-only session cookie to be expired, got %#v", cookies)
	}
	if !setDomain {
		t.Fatalf("expected Domain=.j11.net session cookie, got %#v", cookies)
	}
}
