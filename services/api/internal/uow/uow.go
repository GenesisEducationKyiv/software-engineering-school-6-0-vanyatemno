// Package uow runs one database transaction and hands the orchestrator the
// tx-bound repositories, so all of T1 (subscription, codes, saga, outbox)
// commits or rolls back atomically.
package uow

import (
	"context"

	"se-school/internal/infrastructure/db"
	codeRepo "se-school/internal/repositories/code"
	outboxRepo "se-school/internal/repositories/outbox"
	sagaRepo "se-school/internal/repositories/saga"
	subRepo "se-school/internal/repositories/subscription"
	subscriptionSvc "se-school/internal/services/subscription"

	"github.com/jackc/pgx/v5"
)

type UnitOfWork struct {
	transactor *db.Transactor
	subs       *subRepo.Repository
	codes      *codeRepo.Repository
	sagas      *sagaRepo.Repository
	outbox     *outboxRepo.Repository
}

func New(
	transactor *db.Transactor,
	subs *subRepo.Repository,
	codes *codeRepo.Repository,
	sagas *sagaRepo.Repository,
	outbox *outboxRepo.Repository,
) *UnitOfWork {
	return &UnitOfWork{transactor: transactor, subs: subs, codes: codes, sagas: sagas, outbox: outbox}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(ctx context.Context, r *subscriptionSvc.TxRepos) error) error {
	return u.transactor.RunInTx(ctx, func(tx pgx.Tx) error {
		return fn(ctx, &subscriptionSvc.TxRepos{
			Subscriptions: u.subs.WithTx(tx),
			Codes:         u.codes.WithTx(tx),
			Sagas:         u.sagas.WithTx(tx),
			Outbox:        u.outbox.WithTx(tx),
		})
	})
}
