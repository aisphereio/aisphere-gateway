package conf

import (
	"time"

	"github.com/aisphereio/kernel/accessx"
	"github.com/aisphereio/kernel/authn"
	"github.com/aisphereio/kernel/authn/casdoor"
	"github.com/aisphereio/kernel/authn/oidcx"
	"github.com/aisphereio/kernel/authz/spicedb"
	"github.com/aisphereio/kernel/cachex"
	"github.com/aisphereio/kernel/dbx"
	"github.com/aisphereio/kernel/dtmx"
	"github.com/aisphereio/kernel/logx"
	"github.com/aisphereio/kernel/objectstorex"
	khttp "github.com/aisphereio/kernel/transportx/http"
)

type Bootstrap struct {
	Service  ServiceConfig  `json:"service" yaml:"service"`
	Server   ServerConfig   `json:"server" yaml:"server"`
	Log      logx.Config    `json:"log" yaml:"log"`
	Data     DataConfig     `json:"data" yaml:"data"`
	Security SecurityConfig `json:"security" yaml:"security"`
	Gateway  GatewayConfig  `json:"gateway" yaml:"gateway"`
	Audit    AuditConfig    `json:"audit" yaml:"audit"`
	Metrics  MetricsConfig  `json:"metrics" yaml:"metrics"`
	DTM      dtmx.Config    `json:"dtm" yaml:"dtm"`
}

type ServiceConfig struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	Env     string `json:"env" yaml:"env"`
}

type ServerConfig struct {
	HTTP HTTPConfig `json:"http" yaml:"http"`
	GRPC GRPCConfig `json:"grpc" yaml:"grpc"`
}

type HTTPConfig struct {
	Addr    string           `json:"addr" yaml:"addr"`
	Timeout time.Duration    `json:"timeout_ns" yaml:"timeout_ns"`
	CORS    khttp.CORSConfig `json:"cors" yaml:"cors"`
}

type GRPCConfig struct {
	Addr    string        `json:"addr" yaml:"addr"`
	Timeout time.Duration `json:"timeout_ns" yaml:"timeout_ns"`
}

type DataConfig struct {
	Database    DatabaseConfig    `json:"database" yaml:"database"`
	Cache       CacheConfig       `json:"cache" yaml:"cache"`
	ObjectStore ObjectStoreConfig `json:"object_store" yaml:"object_store"`
}

type DatabaseConfig struct {
	Enabled bool       `json:"enabled" yaml:"enabled"`
	Config  dbx.Config `json:"config" yaml:"config"`
}

type CacheConfig struct {
	Enabled bool          `json:"enabled" yaml:"enabled"`
	Config  cachex.Config `json:"config" yaml:"config"`
}

type ObjectStoreConfig struct {
	Enabled bool                `json:"enabled" yaml:"enabled"`
	Config  objectstorex.Config `json:"config" yaml:"config"`
}

type SecurityConfig struct {
	Authn        AuthnConfig                      `json:"authn" yaml:"authn"`
	Authz        AuthzConfig                      `json:"authz" yaml:"authz"`
	Access       accessx.AccessConfig             `json:"access" yaml:"access"`
	InternalCall authn.InternalServiceTokenConfig `json:"internal_call" yaml:"internal_call"`
}

type AuthnConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Mode     string `json:"mode" yaml:"mode"`
	Provider string `json:"provider" yaml:"provider"`

	// OIDC is the provider-neutral JWT verification config used by Gateway.
	// For Casdoor deployments, provider=casdoor can either fill this section or
	// reuse the legacy casdoor section below.
	OIDC oidcx.Config `json:"oidc" yaml:"oidc"`

	// Casdoor remains for compatibility with existing configs and IAM-style
	// hosted login/token exchange. Gateway verification maps these fields into
	// OIDC and does not require client_secret.
	Casdoor casdoor.Config `json:"casdoor" yaml:"casdoor"`

	// CacheTTL enables token->principal verification-result cache when data.cache
	// is enabled. TTL is always capped by token exp. Zero uses Kernel default.
	CacheTTL time.Duration `json:"cache_ttl_ns" yaml:"cache_ttl_ns"`
}

type AuthzConfig struct {
	Enabled     bool           `json:"enabled" yaml:"enabled"`
	Provider    string         `json:"provider" yaml:"provider"`
	DevAllowAll bool           `json:"dev_allow_all" yaml:"dev_allow_all"`
	SpiceDB     spicedb.Config `json:"spicedb" yaml:"spicedb"`
}

type GatewayConfig struct {
	RouteRegistry RouteRegistryConfig       `json:"route_registry" yaml:"route_registry"`
	Hosts         map[string]string         `json:"hosts" yaml:"hosts"`
	Upstreams     map[string]UpstreamConfig `json:"upstreams" yaml:"upstreams"`
}

type RouteRegistryConfig struct {
	Provider       string        `json:"provider" yaml:"provider"`
	Prefix         string        `json:"prefix" yaml:"prefix"`
	Endpoints      []string      `json:"endpoints" yaml:"endpoints"`
	DialTimeout    time.Duration `json:"dial_timeout_ns" yaml:"dial_timeout_ns"`
	RequestTimeout time.Duration `json:"request_timeout_ns" yaml:"request_timeout_ns"`
}

type UpstreamConfig struct {
	Target string `json:"target" yaml:"target"`
}

type AuditConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Store   string `json:"store" yaml:"store"`
}

type MetricsConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Addr    string `json:"addr" yaml:"addr"`
	Path    string `json:"path" yaml:"path"`
	Pprof   bool   `json:"pprof" yaml:"pprof"`
	Runtime bool   `json:"runtime" yaml:"runtime"`
}
