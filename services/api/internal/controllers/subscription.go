package controllers

import (
	"net/http"
	"se-school/internal/infrastructure/logging"
	"se-school/internal/models"
	"se-school/internal/models/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SubscriptionController struct {
	subscriptionService SubscriptionsService
}

func NewSubscriptionController(
	subscriptionService SubscriptionsService,
) *SubscriptionController {
	return &SubscriptionController{
		subscriptionService: subscriptionService,
	}
}

// Subscribe handles POST /api/subscribe.
// Accepts form-data or JSON with "email" and "repo" fields.
//
// Subscribing starts an orchestrated saga (create subscription + dispatch the
// confirmation email across the notifications service). The endpoint returns
// 202 Accepted with a saga id; the caller polls GET /subscribe/status/{sagaId}
// to learn whether the confirmation email was dispatched (COMPLETED) or the
// subscription was rolled back (COMPENSATED).
//
//	@Summary		Subscribe to release notifications
//	@Description	Starts the confirmation saga for an email + GitHub repository. Returns 202 with a saga id to poll for completion.
//	@Tags			subscription
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			email	formData	string	true	"Email address to subscribe"
//	@Param			repo	formData	string	true	"GitHub repository in owner/repo format (e.g., golang/go)"
//	@Success		202		{object}	dto.CreateSubscriptionResponse	"Subscription accepted; poll status by saga id"
//	@Failure		400		{object}	object{error=string}	"Invalid input (e.g., invalid repo format)"
//	@Failure		404		{object}	object{error=string}	"Repository not found on GitHub"
//	@Failure		409		{object}	object{error=string}	"Email already subscribed to this repository"
//	@Security		ApiKeyAuth
//	@Router			/subscribe [post]
func (sc *SubscriptionController) Subscribe(c *gin.Context) {
	var req dto.CreateSubscriptionRequest

	err := c.ShouldBind(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	sagaID, err := sc.subscriptionService.Create(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusAccepted, dto.CreateSubscriptionResponse{
		SagaID: sagaID,
		State:  models.SagaStateAwaitingNotification,
	})
}

// Status handles GET /api/subscribe/status/:sagaId.
//
//	@Summary		Get subscribe saga status
//	@Description	Returns the current state of a subscribe saga started via POST /subscribe.
//	@Tags			subscription
//	@Produce		json
//	@Param			sagaId	path		string	true	"Saga id returned by POST /subscribe"
//	@Success		200		{object}	dto.SubscriptionStatusResponse	"Current saga state"
//	@Failure		400		{object}	object{error=string}	"Invalid saga id"
//	@Failure		404		{object}	object{error=string}	"Saga not found"
//	@Security		ApiKeyAuth
//	@Router			/subscribe/status/{sagaId} [get]
func (sc *SubscriptionController) Status(c *gin.Context) {
	var req dto.SubscriptionStatusRequest
	if err := c.ShouldBindUri(&req); err != nil {
		logging.FromContext(c.Request.Context()).Warn("invalid status request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid saga id"})
		return
	}

	saga, err := sc.subscriptionService.Status(c.Request.Context(), req.SagaID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.SubscriptionStatusResponse{
		SagaID: saga.ID,
		State:  saga.State,
	})
}

// Confirm handles GET /api/confirm/:token.
//
//	@Summary		Confirm email subscription
//	@Description	Confirms a subscription using the token sent in the confirmation email.
//	@Tags			subscription
//	@Produce		json
//	@Param			token	path		string	true	"Confirmation token"
//	@Success		200		{object}	object{message=string}	"Subscription confirmed successfully"
//	@Failure		400		{object}	object{error=string}	"Invalid token"
//	@Failure		404		{object}	object{error=string}	"Token not found"
//	@Security		ApiKeyAuth
//	@Router			/confirm/{token} [get]
func (sc *SubscriptionController) Confirm(c *gin.Context) {
	var req dto.ConfirmSubscriptionRequest
	if err := c.ShouldBindUri(&req); err != nil {
		logging.FromContext(c.Request.Context()).Warn("invalid confirm request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	err := sc.subscriptionService.Confirm(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription confirmed successfully"})
}

// Unsubscribe handles GET /api/unsubscribe/:token.
//
//	@Summary		Unsubscribe from release notifications
//	@Description	Unsubscribes an email from release notifications using the token sent in emails.
//	@Tags			subscription
//	@Produce		json
//	@Param			token	path		string	true	"Unsubscribe token"
//	@Success		200		{object}	object{message=string}	"Unsubscribed successfully"
//	@Failure		400		{object}	object{error=string}	"Invalid token"
//	@Failure		404		{object}	object{error=string}	"Token not found"
//	@Security		ApiKeyAuth
//	@Router			/unsubscribe/{token} [get]
func (sc *SubscriptionController) Unsubscribe(c *gin.Context) {
	var req dto.UnsubscribeRequest
	if err := c.ShouldBindUri(&req); err != nil {
		logging.FromContext(c.Request.Context()).Warn("invalid unsubscribe request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	err := sc.subscriptionService.Unsubscribe(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

// GetSubscriptions handles GET /api/subscriptions?email=...
//
//	@Summary		Get subscriptions for an email
//	@Description	Returns all active subscriptions for the given email address.
//	@Tags			subscription
//	@Produce		json
//	@Param			email	query		string	true	"Email address to look up subscriptions for"
//	@Success		200		{array}		dto.SubscriptionResponse	"Successful operation - list of subscriptions returned"
//	@Failure		400		{object}	object{error=string}		"Invalid email"
//	@Security		ApiKeyAuth
//	@Router			/subscriptions [get]
func (sc *SubscriptionController) GetSubscriptions(c *gin.Context) {
	var req dto.GetSubscriptionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logging.FromContext(c.Request.Context()).Warn("invalid get subscriptions request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email"})
		return
	}

	subscriptions, err := sc.subscriptionService.ListByEmail(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}
