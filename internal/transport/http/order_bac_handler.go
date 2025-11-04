package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mmispoc/internal/service"
)

// OrderBACHandler exposes GET /order-bac/{id}.
type OrderBACHandler struct {
	userService  *service.UserService
	orderService *service.OrderService
}

// NewOrderBACHandler builds a handler for broken access control testing.
func NewOrderBACHandler(userService *service.UserService, orderService *service.OrderService) http.Handler {
	return &OrderBACHandler{
		userService:  userService,
		orderService: orderService,
	}
}

func (h *OrderBACHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	restaurantID, err := extractRestaurantID(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid restaurant id")
		return
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	const bearerPrefix = "Bearer "
	if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}
	token := strings.TrimSpace(authHeader[len(bearerPrefix):])

	if _, err := h.userService.ValidateAccessToken(r.Context(), token); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		case errors.Is(err, service.ErrTokenExpired):
			writeError(w, http.StatusUnauthorized, "token expired")
			return
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	orders, restaurantName, err := h.orderService.GetOrdersByRestaurant(r.Context(), restaurantID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderInvalidRestaurantID):
			writeError(w, http.StatusBadRequest, "invalid restaurant id")
		case errors.Is(err, service.ErrOrderRestaurantNotFound):
			writeError(w, http.StatusNotFound, "restaurant not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	type orderDTO struct {
		ID           int64  `json:"id"`
		Code         string `json:"code"`
		RestaurantID int64  `json:"restaurant_id"`
		IngredientID int64  `json:"ingredient_id"`
		Number       int    `json:"number"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at,omitempty"`
	}

	response := make([]orderDTO, 0, len(orders))
	for _, order := range orders {
		dto := orderDTO{
			ID:           order.ID,
			Code:         order.Code,
			RestaurantID: order.RestaurantID,
			IngredientID: order.IngredientID,
			Number:       order.Number,
			CreatedAt:    order.CreatedAt.Format(time.RFC3339),
		}
		if !order.UpdatedAt.IsZero() {
			dto.UpdatedAt = order.UpdatedAt.Format(time.RFC3339)
		}
		response = append(response, dto)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":           len(response),
		"restaurant_name": restaurantName,
		"orders":          response,
	})
}

func extractRestaurantID(path string) (int64, error) {
	const prefix = "/order-bac/"
	if !strings.HasPrefix(path, prefix) {
		return 0, errors.New("invalid path")
	}
	idPart := strings.TrimSpace(path[len(prefix):])
	if idPart == "" {
		return 0, errors.New("missing id")
	}
	return strconv.ParseInt(idPart, 10, 64)
}
