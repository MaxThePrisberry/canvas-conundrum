package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleSegment serves GET /api/segments/{segmentId}; handlePreview serves
// GET /api/preview/full. Authorization is decided by the engine (it owns
// phase, assignments, and grid visibility); the bytes it returns are
// immutable so they are written outside the engine goroutine.

func (s *Server) handleSegment(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeAssetError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or malformed Authorization header.")
		return
	}
	verdict := s.engine.QueryAsset(token, r.PathValue("segmentId"), false)
	writeAssetVerdict(w, verdict.Status, string(verdict.Code), verdict.Message, verdict.Bytes)
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeAssetError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or malformed Authorization header.")
		return
	}
	verdict := s.engine.QueryAsset(token, "", true)
	writeAssetVerdict(w, verdict.Status, string(verdict.Code), verdict.Message, verdict.Bytes)
}

// bearerToken extracts the Authorization header token. Tokens are accepted
// from the header only — query-string tokens leak into proxy logs and are
// deliberately ignored.
func bearerToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) || len(auth) == len(prefix) {
		return "", false
	}
	return auth[len(prefix):], true
}

func writeAssetVerdict(w http.ResponseWriter, status int, code, message string, body []byte) {
	if status == http.StatusOK {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}
	writeAssetError(w, status, code, message)
}

func writeAssetError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}
