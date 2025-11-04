package httptransport

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"mmispoc/internal/service"
)

// ProfileHandler handles GET /profile requests.
type ProfileHandler struct {
	userService *service.UserService
}

// NewProfileHandler builds a profile handler.
func NewProfileHandler(userService *service.UserService) http.Handler {
	return &ProfileHandler{userService: userService}
}

func (h *ProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	const bearerPrefix = "Bearer "
	if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}

	token := strings.TrimSpace(authHeader[len(bearerPrefix):])

	user, err := h.userService.ValidateAccessToken(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			writeError(w, http.StatusUnauthorized, "invalid token")
		case errors.Is(err, service.ErrTokenExpired):
			writeError(w, http.StatusUnauthorized, "token expired")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	profile, err := h.userService.GetProfile(r.Context(), user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRestaurantNotFound):
			writeError(w, http.StatusNotFound, "restaurant not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":              profile.ID,
		"user_name":       profile.Username,
		"restaurant_id":   profile.RestaurantID,
		"restaurant_name": profile.RestaurantName,
		"created_at":      profile.CreatedAt.Format(time.RFC3339),
	})
}
