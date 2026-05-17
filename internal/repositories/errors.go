package repositories

import "errors"

// ErrNotFound is returned by repository methods when the requested record does not exist.
// Services and controllers should use errors.Is to detect it instead of depending on a specific driver.
var ErrNotFound = errors.New("record not found")
