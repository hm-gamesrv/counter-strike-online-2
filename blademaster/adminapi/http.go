package adminapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/KouKouChan/CSO2-Server/logging"
)

// bearerToken extracts the credential from an "Authorization: Bearer x" header.
func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}

	const prefix = "bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeOperationError maps an operation error onto an HTTP status.
func writeOperationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotOnline):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrRoomNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNameInUse), errors.Is(err, ErrNicknameInUse):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidAccount):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		logging.Error("admin: operation failed:", err)
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func parseByte(value string) (uint8, error) {
	n, err := strconv.ParseUint(value, 10, 8)
	if err != nil || n == 0 {
		return 0, errors.New("not a valid byte")
	}
	return uint8(n), nil
}

// decodeBody reads a size-limited JSON body.
func decodeBody(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxAdminBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dest); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain exactly one JSON value")
	}
	return nil
}
