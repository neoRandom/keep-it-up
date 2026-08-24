package constant

import "time"

const DBDatetimeFormat string = time.RFC3339

const EnvFilename string = ".env"

// Idempotency record states. These are shared between the Driver layer (which
// owns the idempotency port) and the Driven Valkey adapter so the two never
// drift apart on the stored value format.
const (
	IdempotencyStatusInProgress string = "IN_PROGRESS"
	IdempotencyStatusCompleted  string = "COMPLETED"
)
