package adminapi

import (
	"embed"
	"net/http"
)

// consoleFS contains all console assets so releases need only the executable.
//
//go:embed page.html web/*
var consoleFS embed.FS

var pageHTML, _ = consoleFS.ReadFile("page.html")

func (s *adminServer) handlePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	_, _ = w.Write(pageHTML)
}

func (s *adminServer) handleAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.FileServer(http.FS(consoleFS)).ServeHTTP(w, r)
}
