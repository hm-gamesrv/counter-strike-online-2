package adminapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "cso2-server/config"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

// testToken is 21 characters of letters, digits and underscores: a valid
// ADMIN_TOKEN.
const testToken = "test_token_1234567890"

type testAdminServer struct {
	URL     string
	handler http.Handler
}

// Execute real requests through the complete router without opening a socket.
func (s *testAdminServer) Client() *http.Client {
	return &http.Client{Transport: s}
}

func (s *testAdminServer) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, r)
	return recorder.Result(), nil
}
func newTestAdminServer(t *testing.T) *testAdminServer {
	t.Helper()

	Level = 0
	// The status endpoint reports the deployment settings, so give it something
	// to report.
	Env.AdminPort = 1315
	Env.TCPPort = 30001
	Env.UDPPort = 30002

	// Account CRUD needs a real DATA_DIR.
	if err := InitDataDir(t.TempDir()); err != nil {
		t.Fatalf("InitDataDir failed: %v", err)
	}

	s := &adminServer{token: testToken, startedAt: time.Now()}
	ts := &testAdminServer{URL: "http://admin.test", handler: s.routes()}

	return ts
}

// call performs an authenticated JSON request against the admin API.
func call(t *testing.T, ts *testAdminServer, method, path, body string) (int, string) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, ts.URL+path, reader)
	if err != nil {
		t.Fatalf("cannot build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testToken)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("cannot read response: %v", err)
	}
	return res.StatusCode, string(payload)
}

func doRequest(t *testing.T, ts *testAdminServer, method, path, token string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("cannot build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func TestValidateAdminToken(t *testing.T) {
	valid := []string{
		"abcdefghijklmnop",                       // exactly 16
		strings.Repeat("a", 128),                 // exactly 128
		"test_token_1234567890",                  // mixed
		"ABCDEFGHIJKLMNOP_0123456789_abcdefghij", // upper, lower, digits, underscore
	}

	for _, token := range valid {
		if err := ValidateAdminToken(token); err != nil {
			t.Errorf("ValidateAdminToken(%q) = %v, want nil", token, err)
		}
	}

	invalid := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"too short", "abcdefghijklmno"},             // 15
		{"too long", strings.Repeat("a", 129)},       // 129
		{"contains dash", "abcdefghijklmnop-123456"}, // dash
		{"contains space", "abcdefghijklmnop 123456"},
		{"contains dot", "abcdefghijklmnop.123456"},
		{"contains slash", "abcdefghijklmnop/123456"},
		{"contains colon", "abcdefghijklmnop:123456"},
		{"non-ascii", "abcdefghijklmnop中文"},
		{"newline", "abcdefghijklmnop\n"},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAdminToken(tc.token); err == nil {
				t.Fatalf("ValidateAdminToken(%q) = nil, want an error", tc.token)
			}
		})
	}
}

func TestAdminAPIRoutesRequireToken(t *testing.T) {
	ts := newTestAdminServer(t)

	protected := []struct {
		method string
		path   string
	}{
		{"GET", "/api/status"},
		{"GET", "/api/players"},
		{"POST", "/api/players/alice/kick"},
		{"GET", "/api/accounts"},
		{"POST", "/api/accounts"},
		{"GET", "/api/accounts/alice"},
		{"PATCH", "/api/accounts/alice"},
		{"DELETE", "/api/accounts/alice"},
		{"POST", "/api/accounts/alice/items"},
		{"DELETE", "/api/accounts/alice/items/1"},
		{"POST", "/api/accounts/alice/vip"},
		{"POST", "/api/accounts/alice/gm"},
		{"DELETE", "/api/rooms/1/1/1"},
		{"POST", "/api/save"},
	}

	for _, route := range protected {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			res := doRequest(t, ts, route.method, route.path, "")
			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("without a token: status = %d, want 401", res.StatusCode)
			}

			res = doRequest(t, ts, route.method, route.path, "wrong_token_1234567890")
			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("with a wrong token: status = %d, want 401", res.StatusCode)
			}
		})
	}
}

func TestAdminAPIAcceptsValidToken(t *testing.T) {
	ts := newTestAdminServer(t)

	res := doRequest(t, ts, "GET", "/api/status", testToken)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// The scheme name is case-insensitive per RFC 7235.
func TestAdminAPIAcceptsLowercaseBearerScheme(t *testing.T) {
	ts := newTestAdminServer(t)

	req, err := http.NewRequest("GET", ts.URL+"/api/status", nil)
	if err != nil {
		t.Fatalf("cannot build request: %v", err)
	}
	req.Header.Set("Authorization", "bearer "+testToken)

	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
}

func TestAdminAPIRejectsRawTokenWithoutScheme(t *testing.T) {
	ts := newTestAdminServer(t)

	req, err := http.NewRequest("GET", ts.URL+"/api/status", nil)
	if err != nil {
		t.Fatalf("cannot build request: %v", err)
	}
	req.Header.Set("Authorization", testToken)

	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", res.StatusCode)
	}
}

// The page itself is public: it holds no secrets, only the API is protected.
func TestAdminPageIsPublic(t *testing.T) {
	ts := newTestAdminServer(t)

	res := doRequest(t, ts, "GET", "/", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}

	body := readBody(t, res)
	if !strings.Contains(body, "CSO2") {
		t.Error("the page does not look like the admin console")
	}
	// The page must not embed the token; it is supplied at runtime by the user.
	if strings.Contains(body, testToken) {
		t.Error("the page leaked the admin token")
	}
}

func TestAdminStatusPayload(t *testing.T) {
	ts := newTestAdminServer(t)

	res := doRequest(t, ts, "GET", "/api/status", testToken)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}

	var payload statusResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode status payload: %v", err)
	}

	if payload.Version == "" {
		t.Error("version is empty")
	}
	if payload.AdminPort == 0 {
		t.Error("adminPort is 0")
	}
	if payload.UptimeSeconds < 0 {
		t.Errorf("uptimeSeconds = %d, want >= 0", payload.UptimeSeconds)
	}
}

func TestAdminPlayersPayloadIsEmptyAndWellFormed(t *testing.T) {
	ts := newTestAdminServer(t)

	res := doRequest(t, ts, "GET", "/api/players", testToken)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}

	var payload struct {
		Count   int          `json:"count"`
		Players []OnlineUser `json:"players"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode players payload: %v", err)
	}
	if payload.Count != 0 {
		t.Errorf("count = %d, want 0 with no players online", payload.Count)
	}
}

func TestAdminUnknownAccountIsNotFound(t *testing.T) {
	ts := newTestAdminServer(t)

	cases := []struct {
		method string
		path   string
	}{
		{"GET", "/api/accounts/nobody"},
		{"PATCH", "/api/accounts/nobody"},
		{"DELETE", "/api/accounts/nobody"},
		{"DELETE", "/api/accounts/nobody/items/1"},
		{"POST", "/api/accounts/nobody/vip"},
		{"POST", "/api/accounts/nobody/gm"},
	}

	for _, tc := range cases {
		status, body := call(t, ts, tc.method, tc.path, "{}")
		if status != http.StatusNotFound {
			t.Errorf("%s %s: status = %d, want 404 (%s)", tc.method, tc.path, status, body)
		}
	}
}

func TestAdminKickRequiresOnlineUser(t *testing.T) {
	ts := newTestAdminServer(t)

	// Nobody is connected, so a kick is a conflict rather than a 404: the
	// account may well exist, it just is not online.
	res := doRequest(t, ts, "POST", "/api/players/nobody/kick", testToken)
	if res.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", res.StatusCode)
	}
}

// The whole point of the change: accounts are created from the admin API.
func TestAdminAccountLifecycle(t *testing.T) {
	ts := newTestAdminServer(t)

	// --- create ---
	status, body := call(t, ts, "POST", "/api/accounts",
		`{"username":"alice","nickname":"AliceInGame","password":"s3cret","mail":"a@b.c"}`)
	if status != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201 (%s)", status, body)
	}

	var created AccountDetail
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("cannot decode created account: %v", err)
	}
	if created.UserName != "alice" || created.IngameName != "AliceInGame" {
		t.Errorf("created account has wrong names: %+v", created)
	}
	if !IsUUID(created.Uuid) {
		t.Errorf("created account has no valid uuid: %q", created.Uuid)
	}
	if created.Level != 1 {
		t.Errorf("level = %d, want the default 1", created.Level)
	}
	if created.Online {
		t.Error("a freshly created account must not be marked online")
	}

	// --- duplicate names are refused ---
	status, _ = call(t, ts, "POST", "/api/accounts",
		`{"username":"alice","nickname":"SomeoneElse","password":"x"}`)
	if status != http.StatusConflict {
		t.Errorf("duplicate username: status = %d, want 409", status)
	}
	status, _ = call(t, ts, "POST", "/api/accounts",
		`{"username":"carol","nickname":"AliceInGame","password":"x"}`)
	if status != http.StatusConflict {
		t.Errorf("duplicate nickname: status = %d, want 409", status)
	}

	// --- list ---
	status, body = call(t, ts, "GET", "/api/accounts", "")
	if status != http.StatusOK {
		t.Fatalf("list: status = %d, want 200", status)
	}
	var list struct {
		Count    int          `json:"count"`
		Accounts []AccountRef `json:"accounts"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatalf("cannot decode account list: %v", err)
	}
	if list.Count != 1 || len(list.Accounts) != 1 {
		t.Fatalf("account list = %+v, want exactly alice", list)
	}
	if list.Accounts[0].UserName != "alice" {
		t.Errorf("listed account = %q, want alice", list.Accounts[0].UserName)
	}

	// --- patch ---
	status, body = call(t, ts, "PATCH", "/api/accounts/alice",
		`{"nickname":"Renamed","level":42,"points":777,"gm":true}`)
	if status != http.StatusOK {
		t.Fatalf("patch: status = %d, want 200 (%s)", status, body)
	}

	var updated AccountDetail
	if err := json.Unmarshal([]byte(body), &updated); err != nil {
		t.Fatalf("cannot decode updated account: %v", err)
	}
	if updated.IngameName != "Renamed" {
		t.Errorf("nickname = %q, want Renamed", updated.IngameName)
	}
	if updated.Level != 42 || updated.Points != 777 {
		t.Errorf("level/points = %d/%d, want 42/777", updated.Level, updated.Points)
	}
	if !updated.GM {
		t.Error("gm was not applied")
	}
	// Untouched fields must survive a partial patch.
	if updated.Mail != "a@b.c" {
		t.Errorf("mail = %q, want it untouched (a@b.c)", updated.Mail)
	}

	// --- patch rejects a name that is taken ---
	status, _ = call(t, ts, "POST", "/api/accounts", `{"username":"bob","nickname":"BobNick","password":"x"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating bob failed: %d", status)
	}
	status, _ = call(t, ts, "PATCH", "/api/accounts/alice", `{"nickname":"BobNick"}`)
	if status != http.StatusConflict {
		t.Errorf("patching to a taken nickname: status = %d, want 409", status)
	}

	// --- delete ---
	status, body = call(t, ts, "DELETE", "/api/accounts/alice", "")
	if status != http.StatusOK {
		t.Fatalf("delete: status = %d, want 200 (%s)", status, body)
	}
	status, _ = call(t, ts, "GET", "/api/accounts/alice", "")
	if status != http.StatusNotFound {
		t.Errorf("getting a deleted account: status = %d, want 404", status)
	}

	status, body = call(t, ts, "GET", "/api/accounts", "")
	var after struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal([]byte(body), &after)
	if after.Count != 1 {
		t.Errorf("account count after delete = %d, want 1 (bob)", after.Count)
	}

	// The nickname must have been released along with the account.
	status, _ = call(t, ts, "POST", "/api/accounts", `{"username":"dave","nickname":"Renamed","password":"x"}`)
	if status != http.StatusCreated {
		t.Errorf("reusing a deleted account's nickname: status = %d, want 201", status)
	}
}

func TestAdminCreateAccountValidation(t *testing.T) {
	ts := newTestAdminServer(t)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"missing username", `{"nickname":"N","password":"p"}`, http.StatusBadRequest},
		{"missing nickname", `{"username":"u","password":"p"}`, http.StatusBadRequest},
		{"missing password", `{"username":"u","nickname":"N"}`, http.StatusBadRequest},
		{"whitespace username", `{"username":"   ","nickname":"N","password":"p"}`, http.StatusBadRequest},
		{"over-long username", `{"username":"` + strings.Repeat("a", 65) + `","nickname":"N","password":"p"}`, http.StatusBadRequest},
		{"not json", `{nope`, http.StatusBadRequest},
		{"unknown field", `{"username":"u","nickname":"N","password":"p","zzz":1}`, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if status, _ := call(t, ts, "POST", "/api/accounts", tc.body); status != tc.want {
				t.Errorf("status = %d, want %d", status, tc.want)
			}
		})
	}
}

func TestAdminPatchAccountValidation(t *testing.T) {
	ts := newTestAdminServer(t)

	if status, body := call(t, ts, "POST", "/api/accounts",
		`{"username":"alice","nickname":"AliceInGame","password":"p"}`); status != http.StatusCreated {
		t.Fatalf("setup failed: %d %s", status, body)
	}

	cases := []struct {
		name string
		body string
		want int
	}{
		{"empty nickname", `{"nickname":""}`, http.StatusBadRequest},
		{"empty password", `{"password":""}`, http.StatusBadRequest},
		{"unknown field", `{"nope":1}`, http.StatusBadRequest},
		{"not json", `{`, http.StatusBadRequest},
		// A empty patch is legal and simply changes nothing.
		{"empty patch", `{}`, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if status, _ := call(t, ts, "PATCH", "/api/accounts/alice", tc.body); status != tc.want {
				t.Errorf("status = %d, want %d", status, tc.want)
			}
		})
	}
}

func TestAdminBadRequestPayloads(t *testing.T) {
	ts := newTestAdminServer(t)

	cases := []struct {
		name string
		body string
	}{
		{"not json", "{nope"},
		{"unknown field", `{"itemId":1,"count":1,"extra":true}`},
		{"itemId missing", `{"count":1}`},
		{"itemId zero", `{"itemId":0,"count":1}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if status, _ := call(t, ts, "POST", "/api/accounts/alice/items", tc.body); status != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", status)
			}
		})
	}
}

func TestAdminRoomPathValidation(t *testing.T) {
	ts := newTestAdminServer(t)

	cases := []string{
		"/api/rooms/0/1/1",     // channel server 0
		"/api/rooms/1/0/1",     // channel 0
		"/api/rooms/1/1/0",     // room 0
		"/api/rooms/abc/1/1",   // not a number
		"/api/rooms/1/1/99999", // room out of range
	}

	for _, path := range cases {
		res := doRequest(t, ts, "DELETE", path, testToken)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", path, res.StatusCode)
		}
	}
}

func TestBearerTokenParsing(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"Bearer abc", "abc"},
		{"bearer abc", "abc"},
		{"BEARER abc", "abc"},
		{"Bearer   abc  ", "abc"},
		{"", ""},
		{"abc", ""},
		{"Basic abc", ""},
		{"Bearer", ""},
	}

	for _, tc := range cases {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatalf("cannot build request: %v", err)
		}
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}

		if got := bearerToken(req); got != tc.want {
			t.Errorf("bearerToken(%q) = %q, want %q", tc.header, got, tc.want)
		}
	}
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()

	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := res.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}
