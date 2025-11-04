package httptransport

import (
	"net/http"

	"mmispoc/internal/service"
)

// NewRouter wires HTTP routes.
func NewRouter(userService *service.UserService, orderService *service.OrderService) http.Handler {
	mux := http.NewServeMux()

	signupHandler := NewSignupHandler(userService)
	loginHandler := NewLoginHandler(userService)
	orderCreateHandler := NewOrderCreateHandler(userService, orderService)
	orderBACHandler := NewOrderBACHandler(userService, orderService)
	orderDetailHandler := NewOrderDetailHandler(userService, orderService)
	profileHandler := NewProfileHandler(userService)

	mux.Handle("/signup", signupHandler)
	mux.Handle("/login", loginHandler)
	mux.Handle("/profile", profileHandler)
	mux.Handle("/order/create", orderCreateHandler)
	mux.Handle("/order/", orderDetailHandler)
	mux.Handle("/order-bac/", orderBACHandler)

	return withDefaultHeaders(mux)
}

func withDefaultHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		next.ServeHTTP(w, r)
	})
}
