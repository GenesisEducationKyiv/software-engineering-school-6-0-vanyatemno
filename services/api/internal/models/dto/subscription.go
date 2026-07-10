package dto

type CreateSubscriptionRequest struct {
	Email string `json:"email" form:"email" binding:"required,email"`
	Repo  string `json:"repo" form:"repo" binding:"required"`
}

// Returned 202 Accepted: the subscription is not durable yet — the caller polls
// GET /subscribe/status/{sagaId} until the saga completes.
type CreateSubscriptionResponse struct {
	SagaID string `json:"sagaId"`
	State  string `json:"state"`
}

type SubscriptionStatusRequest struct {
	SagaID string `uri:"sagaId" binding:"required"`
}

type SubscriptionStatusResponse struct {
	SagaID string `json:"sagaId"`
	State  string `json:"state"`
}

type ConfirmSubscriptionRequest struct {
	Token string `uri:"token" binding:"required"`
}

type UnsubscribeRequest struct {
	Token string `uri:"token" binding:"required"`
}

type GetSubscriptionsRequest struct {
	Email string `form:"email" binding:"required,email"`
}

type SubscriptionResponse struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag,omitempty"`
}
