package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ghnotify/contract"
	"ghnotify/notifier/internal/mailer"
	"ghnotify/notifier/internal/repositories/delivery"
	"ghnotify/notifier/internal/templates"
)

// --- fakes ------------------------------------------------------------------

type deliveryStoreFake struct {
	byKey    map[string]*delivery.Delivery
	nextID   int64
	claimErr error
}

func newDeliveryStoreFake() *deliveryStoreFake {
	return &deliveryStoreFake{byKey: map[string]*delivery.Delivery{}}
}

func dkey(idem, recipient string) string { return idem + "|" + recipient }

func (s *deliveryStoreFake) Claim(_ context.Context, d *delivery.Delivery) (delivery.State, error) {
	if s.claimErr != nil {
		return "", s.claimErr
	}
	k := dkey(d.IdempotencyKey, d.Recipient)
	if existing, ok := s.byKey[k]; ok {
		d.ID = existing.ID
		d.State = existing.State
		return existing.State, nil
	}
	s.nextID++
	rec := &delivery.Delivery{ID: s.nextID, IdempotencyKey: d.IdempotencyKey, Recipient: d.Recipient, State: delivery.StateSending}
	s.byKey[k] = rec
	d.ID = rec.ID
	d.State = delivery.StateSending
	return delivery.StateSending, nil
}

func (s *deliveryStoreFake) SetState(_ context.Context, id int64, state delivery.State, _ string) error {
	for _, r := range s.byKey {
		if r.ID == id {
			r.State = state
		}
	}
	return nil
}

func (s *deliveryStoreFake) seedSent(idem, recipient string) {
	s.nextID++
	s.byKey[dkey(idem, recipient)] = &delivery.Delivery{ID: s.nextID, IdempotencyKey: idem, Recipient: recipient, State: delivery.StateSent}
}

func (s *deliveryStoreFake) stateFor(idem, recipient string) delivery.State {
	if r, ok := s.byKey[dkey(idem, recipient)]; ok {
		return r.State
	}
	return ""
}

type deduperFake struct {
	claimed  map[string]bool
	claimErr error
	released int
}

func newDeduperFake() *deduperFake { return &deduperFake{claimed: map[string]bool{}} }

func (d *deduperFake) Claim(_ context.Context, key string) (bool, error) {
	if d.claimErr != nil {
		return false, d.claimErr
	}
	if d.claimed[key] {
		return false, nil
	}
	d.claimed[key] = true
	return true, nil
}

func (d *deduperFake) Release(_ context.Context, key string) error {
	d.released++
	delete(d.claimed, key)
	return nil
}

type templatesFake struct {
	calls     int
	renderErr error
}

func (t *templatesFake) RenderTemplate(_ contract.TemplateName, _ any) (*templates.RenderedTemplate, error) {
	t.calls++
	if t.renderErr != nil {
		return nil, t.renderErr
	}
	return &templates.RenderedTemplate{Subject: "s", Body: "b"}, nil
}

type mailerFake struct {
	sent    int
	failFor int // fail this many times before succeeding
	err     error
}

func (m *mailerFake) Send(_ *mailer.Message) error {
	if m.err != nil {
		return m.err
	}
	if m.failFor > 0 {
		m.failFor--
		return errors.New("transient send failure")
	}
	m.sent++
	return nil
}

// --- harness ----------------------------------------------------------------

func newTestDispatcher(maxRetries int) (*Dispatcher, *deliveryStoreFake, *deduperFake, *templatesFake, *mailerFake) {
	retryBackoff = time.Millisecond // keep retries fast under test
	del := newDeliveryStoreFake()
	ded := newDeduperFake()
	tmpl := &templatesFake{}
	ml := &mailerFake{}
	return New(del, ded, tmpl, ml, maxRetries), del, ded, tmpl, ml
}

func repoCmd(t *testing.T) Command {
	t.Helper()
	pb, err := json.Marshal(contract.RepositoryUpdateEmailPayload{Name: "r", Owner: "o", Version: "v1", UnsubscribeURL: "u"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return Command{Template: contract.RepositoryUpdated, Recipient: "a@x.com", Payload: pb, IdempotencyKey: "key1"}
}

// --- tests ------------------------------------------------------------------

func TestSend_Success_MarksSent(t *testing.T) {
	d, del, _, tmpl, ml := newTestDispatcher(3)

	if err := d.Send(context.Background(), repoCmd(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ml.sent != 1 {
		t.Fatalf("sent = %d, want 1", ml.sent)
	}
	if tmpl.calls != 1 {
		t.Fatalf("render calls = %d, want 1", tmpl.calls)
	}
	if st := del.stateFor("key1", "a@x.com"); st != delivery.StateSent {
		t.Fatalf("delivery state = %q, want SENT", st)
	}
}

func TestSend_AlreadySent_SkipsSend(t *testing.T) {
	d, del, _, tmpl, ml := newTestDispatcher(3)
	del.seedSent("key1", "a@x.com")

	if err := d.Send(context.Background(), repoCmd(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ml.sent != 0 || tmpl.calls != 0 {
		t.Fatalf("expected no render/send for an already-sent delivery; render=%d send=%d", tmpl.calls, ml.sent)
	}
}

func TestSend_RenderError_TerminalAndMarksFailed(t *testing.T) {
	d, del, _, tmpl, ml := newTestDispatcher(3)
	tmpl.renderErr = errors.New("render boom")

	err := d.Send(context.Background(), repoCmd(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if IsRetryable(err) {
		t.Fatal("render error must be terminal (non-retryable)")
	}
	if tmpl.calls != 1 {
		t.Fatalf("render calls = %d, want 1 (no retry on terminal error)", tmpl.calls)
	}
	if ml.sent != 0 {
		t.Fatalf("sent = %d, want 0", ml.sent)
	}
	if st := del.stateFor("key1", "a@x.com"); st != delivery.StateFailed {
		t.Fatalf("delivery state = %q, want FAILED", st)
	}
}

func TestSend_MailErrorEveryAttempt_RetriesThenFails(t *testing.T) {
	d, del, ded, _, ml := newTestDispatcher(2) // 1 + 2 retries
	ml.err = errors.New("smtp down")

	err := d.Send(context.Background(), repoCmd(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ml.sent != 0 {
		t.Fatalf("sent = %d, want 0", ml.sent)
	}
	// Each failed attempt releases its dedup claim so the next attempt re-sends.
	if ded.released != 3 {
		t.Fatalf("dedup releases = %d, want 3 (1 + 2 retries)", ded.released)
	}
	if st := del.stateFor("key1", "a@x.com"); st != delivery.StateFailed {
		t.Fatalf("delivery state = %q, want FAILED", st)
	}
}

func TestSend_TransientThenSuccess(t *testing.T) {
	d, del, _, _, ml := newTestDispatcher(3)
	ml.failFor = 2 // fail twice, succeed on the third attempt

	if err := d.Send(context.Background(), repoCmd(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ml.sent != 1 {
		t.Fatalf("sent = %d, want 1", ml.sent)
	}
	if st := del.stateFor("key1", "a@x.com"); st != delivery.StateSent {
		t.Fatalf("delivery state = %q, want SENT", st)
	}
}

func TestSend_DedupAlreadyClaimed_TreatedAsSent(t *testing.T) {
	d, del, ded, _, ml := newTestDispatcher(3)
	ded.claimed[DedupKey("key1", "a@x.com")] = true // Redis marker already present

	if err := d.Send(context.Background(), repoCmd(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ml.sent != 0 {
		t.Fatalf("sent = %d, want 0 (dedup skip)", ml.sent)
	}
	if st := del.stateFor("key1", "a@x.com"); st != delivery.StateSent {
		t.Fatalf("delivery state = %q, want SENT", st)
	}
}

func TestSend_DeliveryClaimError_Retryable(t *testing.T) {
	d, del, _, _, ml := newTestDispatcher(0)
	del.claimErr = errors.New("db down")

	err := d.Send(context.Background(), repoCmd(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsRetryable(err) {
		t.Fatal("delivery-store error should be retryable")
	}
	if ml.sent != 0 {
		t.Fatalf("sent = %d, want 0", ml.sent)
	}
}
