package service

import (
	"context"
	"testing"

	v1 "aisphere-gateway/api/gateway/v1"

	accessv1 "github.com/aisphereio/kernel/api/aisphere/access/v1"
	"github.com/aisphereio/kernel/gatewayx"
)

func TestGatewayAdminServiceReturnsRouteSnapshotFromRegistry(t *testing.T) {
	registry := gatewayx.NewMemoryRegistry()
	if err := registry.RegisterManifest(gatewayx.Manifest{
		Service:   "iam-service",
		Namespace: "aisphere",
		Routes: []gatewayx.GatewayRoute{{
			ID:     "iam.get.me",
			Method: "GET",
			Path:   "/v1/iam/me",
			Upstream: gatewayx.UpstreamRef{
				Service:   "iam-service",
				Namespace: "aisphere",
				Protocol:  "grpc",
				Operation: "/iam.v1.IAMAuthService/GetMe",
			},
			Gateway: gatewayx.GatewayPolicy{
				Exposure:  accessv1.Exposure_AUTHENTICATED,
				AuthnMode: gatewayx.AuthnModePassive,
			},
		}},
	}); err != nil {
		t.Fatalf("RegisterManifest returned error: %v", err)
	}

	svc := NewGatewayAdminService(GatewayAdminDeps{
		Registry: registry,
		Name:     "aisphere-gateway",
		Version:  "test",
	})
	reply, err := svc.GetRouteSnapshot(context.Background(), &v1.GetRouteSnapshotRequest{})

	if err != nil {
		t.Fatalf("GetRouteSnapshot returned error: %v", err)
	}
	if len(reply.Routes) != 1 {
		t.Fatalf("route count = %d", len(reply.Routes))
	}
	if reply.Revision == "" || reply.Source != "route_registry" || reply.GeneratedAt == nil {
		t.Fatalf("snapshot metadata missing: %+v", reply)
	}
	if reply.Routes[0].AuthnMode != "passive" || reply.Routes[0].Upstream.Operation != "/iam.v1.IAMAuthService/GetMe" {
		t.Fatalf("unexpected route: %+v", reply.Routes[0])
	}
}

func TestGatewayAdminServiceReturnsUpstreamEndpoints(t *testing.T) {
	registry := gatewayx.NewMemoryRegistry()
	if err := registry.RegisterManifest(gatewayx.Manifest{
		Service:   "iam-service",
		Namespace: "aisphere",
		Routes: []gatewayx.GatewayRoute{{
			ID:     "iam.get.me",
			Method: "GET",
			Path:   "/v1/iam/me",
			Upstream: gatewayx.UpstreamRef{
				Service:   "iam-service",
				Namespace: "aisphere",
				Protocol:  "grpc",
				Operation: "/iam.v1.IAMAuthService/GetMe",
			},
		}},
	}); err != nil {
		t.Fatalf("RegisterManifest returned error: %v", err)
	}
	hosts := gatewayx.StaticHosts{"iam-service.aisphere": "127.0.0.1:19080"}
	svc := NewGatewayAdminService(GatewayAdminDeps{Registry: registry, Hosts: hosts})

	reply, err := svc.GetUpstreamHealth(context.Background(), &v1.GetUpstreamHealthRequest{Service: "iam-service"})

	if err != nil {
		t.Fatalf("GetUpstreamHealth returned error: %v", err)
	}
	if reply.Namespace != "aisphere" || reply.Status != "configured" || len(reply.Endpoints) != 1 {
		t.Fatalf("unexpected upstream health: %+v", reply)
	}
	if reply.Endpoints[0].Target != "127.0.0.1:19080" {
		t.Fatalf("endpoint = %+v", reply.Endpoints[0])
	}
}

func TestGatewayAdminServiceReturnsVersion(t *testing.T) {
	svc := NewGatewayAdminService(GatewayAdminDeps{Name: "aisphere-gateway", Version: "dev"})

	reply, err := svc.GetGatewayVersion(context.Background(), &v1.GetGatewayVersionRequest{})

	if err != nil {
		t.Fatalf("GetGatewayVersion returned error: %v", err)
	}
	if reply.Name != "aisphere-gateway" || reply.Version != "dev" {
		t.Fatalf("version reply = %+v", reply)
	}
}
