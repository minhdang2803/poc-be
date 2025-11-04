package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"

	"mmispoc/internal/service"
)

// LoginHandler handles POST /login requests.
type LoginHandler struct {
	userService *service.UserService
}

// NewLoginHandler builds a login handler.
func NewLoginHandler(userService *service.UserService) http.Handler {
	return &LoginHandler{userService: userService}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	token, err := h.userService.Authenticate(r.Context(), payload.Username, payload.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": token,
	})
}
