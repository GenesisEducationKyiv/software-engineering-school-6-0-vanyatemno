package controllers

import (
	"net/http"
	"se-school/internal/models/dto"
	"se-school/internal/services/subscription"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SubscriptionController struct {
	subscriptionService subscription.SubscriptionsService
}

// NewSubscriptionController creates a new SubscriptionController backed by the given service.
func NewSubscriptionController(
	subscriptionService subscription.SubscriptionsService,
) *SubscriptionController {
	return &SubscriptionController{
		subscriptionService: subscriptionService,
	}
}

// Subscribe handles POST /api/subscribe.
// Accepts form-data or JSON with "email" and "repo" fields.
//
//	@Summary		Subscribe to release notifications
//	@Description	Subscribe an email to receive notifications about new releases of a GitHub repository. The repository is validated via GitHub API.
//	@Tags			subscription
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			email	formData	string	true	"Email address to subscribe"
//	@Param			repo	formData	string	true	"GitHub repository in owner/repo format (e.g., golang/go)"
//	@Success		200		{object}	object{message=string}	"Subscription successful. Confirmation email sent."
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

	err = sc.subscriptionService.Create(c.Request.Context(), &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription successful. Confirmation email sent."})
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
		zap.L().Warn("invalid confirm request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	err := sc.subscriptionService.Confirm(&req)
	if err != nil {
		handleServiceError(c, err)
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
		zap.L().Warn("invalid unsubscribe request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	err := sc.subscriptionService.Unsubscribe(&req)
	if err != nil {
		handleServiceError(c, err)
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
		zap.L().Warn("invalid get subscriptions request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email"})
		return
	}

	subscriptions, err := sc.subscriptionService.ListByEmail(&req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}
