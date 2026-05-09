# ADR-003 GitHub integration decision
* Status: accepted
* Date: 2026-05-09
* Author: Ivan Markhaichuk

## Context
We have to integrate with GitHub API to:
* Validate that a repository exists before creating a subscription
* Poll for the latest release tag of each tracked repository

---

## Considered technologies
1. Raw HTTP API (`net/http`)
   1. Pros: no external dependency, full control over requests
   2. Cons: manual JSON parsing, manual rate limit handling, more boilerplate
2. GitHub SDK (`google/go-github v84`)
   1. Pros: typed responses, built-in rate limit error types, GitHub API changes are absorbed by SDK maintainers — no need to update request/response handling manually
   2. Cons: additional dependency, requires SDK upgrade on breaking GitHub API changes

---

## Chosen integration: `google/go-github v84`
The SDK eliminates boilerplate for JSON parsing and provides typed error types that make rate-limit handling straightforward. Maintenance of API contract changes is delegated to the SDK authors rather than owned by this project.

## Consequences
### Positive
* Typed `RateLimitError` enables clean rate-limit wait-and-retry logic
* No manual JSON unmarshalling for release responses
### Negative
* Dependency on an external SDK; major GitHub API changes require SDK upgrade
