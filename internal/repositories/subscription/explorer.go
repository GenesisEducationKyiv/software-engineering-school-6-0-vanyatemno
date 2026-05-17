package subscription

import (
	"context"
	"errors"
	"se-school/internal/models"
	"se-school/internal/repositories"

	"github.com/jackc/pgx/v5"
)

func scanSubscription(row pgx.Row) (*models.Subscription, error) {
	var s models.Subscription
	if err := row.Scan(
		&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
		&s.RepositoryID, &s.SubscribeCodeID, &s.UnsubscribeCodeID,
		&s.Email, &s.IsConfirmed, &s.LastSeenTag,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*models.Subscription, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+subscriptionColumns+`
		 FROM subscriptions WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	sub, err := scanSubscription(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return sub, nil
}

func (r *Repository) GetUnupdated(ctx context.Context, repositoryID uint, currentTag string) ([]*models.Subscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+subscriptionColumns+`
		 FROM subscriptions
		 WHERE repository_id = $1 AND last_seen_tag != $2 AND deleted_at IS NULL`,
		repositoryID, currentTag,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// GetByEmail returns every active subscription for the email plus the joined
// Repository for each row (replaces the prior GORM Preload).
func (r *Repository) GetByEmail(ctx context.Context, email string) ([]*models.Subscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			s.id, s.created_at, s.updated_at, s.deleted_at,
			s.repository_id, s.subscribe_code_id, s.unsubscribe_code_id,
			s.email, s.is_confirmed, s.last_seen_tag,
			r.id, r.created_at, r.updated_at, r.deleted_at,
			r.owner, r.name, r.version
		 FROM subscriptions s
		 LEFT JOIN repositories r ON r.id = s.repository_id AND r.deleted_at IS NULL
		 WHERE s.email = $1 AND s.deleted_at IS NULL`,
		email,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.Subscription
	for rows.Next() {
		var s models.Subscription
		var repo models.Repository
		var repoID *uint
		if err := rows.Scan(
			&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
			&s.RepositoryID, &s.SubscribeCodeID, &s.UnsubscribeCodeID,
			&s.Email, &s.IsConfirmed, &s.LastSeenTag,
			&repoID, &repo.CreatedAt, &repo.UpdatedAt, &repo.DeletedAt,
			&repo.Owner, &repo.Name, &repo.Version,
		); err != nil {
			return nil, err
		}
		if repoID != nil {
			repo.ID = *repoID
			s.Repository = &repo
		}
		subs = append(subs, &s)
	}
	return subs, rows.Err()
}

func (r *Repository) GetByCode(ctx context.Context, codeID uint, codeType models.CodeType) (*models.Subscription, error) {
	var column string
	switch codeType {
	case models.CodeTypeUnsubscribe:
		column = "unsubscribe_code_id"
	case models.CodeTypeConfirm:
		column = "subscribe_code_id"
	default:
		return nil, errors.New("invalid codeType")
	}

	row := r.db.QueryRow(ctx,
		`SELECT `+subscriptionColumns+`
		 FROM subscriptions WHERE `+column+` = $1 AND deleted_at IS NULL`,
		codeID,
	)
	sub, err := scanSubscription(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return sub, nil
}
