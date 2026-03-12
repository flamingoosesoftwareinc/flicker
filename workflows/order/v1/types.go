package v1

import (
	"context"
	"time"
)

// Request is the input to the order processing workflow.
type Request struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
}

// User is a customer fetched from an external service.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Reservation represents inventory reserved for the order.
type Reservation struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PaymentResult is the outcome of a payment check.
type PaymentResult struct {
	Status    string `json:"status"` // "approved" / "declined"
	Reference string `json:"reference"`
}

// OrderResult is the final persisted outcome of the order.
type OrderResult struct {
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	PaymentRef    string `json:"payment_ref,omitempty"`
	ReservationID string `json:"reservation_id"`
}

// Repository persists order results.
type Repository interface {
	SaveOrder(ctx context.Context, result *OrderResult) error
}
