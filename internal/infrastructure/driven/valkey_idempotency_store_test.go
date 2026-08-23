package driven

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"keep-it-up/internal/infrastructure/constant"

	"github.com/valkey-io/valkey-go"
)

// newTestStore connects to the Valkey instance under test and returns a store
// together with a cleanup that deletes every key this test created. It skips
// the test when no Valkey instance is reachable so the suite stays green in
// environments without one.
func newTestStore(t *testing.T) (*ValkeyIdempotencyStore, string, func()) {
	t.Helper()

	addr := os.Getenv("VALKEY_ADDRESS")
	if addr == "" {
		addr = "localhost:6379"
	}

	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{addr}})
	if err != nil {
		t.Fatalf("new valkey client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		t.Skipf("valkey not reachable at %s: %v", addr, err)
	}

	prefix := fmt.Sprintf("idempotency:test:%d:", time.Now().UnixNano())

	cleanup := func() {
		// Best-effort removal of every key this test created.
		keys, err := client.Do(context.Background(), client.B().Keys().Pattern(prefix+"*").Build()).AsStrSlice()
		if err == nil && len(keys) > 0 {
			_ = client.Do(context.Background(), client.B().Del().Key(keys...).Build()).Error()
		}
		client.Close()
	}
	t.Cleanup(cleanup)

	return NewValkeyIdempotencyStore(client), prefix, cleanup
}

func TestValkeyIdempotencyStore_Acquire(t *testing.T) {
	store, prefix, _ := newTestStore(t)
	ctx := context.Background()
	key := prefix + "acquire"

	acquired, err := store.Acquire(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !acquired {
		t.Fatal("first Acquire = false, want true")
	}

	acquired, err = store.Acquire(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if acquired {
		t.Fatal("second Acquire = true, want false (key already exists)")
	}
}

func TestValkeyIdempotencyStore_StateLifecycle(t *testing.T) {
	store, prefix, _ := newTestStore(t)
	ctx := context.Background()
	key := prefix + "lifecycle"

	// Unknown key reports an empty status.
	status, _, _, err := store.State(ctx, key)
	if err != nil {
		t.Fatalf("State (absent): %v", err)
	}
	if status != "" {
		t.Fatalf("State (absent) status = %q, want empty", status)
	}

	if _, err := store.Acquire(ctx, key, time.Minute); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	status, _, _, err = store.State(ctx, key)
	if err != nil {
		t.Fatalf("State (in progress): %v", err)
	}
	if status != constant.IdempotencyStatusInProgress {
		t.Fatalf("State status = %q, want %q", status, constant.IdempotencyStatusInProgress)
	}

	body := []byte(`{"message":"conflict","field":"a|b|c"}`)
	if err := store.Complete(ctx, key, 409, body, time.Minute); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	status, code, gotBody, err := store.State(ctx, key)
	if err != nil {
		t.Fatalf("State (completed): %v", err)
	}
	if status != constant.IdempotencyStatusCompleted {
		t.Fatalf("State status = %q, want %q", status, constant.IdempotencyStatusCompleted)
	}
	if code != 409 {
		t.Fatalf("State code = %d, want 409", code)
	}
	if string(gotBody) != string(body) {
		t.Fatalf("State body = %q, want %q", gotBody, body)
	}
}

func TestValkeyIdempotencyStore_TTLExpiry(t *testing.T) {
	store, prefix, _ := newTestStore(t)
	ctx := context.Background()
	key := prefix + "ttl"

	if _, err := store.Acquire(ctx, key, time.Second); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	time.Sleep(1100 * time.Millisecond)

	status, _, _, err := store.State(ctx, key)
	if err != nil {
		t.Fatalf("State (expired): %v", err)
	}
	if status != "" {
		t.Fatalf("State (expired) status = %q, want empty (key expired)", status)
	}
}
