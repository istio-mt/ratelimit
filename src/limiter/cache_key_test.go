package limiter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	pb_struct "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	pb "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"github.com/envoyproxy/ratelimit/src/config"
)

func BenchmarkCacheKeyGenerator(b *testing.B) {
	const prefix = "a_prefix"
	const domain = "ratelimit.io"
	var (
		limitConfig = &config.RateLimit{
			Limit: &pb.RateLimitResponse_RateLimit{
				RequestsPerUnit: 10086,
				Unit:            pb.RateLimitResponse_RateLimit_SECOND,
			},
		}
		descriptor = &pb_struct.RateLimitDescriptor{
			Entries: []*pb_struct.RateLimitDescriptor_Entry{
				{
					Key:   "PATH",
					Value: "/foo",
				},
				{
					Key:   "HEADER",
					Value: "X-Token",
				},
				{
					Key:   "key1",
					Value: "value1",
				},
				{
					Key:   "long-key1",
					Value: strings.Repeat("a", 128),
				},
				{
					Key:   "long-key2",
					Value: strings.Repeat("b", 128),
				},
			},
		}
	)
	b.Run("humanReadable", benchmarkCacheKeyGenerator(NewHumanReadableCacheKeyGenerator(prefix), domain, prefix, descriptor, limitConfig, 0))
	for _, hashSample := range []struct {
		name string
		hash func(*bytes.Buffer) string
	}{
		{
			name: "MurMurHash3",
			hash: MurMurHash3,
		},
		{
			name: "CityHash",
			hash: CityHash,
		},
		{
			name: "FarmHash",
			hash: FarmHash,
		},
		{
			name: "FNV",
			hash: FNVHash,
		},
		{
			name: "CRC32",
			hash: CRC32Hash,
		},
	} {
		b.Run("hash_"+hashSample.name, benchmarkCacheKeyGenerator(
			NewHashCacheKeyGenerator(prefix, hashSample.hash), domain, prefix, descriptor, limitConfig, 0))
	}
}

func benchmarkCacheKeyGenerator(generator CacheKeyGenerator, domain, prefix string,
	descriptor *pb_struct.RateLimitDescriptor, limit *config.RateLimit, now int64) func(*testing.B) {
	key := generator.GenerateCacheKey(domain, descriptor, limit, now)
	fmt.Printf("Key size: %d\n", len(key.Key))
	return func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			generator.GenerateCacheKey(domain, descriptor, limit, now)
		}
	}
}
