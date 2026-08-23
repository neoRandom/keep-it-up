package driven

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"keep-it-up/internal/infrastructure/constant"

	"github.com/valkey-io/valkey-go"
)

// idempotencyValueSeparator joins the status, status code, and (base64) body in
// a single stored value. base64url never emits '|', so it is safe as a delimiter.
const idempotencyValueSeparator = "|"

// ValkeyIdempotencyStore is the Valkey-backed implementation of the idempotency
// port required by the HTTP driver adapter. It stores each idempotency key as a
// single value so the state and the cached response share one TTL.
//
// The stored value format is:
//
//	IN_PROGRESS
//	COMPLETED|<status code>|<base64url body>
type ValkeyIdempotencyStore struct {
	client valkey.Client
}

func NewValkeyIdempotencyStore(client valkey.Client) *ValkeyIdempotencyStore {
	return &ValkeyIdempotencyStore{client: client}
}

// Acquire atomically claims key by writing IN_PROGRESS with SET NX EX. It
// reports true only when this call created the key, guaranteeing that exactly
// one caller proceeds for a given key.
func (s *ValkeyIdempotencyStore) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	cmd := s.client.B().Set().Key(key).Value(constant.IdempotencyStatusInProgress).Nx().Ex(ttl).Build()
	err := s.client.Do(ctx, cmd).Error()
	switch {
	case err == nil:
		return true, nil
	case valkey.IsValkeyNil(err):
		return false, nil
	default:
		return false, err
	}
}

// State returns the stored status and, when COMPLETED, the cached response.
// A missing key yields an empty status rather than an error.
func (s *ValkeyIdempotencyStore) State(ctx context.Context, key string) (string, int, []byte, error) {
	res := s.client.Do(ctx, s.client.B().Get().Key(key).Build())
	val, err := res.ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return "", 0, nil, nil
		}
		return "", 0, nil, err
	}

	if val == constant.IdempotencyStatusInProgress {
		return constant.IdempotencyStatusInProgress, 0, nil, nil
	}

	parts := strings.SplitN(val, idempotencyValueSeparator, 3)
	if len(parts) != 3 || parts[0] != constant.IdempotencyStatusCompleted {
		return "", 0, nil, fmt.Errorf("unexpected idempotency record %q", val)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, nil, fmt.Errorf("invalid status code in idempotency record: %w", err)
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", 0, nil, fmt.Errorf("invalid body in idempotency record: %w", err)
	}
	return constant.IdempotencyStatusCompleted, code, body, nil
}

// Complete transitions key to COMPLETED, persisting the response and resetting
// the TTL so the cached result expires together with the key.
func (s *ValkeyIdempotencyStore) Complete(ctx context.Context, key string, statusCode int, body []byte, ttl time.Duration) error {
	value := constant.IdempotencyStatusCompleted +
		idempotencyValueSeparator + strconv.Itoa(statusCode) +
		idempotencyValueSeparator + base64.RawURLEncoding.EncodeToString(body)

	cmd := s.client.B().Set().Key(key).Value(value).Ex(ttl).Build()
	return s.client.Do(ctx, cmd).Error()
}
