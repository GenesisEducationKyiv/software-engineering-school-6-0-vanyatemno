package dto

// CreateSubscriptionRequest represents the request body for POST /subscribe.
type CreateSubscriptionRequest struct {
	Email string `json:"email" form:"email" binding:"required,email"`
	Repo  string `json:"repo" form:"repo" binding:"required"`
}

// CreateSubscriptionResponse is returned (202 Accepted) when a subscribe request
// starts the confirmation saga. The subscription is not durable yet — the caller
// polls GET /subscribe/status/{sagaId} until the saga completes.
type CreateSubscriptionResponse struct {
	SagaID string `json:"sagaId"`
	State  string `json:"state"`
}

// SubscriptionStatusRequest represents the path parameter for
// GET /subscribe/status/{sagaId}.
type SubscriptionStatusRequest struct {
	SagaID string `uri:"sagaId" binding:"required"`
}

// SubscriptionStatusResponse reports the current saga state for a subscribe
// request (AWAITING_NOTIFICATION|COMPLETED|COMPENSATING|COMPENSATED|FAILED).
type SubscriptionStatusResponse struct {
	SagaID string `json:"sagaId"`
	State  string `json:"state"`
}

// ConfirmSubscriptionRequest represents the path parameter for GET /confirm/{token}.
type ConfirmSubscriptionRequest struct {
	Token string `uri:"token" binding:"required"`
}

// UnsubscribeRequest represents the path parameter for GET /unsubscribe/{token}.
type UnsubscribeRequest struct {
	Token string `uri:"token" binding:"required"`
}

// GetSubscriptionsRequest represents the query parameter for GET /subscriptions.
type GetSubscriptionsRequest struct {
	Email string `form:"email" binding:"required,email"`
}

// SubscriptionResponse represents the Subscription definition returned by GET /subscriptions.
type SubscriptionResponse struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag,omitempty"`
}
