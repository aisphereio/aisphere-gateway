package v1

import (
	"context"

	"github.com/aisphereio/kernel/accessx"
	accessv1 "github.com/aisphereio/kernel/api/aisphere/access/v1"
	"github.com/aisphereio/kernel/authz"
	"github.com/aisphereio/kernel/requestx"
)

var gatewayAdminRules = authz.Rules{
	"/gateway.v1.GatewayAdminService/GetRouteSnapshot": {
		Service:    "gateway.v1.GatewayAdminService",
		Method:     "GetRouteSnapshot",
		FullMethod: "/gateway.v1.GatewayAdminService/GetRouteSnapshot",
		Action:     "read",
		Resource:   "gateway:routes",
		Audience:   "gateway-service",
		Mode:       authz.RuleModeCheckOnly,
		AuditEvent: "gateway.routes.snapshot",
		AuditRisk:  "low",
	},
	"/gateway.v1.GatewayAdminService/ReloadRoutes": {
		Service:    "gateway.v1.GatewayAdminService",
		Method:     "ReloadRoutes",
		FullMethod: "/gateway.v1.GatewayAdminService/ReloadRoutes",
		Action:     "reload",
		Resource:   "gateway:routes",
		Audience:   "gateway-service",
		Mode:       authz.RuleModeCheckOnly,
		AuditEvent: "gateway.routes.reload",
		AuditRisk:  "medium",
	},
	"/gateway.v1.GatewayAdminService/GetUpstreamHealth": {
		Service:    "gateway.v1.GatewayAdminService",
		Method:     "GetUpstreamHealth",
		FullMethod: "/gateway.v1.GatewayAdminService/GetUpstreamHealth",
		Action:     "read",
		Resource:   "gateway:upstream:{service}",
		Audience:   "gateway-service",
		Mode:       authz.RuleModeCheckOnly,
		AuditEvent: "gateway.upstream.health",
		AuditRisk:  "low",
	},
	"/gateway.v1.GatewayAdminService/GetGatewayVersion": {
		Service:    "gateway.v1.GatewayAdminService",
		Method:     "GetGatewayVersion",
		FullMethod: "/gateway.v1.GatewayAdminService/GetGatewayVersion",
		Action:     "read",
		Resource:   "gateway:version",
		Audience:   "gateway-service",
		Mode:       authz.RuleModeCheckOnly,
		AuditEvent: "gateway.version",
		AuditRisk:  "low",
	},
}

func GatewayAdminServiceRequestInfoResolver(ctx context.Context, operation string, req any) (requestx.Info, bool, error) {
	_ = ctx
	_ = req
	rule, ok := gatewayAdminRules[normalizeGatewayAdminOperation(operation)]
	if !ok {
		return requestx.Info{}, false, nil
	}
	exposure := accessv1.Exposure_SYSTEM
	if rule.Method == "GetGatewayVersion" {
		exposure = accessv1.Exposure_PUBLIC
	}
	return requestx.Info{
		Service:       rule.Service,
		Method:        rule.Method,
		Operation:     rule.FullMethod,
		Exposure:      exposure,
		Action:        rule.Action,
		Resource:      rule.Resource,
		TargetService: rule.Audience,
		Labels:        map[string]string{"authz_mode": string(rule.Mode), "audit_event": rule.AuditEvent, "audit_risk": rule.AuditRisk},
	}.Normalize(), true, nil
}

func GatewayAdminServiceAccessResolver(ctx context.Context, operation string, req any) (accessx.Check, bool, error) {
	rule, ok := gatewayAdminRules[normalizeGatewayAdminOperation(operation)]
	if !ok {
		return accessx.Check{}, false, nil
	}
	resource, err := (authz.RuleResolver{}).ResolveResource(rule, req)
	if err != nil {
		return accessx.Check{}, true, err
	}
	_ = ctx
	return accessx.Check{
		Permission:  rule.Action,
		Resource:    resource,
		AuditAction: rule.AuditEvent,
		Metadata:    map[string]any{"authz_rule": rule.FullMethod, "authz_mode": string(rule.Mode)},
	}, true, nil
}

func normalizeGatewayAdminOperation(operation string) string {
	switch operation {
	case "GetRouteSnapshot", "gateway.v1.GatewayAdminService/GetRouteSnapshot":
		return "/gateway.v1.GatewayAdminService/GetRouteSnapshot"
	case "ReloadRoutes", "gateway.v1.GatewayAdminService/ReloadRoutes":
		return "/gateway.v1.GatewayAdminService/ReloadRoutes"
	case "GetUpstreamHealth", "gateway.v1.GatewayAdminService/GetUpstreamHealth":
		return "/gateway.v1.GatewayAdminService/GetUpstreamHealth"
	case "GetGatewayVersion", "gateway.v1.GatewayAdminService/GetGatewayVersion":
		return "/gateway.v1.GatewayAdminService/GetGatewayVersion"
	default:
		return operation
	}
}
