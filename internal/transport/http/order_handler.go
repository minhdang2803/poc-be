package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"mmispoc/internal/service"
)

// OrderCreateHandler handles POST /order/create requests.
type OrderCreateHandler struct {
	orderService *service.OrderService
	userService  *service.UserService
}

// NewOrderCreateHandler builds an order handler.
func NewOrderCreateHandler(userService *service.UserService, orderService *service.OrderService) http.Handler {
	return &OrderCreateHandler{
		orderService: orderService,
		userService:  userService,
	}
}

func (h *OrderCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	authHeader := r.Header.Get("Authorization")
	const bearerPrefix = "Bearer "
	if authHeader == "" {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}

	authHeader = strings.TrimSpace(authHeader)
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}
	accessToken := strings.TrimSpace(authHeader[len(bearerPrefix):])

	var payload struct {
		RestaurantID int64 `json:"restaurant_id"`
		Orders       []struct {
			IngredientID int64 `json:"ingredient_id"`
			Number       int   `json:"number"`
		} `json:"orders"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	user, err := h.userService.ValidateAccessToken(r.Context(), accessToken)
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

	items := make([]service.OrderItem, 0, len(payload.Orders))
	for _, row := range payload.Orders {
		items = append(items, service.OrderItem{
			IngredientID: row.IngredientID,
			Number:       row.Number,
		})
	}

	if user.RestaurantID != 0 && payload.RestaurantID != 0 && user.RestaurantID != payload.RestaurantID {
		writeError(w, http.StatusForbidden, "restaurant mismatch")
		return
	}

	if payload.RestaurantID == 0 && user.RestaurantID != 0 {
		payload.RestaurantID = user.RestaurantID
	}

	if err := h.orderService.CreateOrders(r.Context(), payload.RestaurantID, items); err != nil {
		switch {
		case errors.Is(err, service.ErrOrderInvalidRestaurantID):
			writeError(w, http.StatusBadRequest, "invalid restaurant id")
		case errors.Is(err, service.ErrOrderRestaurantNotFound):
			writeError(w, http.StatusBadRequest, "restaurant not found")
		case errors.Is(err, service.ErrOrderEmptyItems):
			writeError(w, http.StatusBadRequest, "orders must include at least one item")
		case errors.Is(err, service.ErrOrderInvalidIngredientID):
			writeError(w, http.StatusBadRequest, "invalid ingredient id")
		case errors.Is(err, service.ErrOrderInvalidNumber):
			writeError(w, http.StatusBadRequest, "invalid number")
		case errors.Is(err, service.ErrOrderIngredientNotFound):
			writeError(w, http.StatusBadRequest, "ingredient not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"created": len(items),
	})
}
