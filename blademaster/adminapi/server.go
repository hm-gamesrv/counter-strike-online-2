package adminapi

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/KouKouChan/CSO2-Server/config"
	"github.com/KouKouChan/CSO2-Server/logging"
)

// AdminTokenEnv is the environment variable holding the admin bearer token.
const AdminTokenEnv = "ADMIN_TOKEN"

const (
	adminTokenMinLength = 16
	adminTokenMaxLength = 128

	maxAdminBodyBytes = 64 * 1024
)

// ValidateAdminToken checks the ADMIN_TOKEN format: 16-128 characters drawn
// from letters, digits and underscores.
func ValidateAdminToken(token string) error {
	if token == "" {
		return fmt.Errorf("%s is not set", AdminTokenEnv)
	}
	if len(token) < adminTokenMinLength || len(token) > adminTokenMaxLength {
		return fmt.Errorf("%s must be between %d and %d characters, got %d",
			AdminTokenEnv, adminTokenMinLength, adminTokenMaxLength, len(token))
	}
	for i := 0; i < len(token); i++ {
		c := token[i]
		valid := (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '_'
		if !valid {
			return fmt.Errorf("%s may only contain letters, digits and underscores; %q is not allowed",
				AdminTokenEnv, string(c))
		}
	}
	return nil
}

type adminServer struct {
	token     string
	startedAt time.Time
}

// Start runs the admin web API.
//
// It is deliberately fail-closed: if ADMIN_TOKEN is missing or malformed the
// service does not start at all, because an unauthenticated administration API
// is worse than no administration API.
func Start() {
	defer func() {
		if err := recover(); err != nil {
			logging.Error("Admin web API suffered a fault:", err)
		}
	}()

	token := os.Getenv(AdminTokenEnv)
	if err := ValidateAdminToken(token); err != nil {
		logging.Error("Admin web API is disabled:", err)
		logging.Error("Set", AdminTokenEnv, "to 16-128 characters of letters, digits and underscores, then restart.")
		return
	}

	s := &adminServer{token: token, startedAt: time.Now()}
	mux := s.routes()

	addr := fmt.Sprintf(":%d", config.Env.AdminPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logging.Info("Admin web API is running at http://[AnyAdapter]:" + strconv.Itoa(int(config.Env.AdminPort)))
	logging.Info("Open the admin page in a browser and paste the value of " + AdminTokenEnv)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logging.Error("Admin web API stopped:", err)
	}
}

func (s *adminServer) routes() *http.ServeMux {
	mux := http.NewServeMux()

	// The page itself carries no secrets: it only renders what the API returns,
	// and the API is what is protected.
	mux.HandleFunc("GET /{$}", s.handlePage)
	mux.HandleFunc("GET /web/", s.handleAsset)
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/status", s.auth(s.handleStatus))

	// "players" is who is connected right now; "accounts" is the persistent
	// records.
	mux.HandleFunc("GET /api/players", s.auth(s.handleListPlayers))
	mux.HandleFunc("POST /api/players/{name}/kick", s.auth(s.handleKick))

	mux.HandleFunc("GET /api/accounts", s.auth(s.handleListAccounts))
	mux.HandleFunc("POST /api/accounts", s.auth(s.handleCreateAccount))
	mux.HandleFunc("GET /api/accounts/{name}", s.auth(s.handleGetAccount))
	mux.HandleFunc("PATCH /api/accounts/{name}", s.auth(s.handleUpdateAccount))
	mux.HandleFunc("DELETE /api/accounts/{name}", s.auth(s.handleDeleteAccount))
	mux.HandleFunc("POST /api/accounts/{name}/items", s.auth(s.handleAddItem))
	mux.HandleFunc("DELETE /api/accounts/{name}/items/{itemId}", s.auth(s.handleRemoveItem))
	mux.HandleFunc("POST /api/accounts/{name}/vip", s.auth(s.handleSetVIP))
	mux.HandleFunc("POST /api/accounts/{name}/gm", s.auth(s.handleSetGM))

	mux.HandleFunc("DELETE /api/rooms/{channelServer}/{channel}/{room}", s.auth(s.handleDeleteRoom))
	mux.HandleFunc("POST /api/save", s.auth(s.handleSave))

	return mux
}

// auth wraps a handler with bearer-token authentication.
func (s *adminServer) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		presented := bearerToken(r)

		// Constant-time compare so the token cannot be guessed byte by byte
		// from the response timing.
		if presented == "" ||
			subtle.ConstantTimeCompare([]byte(presented), []byte(s.token)) != 1 {
			logging.Warn("admin: rejected", r.Method, r.URL.Path, "from", r.RemoteAddr, "(bad or missing token)")
			w.Header().Set("WWW-Authenticate", `Bearer realm="cso2-admin"`)
			writeError(w, http.StatusUnauthorized, "invalid or missing admin token")
			return
		}
		next(w, r)
	}
}
