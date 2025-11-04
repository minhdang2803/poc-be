package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mmispoc/internal/service"
)

// OrderDetailHandler handles GET /order/{id} requests where id is restaurant id.
type OrderDetailHandler struct {
	userService  *service.UserService
	orderService *service.OrderService
}

// NewOrderDetailHandler builds a handler.
func NewOrderDetailHandler(userService *service.UserService, orderService *service.OrderService) http.Handler {
	return &OrderDetailHandler{
		userService:  userService,
		orderService: orderService,
	}
}

func (h *OrderDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	restaurantID, err := extractRestaurantIDFromOrderPath(r.URL.Path)
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

	if user.RestaurantID != 0 && user.RestaurantID != restaurantID {
		writeError(w, http.StatusForbidden, "order data does not belong to your restaurant")
		return
	}

	orders, restaurantName, err := h.orderService.GetOrdersByRestaurant(r.Context(), restaurantID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderInvalidRestaurantID):
			writeError(w, http.StatusForbidden, "restaurant not linked to user")
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

	result := make([]orderDTO, 0, len(orders))
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
		result = append(result, dto)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":           len(result),
		"restaurant_name": restaurantName,
		"orders":          result,
	})
}

func extractRestaurantIDFromOrderPath(path string) (int64, error) {
	const prefix = "/order/"
	if !strings.HasPrefix(path, prefix) {
		return 0, errors.New("invalid path")
	}
	idPart := strings.TrimSpace(path[len(prefix):])
	if idPart == "" {
		return 0, errors.New("missing id")
	}
	return strconv.ParseInt(idPart, 10, 64)
}
