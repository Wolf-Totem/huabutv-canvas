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

func TestSessionCookieDomainFollowsRequestParent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name      string
		host      string
		parentEnv string
		cookieEnv string
		want      string
	}{
		{name: "www default", host: "www.huabutv.com", want: ".huabutv.com"},
		{name: "www mixed live env", host: "www.huabutv.com", parentEnv: "j11.net", cookieEnv: ".j11.net", want: ".huabutv.com"},
		{name: "streamer huabutv mixed", host: "a.huabutv.com", parentEnv: "j11.net", cookieEnv: ".j11.net", want: ".huabutv.com"},
		{name: "canvas j11 mixed", host: "canvas.j11.net", parentEnv: "j11.net", cookieEnv: ".j11.net", want: ".j11.net"},
		{name: "canvas j11 huabutv primary", host: "canvas.j11.net", parentEnv: "huabutv.com", want: ".j11.net"},
		{name: "www ignores mismatched cookie env", host: "www.huabutv.com", parentEnv: "huabutv.com", cookieEnv: ".j11.net", want: ".huabutv.com"},
		{name: "localhost host-only", host: "localhost", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CANVAS_PUBLIC_PARENT_DOMAIN", test.parentEnv)
			t.Setenv("CANVAS_COOKIE_DOMAIN", test.cookieEnv)
			t.Setenv("CANVAS_PUBLIC_BACKUP_PARENT_DOMAIN", "")
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodGet, "https://"+test.host+"/", nil)
			context.Request.Host = test.host
			if got := sessionCookieDomain(context); got != test.want {
				t.Fatalf("sessionCookieDomain(%q) = %q, want %q", test.host, got, test.want)
			}
		})
	}
}
