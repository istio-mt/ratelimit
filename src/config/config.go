package config

import (
	"fmt"

	pb_struct "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	pb "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"golang.org/x/net/context"

	"github.com/envoyproxy/ratelimit/src/stats"
)

// Errors that may be raised during config parsing.
type RateLimitConfigError string

func (e RateLimitConfigError) Error() string {
	return string(e)
}

// Wrapper for an individual rate limit config entry which includes the defined limit and stats.
type RateLimit struct {
	FullKey    string
	Stats      stats.RateLimitStats
	Limit      *pb.RateLimitResponse_RateLimit
	Unlimited  bool
	ShadowMode bool
	Name       string
	Replaces   []string
}

// Interface for interacting with a loaded rate limit config.
type RateLimitConfig interface {
	// Dump the configuration into string form for debugging.
	Dump() string

	// Get the configured limit for a rate limit descriptor.
	// @param ctx supplies the calling context.
	// @param domain supplies the domain to lookup the descriptor in.
	// @param descriptor supplies the descriptor to look up.
	// @return a rate limit to apply or nil if no rate limit is configured for the descriptor.
	GetLimit(ctx context.Context, domain string, descriptor *pb_struct.RateLimitDescriptor) *RateLimit
}

type DiffType string

const (
	NoneType   DiffType = ""
	AddType    DiffType = "add"
	RemoveType DiffType = "remove"
	ModifyType DiffType = "modify"
)
// Message to apply diff configs to the  aggregate config.
type RateLimitMessage struct {
	NumPods             uint32
	RedisPipelineLength uint32
	InMemoryThreshold   uint32
	Configs             []RateLimitConfigToLoad
}
// Information for a config file to load into the aggregate config.
type RateLimitConfigToLoad struct {
	Name      string
	FileBytes string
	Diff      *RateLimitDiff
}
type RateLimitDiff struct {
	Root YamlRoot
	Type DiffType
}

func (rld *RateLimitDiff) String() string {
	return fmt.Sprintf("{Root:%v, Type:%v}", rld.Root, rld.Type)
}

type YamlReplaces struct {
	Name string
}

type YamlRateLimit struct {
	RequestsPerUnit uint32 `yaml:"requests_per_unit"`
	Unit            string
	Unlimited       bool `yaml:"unlimited"`
	Name            string
	Replaces        []YamlReplaces
}

type YamlDescriptor struct {
	Key         string
	Value       string
	RateLimit   *YamlRateLimit `yaml:"rate_limit"`
	Descriptors []YamlDescriptor
	ShadowMode  bool `yaml:"shadow_mode"`
}

type YamlRoot struct {
	Domain      string
	Descriptors []YamlDescriptor
}

// Interface for loading a configuration from a list of YAML files.
type RateLimitConfigLoader interface {
	// Load a new configuration from a list of YAML files.
	// @param configs supplies a list of full YAML files in string form.
	// @param statsManager supplies the statsManager to initialize stats during runtime.
	// @return a new configuration.
	// @throws RateLimitConfigError if the configuration could not be created.
	Load(configs []RateLimitConfigToLoad, statsManager stats.Manager) RateLimitConfig

	// Load configuration from a list of diffs.
	// @param configs supplies a list of diffs.
	// @param statsManager supplies the statsManager to initialize stats during runtime.
	// @return a new configuration.
	// @throws RateLimitConfigError if the configuration could not be created.
	LoadDiff(configs []RateLimitConfigToLoad, statsManager stats.Manager) RateLimitConfig
}
