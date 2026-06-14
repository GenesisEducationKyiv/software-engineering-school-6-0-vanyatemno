package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ghnotify/contract"
	"ghnotify/notifier/internal/mailer"
	"ghnotify/notifier/internal/templates"

	"github.com/segmentio/kafka-go"
)

// --- mocks ------------------------------------------------------------------

// ReaderMock yields a fixed slice of messages, then cancels the worker's context
// (so Run returns) and reports the cancellation as a fetch error. It records
// every CommitMessages call so tests can assert offsets advanced.
type ReaderMock struct {
	msgs    []kafka.Message
	idx     int
	commits int
	cancel  context.CancelFunc
}

func (r *ReaderMock) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if err := ctx.Err(); err != nil {
		return kafka.Message{}, err
	}
	if r.idx >= len(r.msgs) {
		if r.cancel != nil {
			r.cancel()
		}
		return kafka.Message{}, context.Canceled
	}
	m := r.msgs[r.idx]
	r.idx++
	return m, nil
}

func (r *ReaderMock) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	r.commits += len(msgs)
	return nil
}

// DLQProducerMock records every message written to the dead-letter topic.
type DLQProducerMock struct {
	writes []kafka.Message
}

func (d *DLQProducerMock) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	d.writes = append(d.writes, msgs...)
	return nil
}

// DeduperFake is an in-memory Deduper modelling Redis SET NX: the first Claim of
// a key wins, repeat claims are skipped, and Release frees the key again.
type DeduperFake struct {
	claimed  map[string]bool
	claimErr error
}

func NewDeduperFake() *DeduperFake { return &DeduperFake{claimed: map[string]bool{}} }

func (d *DeduperFake) Claim(_ context.Context, key string) (bool, error) {
	if d.claimErr != nil {
		return false, d.claimErr
	}
	if d.claimed[key] {
		return false, nil
	}
	d.claimed[key] = true
	return true, nil
}

func (d *DeduperFake) Release(_ context.Context, key string) error {
	delete(d.claimed, key)
	return nil
}

// TemplatesServiceMock records render calls and can inject a render error.
type TemplatesServiceMock struct {
	calls     int
	renderErr error
}

func (m *TemplatesServiceMock) RenderTemplate(_ contract.TemplateName, _ any) (*templates.RenderedTemplate, error) {
	m.calls++
	if m.renderErr != nil {
		return nil, m.renderErr
	}
	return &templates.RenderedTemplate{Subject: "subject", Body: "body"}, nil
}

// MailerMock records send attempts and successful sends. sendErr fails every
// send; failFor[recipient] fails that recipient the given number of times before
// succeeding (to model transient SMTP errors).
type MailerMock struct {
	attempts []string
	sent     []string
	sendErr  error
	failFor  map[string]int
}

func (m *MailerMock) Send(msg *mailer.Message) error {
	m.attempts = append(m.attempts, msg.To...)
	if m.sendErr != nil {
		return m.sendErr
	}
	for _, to := range msg.To {
		if m.failFor != nil && m.failFor[to] > 0 {
			m.failFor[to]--
			return errors.New("transient send failure for " + to)
		}
	}
	m.sent = append(m.sent, msg.To...)
	return nil
}

// --- harness ----------------------------------------------------------------

type testDeps struct {
	worker    *Worker
	reader    *ReaderMock
	dlq       *DLQProducerMock
	dedup     *DeduperFake
	templates *TemplatesServiceMock
	mailer    *MailerMock
	ctx       context.Context
	cancel    context.CancelFunc
}

func setupTest(msgs []kafka.Message, maxRetries int) *testDeps {
	retryBackoff = time.Millisecond // keep retries fast under test

	ctx, cancel := context.WithCancel(context.Background())
	reader := &ReaderMock{msgs: msgs, cancel: cancel}
	dlq := &DLQProducerMock{}
	ded := NewDeduperFake()
	tmpl := &TemplatesServiceMock{}
	ml := &MailerMock{}

	return &testDeps{
		worker:    New(reader, dlq, ded, tmpl, ml, maxRetries),
		reader:    reader,
		dlq:       dlq,
		dedup:     ded,
		templates: tmpl,
		mailer:    ml,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func mustMessage(t *testing.T, template contract.TemplateName, receivers []string, payload any) kafka.Message {
	t.Helper()
	pb, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	key := contract.IdempotencyKey(template, pb)
	body, err := json.Marshal(contract.Message{
		Template:       template,
		Receivers:      receivers,
		Payload:        pb,
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	return kafka.Message{Key: []byte(key), Value: body}
}

func count(ss []string, s string) int {
	n := 0
	for _, v := range ss {
		if v == s {
			n++
		}
	}
	return n
}

func confirmPayload() contract.ConfirmEmailPayload {
	return contract.ConfirmEmailPayload{Code: "code123", Link: "https://x/confirm/code123"}
}

func repoPayload() contract.RepositoryUpdateEmailPayload {
	return contract.RepositoryUpdateEmailPayload{Name: "repo", Owner: "owner", Version: "v1.2.3", UnsubscribeURL: "https://x/unsub"}
}

// --- tests ------------------------------------------------------------------

func TestRun_ConfirmMessage_SendsOnceAndCommits(t *testing.T) {
	msg := mustMessage(t, contract.Confirmation, []string{"a@example.com"}, confirmPayload())
	td := setupTest([]kafka.Message{msg}, 3)

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := len(td.mailer.sent); got != 1 {
		t.Fatalf("sent = %d, want 1", got)
	}
	if td.templates.calls != 1 {
		t.Fatalf("render calls = %d, want 1", td.templates.calls)
	}
	if len(td.dlq.writes) != 0 {
		t.Fatalf("dlq writes = %d, want 0", len(td.dlq.writes))
	}
	if td.reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", td.reader.commits)
	}
}

func TestRun_RepositoryUpdateMultiRecipient_SendsToEachAndCommits(t *testing.T) {
	recv := []string{"a@x.com", "b@x.com", "c@x.com"}
	msg := mustMessage(t, contract.RepositoryUpdated, recv, repoPayload())
	td := setupTest([]kafka.Message{msg}, 3)

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := len(td.mailer.sent); got != 3 {
		t.Fatalf("sent = %d, want 3", got)
	}
	if len(td.dlq.writes) != 0 {
		t.Fatalf("dlq writes = %d, want 0", len(td.dlq.writes))
	}
	if td.reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", td.reader.commits)
	}
}

// The core at-most-once guarantee: the same message delivered twice emails each
// recipient exactly once.
func TestRun_DuplicateDelivery_EmailsEachRecipientOnce(t *testing.T) {
	recv := []string{"a@x.com", "b@x.com"}
	msg := mustMessage(t, contract.RepositoryUpdated, recv, repoPayload())
	td := setupTest([]kafka.Message{msg, msg}, 3) // delivered twice

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := len(td.mailer.sent); got != 2 {
		t.Fatalf("sent = %d, want 2 (no duplicates)", got)
	}
	if c := count(td.mailer.sent, "a@x.com"); c != 1 {
		t.Fatalf("a@x.com sent %d times, want 1", c)
	}
	if c := count(td.mailer.sent, "b@x.com"); c != 1 {
		t.Fatalf("b@x.com sent %d times, want 1", c)
	}
	if len(td.dlq.writes) != 0 {
		t.Fatalf("dlq writes = %d, want 0", len(td.dlq.writes))
	}
	if td.reader.commits != 2 {
		t.Fatalf("commits = %d, want 2", td.reader.commits)
	}
}

// A partial failure dead-letters and, on redelivery, re-attempts only the failed
// recipient — the already-sent recipient is not emailed again.
func TestRun_PartialFailureThenRedelivery_NoDuplicateToSucceeded(t *testing.T) {
	recv := []string{"a@x.com", "b@x.com"}
	msg := mustMessage(t, contract.RepositoryUpdated, recv, repoPayload())
	td := setupTest([]kafka.Message{msg, msg}, 0)    // no in-process retry; rely on redelivery
	td.mailer.failFor = map[string]int{"b@x.com": 1} // b fails on first delivery only

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if c := count(td.mailer.sent, "a@x.com"); c != 1 {
		t.Fatalf("a@x.com sent %d times, want 1 (no duplicate)", c)
	}
	if c := count(td.mailer.sent, "b@x.com"); c != 1 {
		t.Fatalf("b@x.com sent %d times, want 1", c)
	}
	if len(td.dlq.writes) != 1 {
		t.Fatalf("dlq writes = %d, want 1 (first delivery dead-lettered)", len(td.dlq.writes))
	}
	if td.reader.commits != 2 {
		t.Fatalf("commits = %d, want 2", td.reader.commits)
	}
}

func TestRun_InvalidJSON_DeadLettersAndCommits(t *testing.T) {
	msg := kafka.Message{Key: []byte("k"), Value: []byte("not json")}
	td := setupTest([]kafka.Message{msg}, 3)

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(td.mailer.attempts) != 0 {
		t.Fatalf("mailer attempts = %d, want 0", len(td.mailer.attempts))
	}
	if td.templates.calls != 0 {
		t.Fatalf("render calls = %d, want 0", td.templates.calls)
	}
	if len(td.dlq.writes) != 1 {
		t.Fatalf("dlq writes = %d, want 1", len(td.dlq.writes))
	}
	if td.reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", td.reader.commits)
	}
}

func TestRun_UnknownTemplate_DeadLettersWithoutSending(t *testing.T) {
	body, err := json.Marshal(contract.Message{
		Template:       "bogus",
		Receivers:      []string{"a@x.com"},
		Payload:        json.RawMessage(`{}`),
		IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	td := setupTest([]kafka.Message{{Value: body}}, 3)

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(td.mailer.attempts) != 0 {
		t.Fatalf("mailer attempts = %d, want 0", len(td.mailer.attempts))
	}
	if len(td.dlq.writes) != 1 {
		t.Fatalf("dlq writes = %d, want 1", len(td.dlq.writes))
	}
	if td.reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", td.reader.commits)
	}
}

func TestRun_RenderError_DeadLettersWithoutSending(t *testing.T) {
	msg := mustMessage(t, contract.Confirmation, []string{"a@x.com"}, confirmPayload())
	td := setupTest([]kafka.Message{msg}, 3)
	td.templates.renderErr = errors.New("render boom")

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(td.mailer.attempts) != 0 {
		t.Fatalf("mailer attempts = %d, want 0", len(td.mailer.attempts))
	}
	if len(td.dlq.writes) != 1 {
		t.Fatalf("dlq writes = %d, want 1", len(td.dlq.writes))
	}
	if td.reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", td.reader.commits)
	}
}

func TestRun_MailerErrorEveryAttempt_RetriesThenDeadLetters(t *testing.T) {
	msg := mustMessage(t, contract.Confirmation, []string{"a@x.com"}, confirmPayload())
	td := setupTest([]kafka.Message{msg}, 2) // initial + 2 retries
	td.mailer.sendErr = errors.New("smtp down")

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(td.mailer.sent) != 0 {
		t.Fatalf("sent = %d, want 0", len(td.mailer.sent))
	}
	if got := len(td.mailer.attempts); got != 3 {
		t.Fatalf("send attempts = %d, want 3 (1 + 2 retries)", got)
	}
	if len(td.dlq.writes) != 1 {
		t.Fatalf("dlq writes = %d, want 1", len(td.dlq.writes))
	}
	if td.reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", td.reader.commits)
	}
}

func TestRun_DedupClaimError_DoesNotSend(t *testing.T) {
	msg := mustMessage(t, contract.Confirmation, []string{"a@x.com"}, confirmPayload())
	td := setupTest([]kafka.Message{msg}, 0)
	td.dedup.claimErr = errors.New("redis down")

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(td.mailer.attempts) != 0 {
		t.Fatalf("mailer attempts = %d, want 0 (must not send when dedup is unavailable)", len(td.mailer.attempts))
	}
	if len(td.dlq.writes) != 1 {
		t.Fatalf("dlq writes = %d, want 1", len(td.dlq.writes))
	}
	if td.reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", td.reader.commits)
	}
}

func TestRun_ContextCancelled_ReturnsWithoutProcessing(t *testing.T) {
	msg := mustMessage(t, contract.Confirmation, []string{"a@x.com"}, confirmPayload())
	td := setupTest([]kafka.Message{msg}, 3)
	td.cancel() // cancel before Run

	if err := td.worker.Run(td.ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(td.mailer.attempts) != 0 {
		t.Fatalf("mailer attempts = %d, want 0", len(td.mailer.attempts))
	}
	if td.reader.commits != 0 {
		t.Fatalf("commits = %d, want 0", td.reader.commits)
	}
}
