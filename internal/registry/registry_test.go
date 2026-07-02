package registry

import (
	"context"
	"testing"

	"github.com/aisphereio/kernel/gatewayx"
)

func TestNewRouteRegistryDefaultsToMemory(t *testing.T) {
	registry, cleanup, err := NewRouteRegistry(context.Background(), Config{})
	if err != nil {
		t.Fatalf("NewRouteRegistry returned error: %v", err)
	}
	defer cleanup()

	if _, ok := registry.(*gatewayx.MemoryRegistry); !ok {
		t.Fatalf("registry type = %T", registry)
	}
}

func TestNewRouteRegistryRejectsEtcdWithoutEndpoints(t *testing.T) {
	_, _, err := NewRouteRegistry(context.Background(), Config{Provider: "etcd"})
	if err == nil {
		t.Fatal("expected error for etcd registry without endpoints")
	}
}
