package adminapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestEmbeddedConsoleAssets(t *testing.T) {
	ts := newTestAdminServer(t)
	for path, contentType := range map[string]string{"/web/app.js": "javascript", "/web/accounts.js": "javascript", "/web/api.js": "javascript", "/web/i18n.js": "javascript", "/web/ui.js": "javascript", "/web/console.css": "text/css"} {
		t.Run(path, func(t *testing.T) {
			res := doRequest(t, ts, "GET", path, "")
			if res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d", res.StatusCode)
			}
			if !strings.Contains(res.Header.Get("Content-Type"), contentType) {
				t.Errorf("unexpected content type: %s", res.Header.Get("Content-Type"))
			}
			if body := readBody(t, res); len(body) == 0 || strings.Contains(body, testToken) {
				t.Error("empty asset or leaked token")
			}
		})
	}
	res := doRequest(t, ts, "GET", "/", "")
	if !strings.Contains(res.Header.Get("Content-Security-Policy"), "script-src 'self'") {
		t.Error("console CSP missing")
	}
	if doRequest(t, ts, "GET", "/web/missing.js", "").StatusCode != http.StatusNotFound {
		t.Error("unknown asset must return 404")
	}
}

func TestRejectsTrailingJSONAndZeroItem(t *testing.T) {
	ts := newTestAdminServer(t)
	for _, body := range []string{`{"itemId":1} {"itemId":2}`, `{"itemId":1} garbage`} {
		status, _ := call(t, ts, "POST", "/api/accounts/nobody/items", body)
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", status)
		}
	}
	status, _ := call(t, ts, "DELETE", "/api/accounts/nobody/items/0", "")
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
}

func TestCloseMissingChannelReturnsNotFound(t *testing.T) {
	ts := newTestAdminServer(t)
	status, _ := call(t, ts, "DELETE", "/api/rooms/255/255/1", "")
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}
