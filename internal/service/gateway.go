package service

import (
	"context"
	"time"

	v1 "github.com/aisphereio/aisphere-gateway/api/gateway/v1"

	"github.com/aisphereio/kernel/gatewayx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GatewayAdminDeps struct {
	Registry gatewayx.RouteRegistry
	Hosts    gatewayx.StaticHosts
	Name     string
	Version  string
}

type GatewayAdminService struct {
	v1.UnimplementedGatewayAdminServiceServer
	deps GatewayAdminDeps
}

func NewGatewayAdminService(deps GatewayAdminDeps) *GatewayAdminService {
	return &GatewayAdminService{deps: deps}
}

func (s *GatewayAdminService) GetRouteSnapshot(ctx context.Context, req *v1.GetRouteSnapshotRequest) (*v1.GetRouteSnapshotReply, error) {
	_ = ctx
	_ = req
	var routes []gatewayx.GatewayRoute
	if s.deps.Registry != nil {
		routes = s.deps.Registry.ListRoutes()
	}
	out := &v1.GetRouteSnapshotReply{
		Revision:    snapshotRevision(routes),
		Source:      "route_registry",
		GeneratedAt: timestamppb.Now(),
		Routes:      make([]*v1.Route, 0, len(routes)),
	}
	for _, route := range routes {
		out.Routes = append(out.Routes, routeToProto(route))
	}
	return out, nil
}

func (s *GatewayAdminService) ReloadRoutes(ctx context.Context, req *v1.ReloadRoutesRequest) (*v1.ReloadRoutesReply, error) {
	snapshot, err := s.GetRouteSnapshot(ctx, &v1.GetRouteSnapshotRequest{})
	if err != nil {
		return nil, err
	}
	warnings := []string(nil)
	if req.GetDryRun() {
		warnings = append(warnings, "dry_run: route registry was read without mutating runtime state")
	}
	return &v1.ReloadRoutesReply{
		RouteCount: int32(len(snapshot.GetRoutes())),
		Revision:   snapshot.GetRevision(),
		Changed:    !req.GetDryRun(),
		Warnings:   warnings,
	}, nil
}

func (s *GatewayAdminService) GetUpstreamHealth(ctx context.Context, req *v1.GetUpstreamHealthRequest) (*v1.GetUpstreamHealthReply, error) {
	_ = ctx
	service := req.GetService()
	namespace := req.GetNamespace()
	if namespace == "" {
		namespace = "aisphere"
	}
	endpoints := []*v1.UpstreamEndpoint{}
	if s.deps.Registry != nil {
		for _, route := range s.deps.Registry.ListRoutes() {
			if route.Upstream.Service == service && (req.GetNamespace() == "" || route.Upstream.Namespace == namespace) {
				if resolved, err := s.deps.Hosts.Resolve(route.Upstream); err == nil {
					endpoints = append(endpoints, &v1.UpstreamEndpoint{
						Target: resolved,
						Port:   int32(route.Upstream.Port),
						Status: "configured",
					})
				} else {
					endpoints = append(endpoints, &v1.UpstreamEndpoint{
						Port:   int32(route.Upstream.Port),
						Status: "unknown",
						Reason: err.Error(),
					})
				}
			}
		}
	}
	status := "unknown"
	if len(endpoints) > 0 {
		status = "configured"
	}
	return &v1.GetUpstreamHealthReply{Service: service, Namespace: namespace, Status: status, Endpoints: endpoints}, nil
}

func (s *GatewayAdminService) GetGatewayVersion(ctx context.Context, req *v1.GetGatewayVersionRequest) (*v1.GetGatewayVersionReply, error) {
	_ = ctx
	_ = req
	return &v1.GetGatewayVersionReply{Name: s.deps.Name, Version: s.deps.Version}, nil
}

func routeToProto(in gatewayx.GatewayRoute) *v1.Route {
	return &v1.Route{
		Id:        in.ID,
		Name:      in.ID,
		Method:    in.Method,
		Path:      in.Path,
		Exposure:  in.Gateway.Exposure.String(),
		AuthnMode: string(in.Gateway.EffectiveAuthnMode()),
		Upstream: &v1.Upstream{
			Service:   in.Upstream.Service,
			Namespace: in.Upstream.Namespace,
			Port:      int32(in.Upstream.Port),
			Protocol:  in.Upstream.Protocol,
			Operation: in.Upstream.Operation,
		},
		Labels: map[string]string{
			"upstream": in.Upstream.Key(),
		},
	}
}

func snapshotRevision(routes []gatewayx.GatewayRoute) string {
	if len(routes) == 0 {
		return time.Now().UTC().Format("20060102T150405Z") + "-empty"
	}
	return time.Now().UTC().Format("20060102T150405Z")
}
