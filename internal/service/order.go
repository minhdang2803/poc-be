package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"mmispoc/internal/repository"
)

// OrderItem represents a single incoming order line.
type OrderItem struct {
	IngredientID int64
	Number       int
}

// OrderService orchestrates order creation.
type OrderService struct {
	orderRepo      *repository.OrderRepository
	restaurantRepo *repository.RestaurantRepository
	ingredientRepo *repository.IngredientRepository
}

// NewOrder constructs an order service.
func NewOrder(orderRepo *repository.OrderRepository, restaurantRepo *repository.RestaurantRepository, ingredientRepo *repository.IngredientRepository) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		restaurantRepo: restaurantRepo,
		ingredientRepo: ingredientRepo,
	}
}

var (
	// ErrOrderInvalidRestaurantID indicates restaurant id is missing or invalid.
	ErrOrderInvalidRestaurantID = errors.New("invalid restaurant id")
	// ErrOrderInvalidID indicates the order id is invalid.
	ErrOrderInvalidID = errors.New("invalid order id")
	// ErrOrderRestaurantNotFound indicates restaurant id not found.
	ErrOrderRestaurantNotFound = errors.New("restaurant not found")
	// ErrOrderEmptyItems indicates payload has no order items.
	ErrOrderEmptyItems = errors.New("order items must not be empty")
	// ErrOrderInvalidIngredientID indicates ingredient id invalid.
	ErrOrderInvalidIngredientID = errors.New("invalid ingredient id")
	// ErrOrderInvalidNumber indicates number invalid.
	ErrOrderInvalidNumber = errors.New("invalid number")
	// ErrOrderIngredientNotFound indicates ingredient missing in db.
	ErrOrderIngredientNotFound = errors.New("ingredient not found")
	// ErrOrderNotFound indicates the order cannot be found.
	ErrOrderNotFound = errors.New("order not found")
	// ErrOrderForbidden indicates the order is not owned by the requesting restaurant.
	ErrOrderForbidden = errors.New("order access forbidden")
)

// CreateOrders validates input and persists orders.
func (s *OrderService) CreateOrders(ctx context.Context, restaurantID int64, items []OrderItem) error {
	if restaurantID <= 0 {
		return ErrOrderInvalidRestaurantID
	}
	if len(items) == 0 {
		return ErrOrderEmptyItems
	}

	exists, err := s.restaurantRepo.Exists(ctx, restaurantID)
	if err != nil {
		return fmt.Errorf("check restaurant: %w", err)
	}
	if !exists {
		return ErrOrderRestaurantNotFound
	}

	now := time.Now().UTC().UnixNano()

	persistItems := make([]repository.Order, 0, len(items))
	for idx, item := range items {
		if item.IngredientID <= 0 {
			return ErrOrderInvalidIngredientID
		}
		if item.Number <= 0 {
			return ErrOrderInvalidNumber
		}

		ingredientExists, err := s.ingredientRepo.Exists(ctx, item.IngredientID)
		if err != nil {
			return fmt.Errorf("check ingredient: %w", err)
		}
		if !ingredientExists {
			return ErrOrderIngredientNotFound
		}

		code := fmt.Sprintf("ORD-%d-%d-%d", restaurantID, now, idx)
		persistItems = append(persistItems, repository.Order{
			Code:         code,
			RestaurantID: restaurantID,
			IngredientID: item.IngredientID,
			Number:       item.Number,
		})
	}

	if err := s.orderRepo.CreateBulk(ctx, restaurantID, persistItems); err != nil {
		return fmt.Errorf("store orders: %w", err)
	}

	return nil
}

// GetOrdersByRestaurant returns all orders for a restaurant.
func (s *OrderService) GetOrdersByRestaurant(ctx context.Context, restaurantID int64) ([]repository.Order, string, error) {
	if restaurantID <= 0 {
		return nil, "", ErrOrderInvalidRestaurantID
	}

	exists, err := s.restaurantRepo.Exists(ctx, restaurantID)
	if err != nil {
		return nil, "", fmt.Errorf("check restaurant: %w", err)
	}
	if !exists {
		return nil, "", ErrOrderRestaurantNotFound
	}

	name, err := s.restaurantRepo.GetName(ctx, restaurantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrOrderRestaurantNotFound
		}
		return nil, "", fmt.Errorf("get restaurant name: %w", err)
	}

	orders, err := s.orderRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, "", fmt.Errorf("list orders: %w", err)
	}

	return orders, name, nil
}

// GetOrder retrieves a single order ensuring ownership by restaurant.
func (s *OrderService) GetOrder(ctx context.Context, orderID, restaurantID int64) (*repository.Order, error) {
	if orderID <= 0 {
		return nil, ErrOrderInvalidID
	}

	if restaurantID <= 0 {
		return nil, ErrOrderInvalidRestaurantID
	}

	order, err := s.orderRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	if order.RestaurantID != restaurantID {
		return nil, ErrOrderForbidden
	}

	return order, nil
}
