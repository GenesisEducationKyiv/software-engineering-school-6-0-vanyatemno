package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghnotify/contract"

	"se-school/internal/models"
	"se-school/internal/models/dto"
	"se-school/internal/notifications"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Create starts the orchestrated Subscribe→confirmation-email saga and returns
// the saga id (for status polling).
//
// In a single database transaction (T1) it persists the subscription (pending),
// its two codes, the saga instance (AWAITING_NOTIFICATION) and an outbox row
// carrying the confirmation-email command. Because the command is written in the
// same transaction as the state it depends on, "subscription created" and
// "command to be sent" are atomic — closing the previous dual-write. The relay
// then publishes the command; the notifier's reply drives the saga to COMPLETED
// or triggers compensation (see saga.go).
//
// The subscription is created unconfirmed: the saga only guarantees the
// confirmation email is dispatched — the user still confirms via the emailed
// link (see confirm.go).
func (s *Service) Create(ctx context.Context, req *dto.CreateSubscriptionRequest) (string, error) {
	parsed, err := parseRepoFields(req.Repo)
	if err != nil {
		zap.L().Error("failed to parse repo fields", zap.Error(err))
		return "", err
	}

	// Resolve (and, for a brand-new repo, create) the repository BEFORE opening
	// the transaction: it may call the GitHub API, and the repositories row is
	// shared state the saga deliberately does not compensate.
	repo, err := s.getOrCreateRepository(ctx, parsed)
	if err != nil {
		return "", err
	}

	// Build the codes (pure/in-memory) and the confirmation-email command up
	// front, so the transaction contains only fast local writes.
	unsubCode, err := s.codeFactory.New(models.CodeTypeUnsubscribe)
	if err != nil {
		return "", err
	}
	subCode, err := s.codeFactory.New(models.CodeTypeConfirm)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(notifications.BuildConfirmEmailPayload(s.frontendURL, subCode.Code))
	if err != nil {
		return "", err
	}
	idemKey := contract.IdempotencyKey(contract.Confirmation, payload)
	sagaID := uuid.NewString()

	command, err := json.Marshal(contract.Message{
		Template:       contract.Confirmation,
		Receivers:      []string{req.Email},
		Payload:        payload,
		IdempotencyKey: idemKey,
		SagaID:         sagaID,
	})
	if err != nil {
		return "", err
	}

	err = s.uow.Do(ctx, func(ctx context.Context, r *TxRepos) error {
		if err := r.Codes.Create(ctx, unsubCode); err != nil {
			return err
		}
		if err := r.Codes.Create(ctx, subCode); err != nil {
			return err
		}

		sub := &models.Subscription{
			RepositoryID:      repo.ID,
			SubscribeCodeID:   subCode.ID,
			UnsubscribeCodeID: unsubCode.ID,
			SubscribeCode:     subCode,
			UnsubscribeCode:   unsubCode,
			Email:             req.Email,
			LastSeenTag:       repo.Version,
		}
		if err := r.Subscriptions.Create(ctx, sub); err != nil {
			return err
		}

		if err := r.Sagas.Create(ctx, &models.SagaInstance{
			ID:             sagaID,
			Type:           models.SagaTypeSubscribeConfirm,
			State:          models.SagaStateAwaitingNotification,
			SubscriptionID: sub.ID,
			Email:          req.Email,
			IdempotencyKey: idemKey,
			DeadlineAt:     time.Now().Add(s.sagaDeadline),
		}); err != nil {
			return err
		}

		return r.Outbox.Create(ctx, &models.OutboxMessage{
			ID:       uuid.NewString(),
			SagaID:   sagaID,
			Topic:    contract.Topic,
			KafkaKey: idemKey,
			Payload:  command,
		})
	})
	if err != nil {
		zap.L().Error("failed to start subscription saga", zap.Error(err))
		return "", err
	}

	zap.L().Info("subscription saga started",
		zap.String("saga_id", sagaID),
		zap.String("email", req.Email),
		zap.String("repo", req.Repo))
	return sagaID, nil
}

func parseRepoFields(repo string) (*parsedRepoValue, error) {
	values := strings.Split(repo, "/")
	if len(values) != 2 {
		return nil, fmt.Errorf("failed to parse repo fields: %s", repo)
	}

	return &parsedRepoValue{
		Owner:          values[0],
		RepositoryName: values[1],
	}, nil
}

func (s *Service) getOrCreateRepository(ctx context.Context, values *parsedRepoValue) (*models.Repository, error) {
	repository, err := s.repositoriesRepository.Find(ctx, &models.Repository{
		Owner: values.Owner,
		Name:  values.RepositoryName,
	})
	if err != nil {
		if !errors.Is(err, models.ErrNotFound) {
			return nil, err
		}
		repository, err = s.createRepository(ctx, values)
		if err != nil {
			return nil, err
		}
	}

	return repository, nil
}

func (s *Service) createRepository(ctx context.Context, values *parsedRepoValue) (*models.Repository, error) {
	currentVersion, err := s.githubIntegration.GetRepositoryVersion(ctx, values.Owner, values.RepositoryName)
	if err != nil {
		return nil, err
	}
	repo := &models.Repository{
		Owner:   values.Owner,
		Name:    values.RepositoryName,
		Version: currentVersion,
	}
	err = s.repositoriesRepository.Create(ctx, repo)
	if err != nil {
		return nil, err
	}

	return repo, nil
}
