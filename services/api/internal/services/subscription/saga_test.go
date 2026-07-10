package subscription

import (
	"context"
	"testing"

	"ghnotify/contract"

	"se-school/internal/models"
)

func awaitingSaga(id string) *models.SagaInstance {
	return &models.SagaInstance{
		ID:             id,
		Type:           models.SagaTypeSubscribeConfirm,
		State:          models.SagaStateAwaitingNotification,
		SubscriptionID: 42,
		Email:          "user@example.com",
	}
}

func TestHandleReply_Dispatched_MarksCompleted(t *testing.T) {
	td := setupTest()
	td.sagas.GetByIDResult = awaitingSaga("saga-1")

	err := td.svc.HandleReply(context.Background(), contract.Reply{
		SagaID: "saga-1",
		Status: contract.ReplyDispatched,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(td.sagas.UpdateStateCalls) != 1 {
		t.Fatalf("expected 1 state update, got %d", len(td.sagas.UpdateStateCalls))
	}
	if td.sagas.UpdateStateCalls[0].State != models.SagaStateCompleted {
		t.Fatalf("expected COMPLETED, got %q", td.sagas.UpdateStateCalls[0].State)
	}
	if td.subs.DeleteCount != 0 {
		t.Fatalf("expected no compensation on success, got %d deletes", td.subs.DeleteCount)
	}
}

func TestHandleReply_Failed_CompensatesAndDeletesSubscription(t *testing.T) {
	td := setupTest()
	td.sagas.GetByIDResult = awaitingSaga("saga-2")
	td.subs.GetByIDResult = &models.Subscription{ID: 42, Email: "user@example.com"}

	err := td.svc.HandleReply(context.Background(), contract.Reply{
		SagaID: "saga-2",
		Status: contract.ReplyFailed,
		Reason: "smtp down",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// COMPENSATING then COMPENSATED.
	if len(td.sagas.UpdateStateCalls) != 2 {
		t.Fatalf("expected 2 state updates, got %d", len(td.sagas.UpdateStateCalls))
	}
	if td.sagas.UpdateStateCalls[0].State != models.SagaStateCompensating {
		t.Fatalf("expected first update COMPENSATING, got %q", td.sagas.UpdateStateCalls[0].State)
	}
	if td.sagas.UpdateStateCalls[1].State != models.SagaStateCompensated {
		t.Fatalf("expected final update COMPENSATED, got %q", td.sagas.UpdateStateCalls[1].State)
	}
	if td.subs.DeleteCount != 1 {
		t.Fatalf("expected the subscription to be deleted once, got %d", td.subs.DeleteCount)
	}
}

func TestHandleReply_TerminalSaga_IsIgnored(t *testing.T) {
	td := setupTest()
	completed := awaitingSaga("saga-3")
	completed.State = models.SagaStateCompleted
	td.sagas.GetByIDResult = completed

	// A duplicate/late reply for an already-completed saga must be a no-op.
	err := td.svc.HandleReply(context.Background(), contract.Reply{
		SagaID: "saga-3",
		Status: contract.ReplyFailed,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(td.sagas.UpdateStateCalls) != 0 {
		t.Fatalf("expected no state changes for terminal saga, got %d", len(td.sagas.UpdateStateCalls))
	}
	if td.subs.DeleteCount != 0 {
		t.Fatalf("expected no compensation for terminal saga, got %d deletes", td.subs.DeleteCount)
	}
}

func TestHandleReply_UnknownSaga_IsIgnored(t *testing.T) {
	td := setupTest()
	td.sagas.GetByIDErr = models.ErrNotFound

	err := td.svc.HandleReply(context.Background(), contract.Reply{
		SagaID: "missing",
		Status: contract.ReplyDispatched,
	})
	if err != nil {
		t.Fatalf("expected nil error for unknown saga, got %v", err)
	}
	if len(td.sagas.UpdateStateCalls) != 0 {
		t.Fatalf("expected no state changes, got %d", len(td.sagas.UpdateStateCalls))
	}
}

func TestSweep_CompensatesStuckSagas(t *testing.T) {
	td := setupTest()
	td.sagas.GetStuckResult = []*models.SagaInstance{awaitingSaga("stuck-1")}
	td.subs.GetByIDResult = &models.Subscription{ID: 42, Email: "user@example.com"}

	if err := td.svc.Sweep(context.Background(), 100); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if td.subs.DeleteCount != 1 {
		t.Fatalf("expected stuck saga's subscription deleted once, got %d", td.subs.DeleteCount)
	}
	calls := td.sagas.UpdateStateCalls
	if len(calls) == 0 || calls[len(calls)-1].State != models.SagaStateCompensated {
		t.Fatalf("expected stuck saga to end COMPENSATED, calls=%v", calls)
	}
}

func TestStatus_ReturnsSaga(t *testing.T) {
	td := setupTest()
	td.sagas.GetByIDResult = awaitingSaga("saga-x")

	saga, err := td.svc.Status(context.Background(), "saga-x")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if saga == nil || saga.ID != "saga-x" {
		t.Fatalf("expected saga saga-x, got %v", saga)
	}
}
