package testenv

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestStartNATS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	env, err := StartNATS(ctx)
	if err != nil {
		t.Fatalf("StartNATS returned error: %v", err)
	}
	t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

	conn, err := net.DialTimeout("tcp", env.URL[len("nats://"):], 3*time.Second)
	if err != nil {
		t.Fatalf("DialTimeout returned error: %v", err)
	}
	_ = conn.Close()
}
