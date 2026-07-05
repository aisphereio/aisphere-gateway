package data

import (
	"context"
	"errors"

	"github.com/aisphereio/kernel/accessx"
	"github.com/aisphereio/kernel/auditx"
	"github.com/aisphereio/kernel/authn"
	"github.com/aisphereio/kernel/authz"
	"github.com/aisphereio/kernel/authz/spicedb"
	"github.com/aisphereio/kernel/cachex"
	_ "github.com/aisphereio/kernel/cachex/redis"
	"github.com/aisphereio/kernel/dbx"
	_ "github.com/aisphereio/kernel/dbx/postgres"
	"github.com/aisphereio/kernel/dtmx"
	"github.com/aisphereio/kernel/logx"
	"github.com/aisphereio/kernel/metricsx"
	"github.com/aisphereio/kernel/objectstorex"
	_ "github.com/aisphereio/kernel/objectstorex/minio"
	"github.com/aisphereio/kernel/securityx"

	"github.com/aisphereio/aisphere-gateway/internal/conf"
)

type ResourceOptions struct {
	Logger  logx.Logger
	Metrics metricsx.Manager
	DTM     dtmx.Manager
}

type Resources struct {
	DB           dbx.DB
	Cache        cachex.Cache
	ObjectStore  objectstorex.Client
	Audit        auditx.Recorder
	Authn        authn.Authenticator
	AuthnRuntime *securityx.AuthnBoundaryRuntime
	Authz        authz.Authorizer
	Access       accessx.Guard
	DTM          dtmx.Manager

	closers []func() error
}

type Data struct {
	Resources *Resources
}

func NewResources(ctx context.Context, bc conf.Bootstrap, opts ResourceOptions) (*Resources, func(), error) {
	logger := opts.Logger
	if logger == nil {
		logger = logx.DefaultLogger()
	}
	metrics := metricsx.Ensure(opts.Metrics)

	r := &Resources{
		Audit: auditx.NewMemoryStore(),
		Authz: authz.DenyAll(),
		DTM:   dtmx.FromContextOr(ctx, opts.DTM),
	}
	if !bc.Audit.Enabled {
		r.Audit = auditx.Noop()
	}
	if bc.Security.Authz.DevAllowAll {
		r.Authz = authz.AllowAllForDevOnly()
	}

	if bc.Data.Database.Enabled {
		dbCfg := bc.Data.Database.Config
		dbCfg.Logger = logger.Named("data.dbx")
		dbCfg.Metrics = metrics
		dbCfg.MetricsEnabled = dbCfg.MetricsEnabled && bc.Metrics.Enabled
		db, err := dbx.New(dbCfg)
		if err != nil {
			return nil, nil, err
		}
		r.DB = db
		r.closers = append(r.closers, db.Close)
	}
	if bc.Data.Cache.Enabled {
		cacheCfg := bc.Data.Cache.Config
		cacheCfg.Logger = logger.Named("data.cachex")
		cacheCfg.Metrics = metrics
		cacheCfg.MetricsEnabled = cacheCfg.MetricsEnabled && bc.Metrics.Enabled
		cache, err := cachex.New(cacheCfg)
		if err != nil {
			r.Close()
			return nil, nil, err
		}
		r.Cache = cache
		r.closers = append(r.closers, cache.Close)
	}
	if bc.Data.ObjectStore.Enabled {
		storeCfg := bc.Data.ObjectStore.Config
		storeCfg.Logger = logger.Named("data.objectstorex")
		storeCfg.Metrics = metrics
		storeCfg.MetricsEnabled = storeCfg.MetricsEnabled && bc.Metrics.Enabled
		store, err := objectstorex.New(storeCfg)
		if err != nil {
			r.Close()
			return nil, nil, err
		}
		r.ObjectStore = store
		r.closers = append(r.closers, store.Close)
	}

// Authn: use Kernel auto-wiring framework (securityx.NewAuthnBoundaryRuntime).
		// This replaces the manual OIDC/JWKS verifier construction. The runtime
		// supports casdoor_jwt, gateway_trusted and hybrid modes.
		if bc.Security.Authn.Enabled {
			runtime, err := securityx.NewAuthnBoundaryRuntime(ctx, securityx.AuthnBoundaryConfig{
				Enabled:      bc.Security.Authn.Enabled,
				Mode:         firstNonEmpty(bc.Security.Authn.Mode, securityx.AuthnModeCasdoorJWT),
				Provider:     bc.Security.Authn.Provider,
				OIDC:         bc.Security.Authn.OIDC,
				InternalCall: bc.Security.InternalCall,
				CacheTTL:     bc.Security.Authn.CacheTTL,
			}, r.Cache)
			if err != nil {
				r.Close()
				return nil, nil, err
			}
			r.AuthnRuntime = runtime
			r.Authn = runtime.Authenticator
		}

	if bc.Security.Authz.Enabled && !bc.Security.Authz.DevAllowAll {
		authorizer, closeFn, err := newAuthorizer(bc.Security.Authz, logger, metrics, bc.Metrics.Enabled)
		if err != nil {
			r.Close()
			return nil, nil, err
		}
		r.Authz = authorizer
		if closeFn != nil {
			r.closers = append(r.closers, closeFn)
		}
	}

	r.Access = accessx.New(r.Authn, r.Authz, r.Audit)
	return r, func() { _ = r.Close() }, pingEnabled(ctx, r)
}

func NewData(resources *Resources) *Data {
	return &Data{Resources: resources}
}

func newAuthorizer(cfg conf.AuthzConfig, logger logx.Logger, metrics metricsx.Manager, metricsEnabled bool) (authz.Authorizer, func() error, error) {
	switch cfg.Provider {
	case "", "spicedb":
		spiceCfg := cfg.SpiceDB
		spiceCfg.Logger = logger.Named("authz.spicedb")
		spiceCfg.Metrics = metrics
		spiceCfg.MetricsEnabled = spiceCfg.MetricsEnabled && metricsEnabled
		client, err := spicedb.New(spiceCfg)
		if err != nil {
			return nil, nil, err
		}
		return client, client.Close, nil
	default:
		return nil, nil, errors.New("unsupported authz provider: " + cfg.Provider)
	}
}

func pingEnabled(ctx context.Context, r *Resources) error {
	if r.DB != nil {
		if err := r.DB.PingContext(ctx); err != nil {
			return err
		}
	}
	if r.Cache != nil {
		if err := r.Cache.Ping(ctx); err != nil {
			return err
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (r *Resources) Close() error {
	var out error
	for i := len(r.closers) - 1; i >= 0; i-- {
		if err := r.closers[i](); err != nil && out == nil {
			out = err
		}
	}
	return out
}