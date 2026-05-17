package dto

// CreateSubscriptionRequest represents the request body for POST /subscribe.
type CreateSubscriptionRequest struct {
	Email string `json:"email" form:"email" binding:"required,email"`
	Repo  string `json:"repo" form:"repo" binding:"required"`
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
