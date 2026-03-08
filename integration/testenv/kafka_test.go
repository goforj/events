package testenv

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestStartKafka(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	env, err := StartKafka(ctx)
	if err != nil {
		t.Fatalf("StartKafka returned error: %v", err)
	}
	t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

	conn, err := net.DialTimeout("tcp", env.Brokers[0], 3*time.Second)
	if err != nil {
		t.Fatalf("DialTimeout returned error: %v", err)
	}
	_ = conn.Close()
}
