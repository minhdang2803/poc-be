package httptransport

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"mmispoc/internal/service"
)

// SignupHandler handles POST /signup requests.
type SignupHandler struct {
	userService *service.UserService
}

// NewSignupHandler builds a handler.
func NewSignupHandler(userService *service.UserService) http.Handler {
	return &SignupHandler{userService: userService}
}

func (h *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload struct {
		Username     string `json:"username"`
		Password     string `json:"password"`
		RestaurantID int64  `json:"restaurant_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	user, err := h.userService.SignUp(r.Context(), payload.Username, payload.Password, payload.RestaurantID)
	if err != nil {
		log.Printf("error: %v", err)
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":            user.ID,
		"username":      user.Username,
		"restaurant_id": user.RestaurantID,
	})
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidUsername):
		writeError(w, http.StatusBadRequest, "invalid username")
	case errors.Is(err, service.ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, "invalid password")
	case errors.Is(err, service.ErrInvalidRestaurantID):
		writeError(w, http.StatusBadRequest, "invalid restaurant id")
	case errors.Is(err, service.ErrRestaurantNotFound):
		writeError(w, http.StatusBadRequest, "restaurant not found")
	case errors.Is(err, service.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "username already exists")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
