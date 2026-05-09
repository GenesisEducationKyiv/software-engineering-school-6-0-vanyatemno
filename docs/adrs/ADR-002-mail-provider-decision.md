# ADR-002 Mail provider decision
* Status: accepted
* Date: 2026-05-09
* Author: Ivan Markhaichuk

## Context
We have to choose the mail provider to:
* Send subscribe/unsubscribe confirmation emails
* Send GitHub repository release update notifications

---

## Considered technologies
1. Local SMTP sender (`go-gomail`)
   1. Pros: simple implementation, no external service dependency, zero cost
   2. Cons: no delivery tracking, no automatic retry on failure
2. External providers (SendGrid, Mailgun, AWS SES):
   1. Pros: built-in deliverability, bounce handling, analytics
   2. Cons: vendor dependency, API key management, cost at scale

---

## Chosen provider: `go-gomail` (Local SMTP)

## Consequences
### Positive
* No third-party account or API key required
* Simple, auditable implementation
### Negative
* No retry queue for failed deliveries
* Deliverability depends on the configured SMTP server
