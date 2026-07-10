// Package uow provides the concrete UnitOfWork the subscription orchestrator
// depends on. It runs one database transaction (via db.Transactor) and hands
// the orchestrator the transaction-bound repositories, so the subscription,
// codes, saga instance and outbox command of T1 all commit (or roll back)
// atomically.
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

// UnitOfWork binds the concrete repositories to a shared transaction. It
// satisfies subscriptionSvc.UnitOfWork.
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

// Do runs fn inside a transaction, committing on success and rolling back on
// error. The repositories in TxRepos are re-bound to the transaction via their
// WithTx methods.
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
