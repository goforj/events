package testenv

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestStartGCPPubSub(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	env, err := StartGCPPubSub(ctx)
	if err != nil {
		t.Fatalf("StartGCPPubSub returned error: %v", err)
	}
	t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

	conn, err := net.DialTimeout("tcp", env.URI, 3*time.Second)
	if err != nil {
		t.Fatalf("DialTimeout returned error: %v", err)
	}
	_ = conn.Close()
}
