package main

import (
	"context"
	"flag"
	"time"

	kernel "github.com/aisphereio/kernel"
	"github.com/aisphereio/kernel/configx"
	configenv "github.com/aisphereio/kernel/configx/env"
	"github.com/aisphereio/kernel/configx/file"
	"github.com/aisphereio/kernel/dtmx"
	_ "github.com/aisphereio/kernel/dtmx/dtm"
	"github.com/aisphereio/kernel/gatewayx"
	"github.com/aisphereio/kernel/logx"
	"github.com/aisphereio/kernel/metricsx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/aisphereio/aisphere-gateway/internal/conf"
	"github.com/aisphereio/aisphere-gateway/internal/data"
	"github.com/aisphereio/aisphere-gateway/internal/dispatch"
	"github.com/aisphereio/aisphere-gateway/internal/registry"
	"github.com/aisphereio/aisphere-gateway/internal/server"
	"github.com/aisphereio/aisphere-gateway/internal/service"
	iamv1 "github.com/aisphereio/aisphere-iam/api/iam/v1"
)

var (
	Name     = "app"
	Version  = "dev"
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs/config.yaml", "config path, eg: -conf configs/config.yaml")
}

func main() {
	flag.Parse()

	cfg := configx.New(configx.WithSource(file.NewSource(flagconf), configenv.NewSource()))
	defer cfg.Close()
	if err := cfg.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := cfg.Scan(&bc); err != nil {
		panic(err)
	}
	applyBuildInfo(&bc)

	logger, _, err := logx.New(bc.Log)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	metrics := metricsx.Noop()
	if bc.Metrics.Enabled {
		metrics = metricsx.NewPrometheusManager(bc.Service.Name, bc.Service.Version, logger)
	}

	dtmManager, err := newDTMManager(bc, logger, metrics)
	if err != nil {
		panic(err)
	}
	defer func() { _ = dtmManager.Close() }()

	resources, cleanup, err := data.NewResources(context.Background(), bc, data.ResourceOptions{
		Logger:  logger,
		Metrics: metrics,
		DTM:     dtmManager,
	})
	if err != nil {
		panic(err)
	}
	defer cleanup()

	routeRegistry, registryCleanup, err := registry.NewRouteRegistry(context.Background(), registry.Config{
		Provider:       bc.Gateway.RouteRegistry.Provider,
		Prefix:         bc.Gateway.RouteRegistry.Prefix,
		Endpoints:      bc.Gateway.RouteRegistry.Endpoints,
		DialTimeout:    bc.Gateway.RouteRegistry.DialTimeout,
		RequestTimeout: bc.Gateway.RouteRegistry.RequestTimeout,
	})
	if err != nil {
		panic(err)
	}
	defer registryCleanup()

	hosts := gatewayx.StaticHosts{}
	for key, value := range bc.Gateway.Hosts {
		hosts[key] = value
	}
	for serviceName, upstream := range bc.Gateway.Upstreams {
		if upstream.Target != "" {
			hosts[serviceName+".aisphere"] = upstream.Target
		}
	}
	invokers := gatewayx.NewInvokerRegistry()
	iamConn, err := newGRPCClientConn(bc.Gateway.Upstreams["iam-service"].Target)
	if err != nil {
		panic(err)
	}
	if iamConn != nil {
		defer iamConn.Close()
		_ = iamv1.RegisterIAMAuthServiceGatewayInvokers(invokers, iamv1.NewIAMAuthServiceClient(iamConn))
		_ = iamv1.RegisterIAMDirectoryServiceGatewayInvokers(invokers, iamv1.NewIAMDirectoryServiceClient(iamConn))
		_ = iamv1.RegisterIAMPermissionServiceGatewayInvokers(invokers, iamv1.NewIAMPermissionServiceClient(iamConn))
	}
	bodyInvoker := dispatch.NewJSONBodyInvoker(invokers, map[string]dispatch.MessageFactory{
		"/iam.v1.IAMAuthService/BuildLoginURL":            func() proto.Message { return &iamv1.BuildLoginURLRequest{} },
		"/iam.v1.IAMAuthService/ExchangeCode":             func() proto.Message { return &iamv1.ExchangeCodeRequest{} },
		"/iam.v1.IAMAuthService/RefreshToken":             func() proto.Message { return &iamv1.RefreshTokenRequest{} },
		"/iam.v1.IAMAuthService/VerifyToken":              func() proto.Message { return &iamv1.VerifyTokenRequest{} },
		"/iam.v1.IAMAuthService/RevokeToken":              func() proto.Message { return &iamv1.RevokeTokenRequest{} },
		"/iam.v1.IAMAuthService/GetMe":                    func() proto.Message { return &iamv1.GetMeRequest{} },
		"/iam.v1.IAMDirectoryService/GetUser":             func() proto.Message { return &iamv1.GetUserRequest{} },
		"/iam.v1.IAMDirectoryService/ListUsers":           func() proto.Message { return &iamv1.ListUsersRequest{} },
		"/iam.v1.IAMDirectoryService/GetOrganization":     func() proto.Message { return &iamv1.GetOrganizationRequest{} },
		"/iam.v1.IAMDirectoryService/ListGroups":          func() proto.Message { return &iamv1.ListGroupsRequest{} },
		"/iam.v1.IAMPermissionService/CheckPermission":    func() proto.Message { return &iamv1.CheckPermissionRequest{} },
		"/iam.v1.IAMPermissionService/WriteRelationship":  func() proto.Message { return &iamv1.WriteRelationshipRequest{} },
		"/iam.v1.IAMPermissionService/DeleteRelationship": func() proto.Message { return &iamv1.DeleteRelationshipRequest{} },
		"/iam.v1.IAMPermissionService/LookupResources":    func() proto.Message { return &iamv1.LookupResourcesRequest{} },
		"/iam.v1.IAMPermissionService/LookupSubjects":     func() proto.Message { return &iamv1.LookupSubjectsRequest{} },
	})
	dispatcher := gatewayx.NewDispatcher(routeRegistry, hosts, bodyInvoker)
	adminService := service.NewGatewayAdminService(service.GatewayAdminDeps{
		Registry: routeRegistry,
		Hosts:    hosts,
		Name:     bc.Service.Name,
		Version:  bc.Service.Version,
	})
	httpServer := server.NewHTTPServer(bc.Server, bc.Log, bc.Metrics, logger, metrics, resources, adminService, dispatcher)
	grpcServer := server.NewGRPCServer(bc.Server, bc.Log, bc.Metrics, logger, metrics, adminService)

	options := []kernel.Option{
		kernel.Name(bc.Service.Name),
		kernel.Version(bc.Service.Version),
		kernel.LogxLogger(logger),
		kernel.Metrics(metrics),
		kernel.DTM(dtmManager),
		kernel.Server(httpServer, grpcServer),
		kernel.StopTimeout(10 * time.Second),
	}
	if bc.Metrics.Enabled && bc.Metrics.Addr != "" {
		options = append(options,
			kernel.PrometheusMetrics(bc.Metrics.Addr),
			kernel.MetricsPath(bc.Metrics.Path),
			kernel.MetricsPprof(bc.Metrics.Pprof),
		)
	}
	options = append(options, kernel.MetricsSystem(bc.Metrics.Enabled && bc.Metrics.Runtime))

	app := kernel.New(options...)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func applyBuildInfo(bc *conf.Bootstrap) {
	if bc.Service.Name == "" {
		bc.Service.Name = Name
	}
	if bc.Service.Version == "" {
		bc.Service.Version = Version
	}
	if bc.Service.Env == "" {
		bc.Service.Env = "local"
	}
	if bc.Log.ServiceName == "" {
		bc.Log.ServiceName = bc.Service.Name
	}
	if bc.Log.Env == "" {
		bc.Log.Env = bc.Service.Env
	}
	if bc.Log.Version == "" {
		bc.Log.Version = bc.Service.Version
	}
	if bc.Metrics.Path == "" {
		bc.Metrics.Path = "/metrics"
	}
	if bc.DTM.ServiceBaseURL == "" && bc.Server.HTTP.Addr != "" {
		bc.DTM.ServiceBaseURL = "http://127.0.0.1" + normalizeAddrPort(bc.Server.HTTP.Addr)
	}
}

func newDTMManager(bc conf.Bootstrap, logger logx.Logger, metrics metricsx.Manager) (dtmx.Manager, error) {
	cfg := bc.DTM
	cfg.Logger = logger.Named("dtmx")
	cfg.Metrics = metrics
	cfg.MetricsEnabled = cfg.MetricsEnabled && bc.Metrics.Enabled
	return dtmx.New(cfg)
}

func newGRPCClientConn(target string) (*grpc.ClientConn, error) {
	if target == "" {
		return nil, nil
	}
	return grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func normalizeAddrPort(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i:]
		}
	}
	return ":8000"
}
