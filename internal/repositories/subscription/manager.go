package subscription

import (
	"context"
	"se-school/internal/models"
	"se-school/internal/repositories"
)

func (r *Repository) Create(ctx context.Context, subscription *models.Subscription) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO subscriptions
			(repository_id, subscribe_code_id, unsubscribe_code_id,
			 email, is_confirmed, last_seen_tag)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		subscription.RepositoryID,
		subscription.SubscribeCodeID,
		subscription.UnsubscribeCodeID,
		subscription.Email,
		subscription.IsConfirmed,
		subscription.LastSeenTag,
	)

	return row.Scan(&subscription.ID, &subscription.CreatedAt, &subscription.UpdatedAt)
}

func (r *Repository) UpdateLastSeenTag(ctx context.Context, id uint, tag string) error {
	res, err := r.db.Exec(ctx,
		`UPDATE subscriptions
		 SET last_seen_tag = $1, updated_at = NOW()
		 WHERE id = $2 AND deleted_at IS NULL`,
		tag, id,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

// Save persists the mutable fields of an already-loaded subscription. Used by
// the service layer after toggling IsConfirmed or other in-memory updates.
func (r *Repository) Save(ctx context.Context, subscription *models.Subscription) error {
	res, err := r.db.Exec(ctx,
		`UPDATE subscriptions
		 SET repository_id = $1,
		     subscribe_code_id = $2,
		     unsubscribe_code_id = $3,
		     email = $4,
		     is_confirmed = $5,
		     last_seen_tag = $6,
		     updated_at = NOW()
		 WHERE id = $7 AND deleted_at IS NULL`,
		subscription.RepositoryID,
		subscription.SubscribeCodeID,
		subscription.UnsubscribeCodeID,
		subscription.Email,
		subscription.IsConfirmed,
		subscription.LastSeenTag,
		subscription.ID,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

// Delete soft-deletes the subscription and the two related codes in a single
// transaction. This replaces the prior AfterDelete hook on the Subscription
// GORM model — moving the cascade into an explicit Tx makes it atomic, which
// the hook was not.
func (r *Repository) Delete(ctx context.Context, subscription *models.Subscription) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE subscriptions SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		subscription.ID,
	); err != nil {
		return err
	}

	if subscription.SubscribeCodeID != 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE codes SET deleted_at = NOW(), updated_at = NOW()
			 WHERE id = $1 AND deleted_at IS NULL`,
			subscription.SubscribeCodeID,
		); err != nil {
			return err
		}
	}
	if subscription.UnsubscribeCodeID != 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE codes SET deleted_at = NOW(), updated_at = NOW()
			 WHERE id = $1 AND deleted_at IS NULL`,
			subscription.UnsubscribeCodeID,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
