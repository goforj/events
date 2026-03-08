package testenv

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestStartRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	env, err := StartRedis(ctx)
	if err != nil {
		t.Fatalf("StartRedis returned error: %v", err)
	}
	t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

	conn, err := net.DialTimeout("tcp", env.Addr, 3*time.Second)
	if err != nil {
		t.Fatalf("DialTimeout returned error: %v", err)
	}
	_ = conn.Close()
}
