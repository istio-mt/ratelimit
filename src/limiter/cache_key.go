package limiter

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"hash/fnv"
	"strconv"
	"sync"

	"github.com/dgryski/go-farm"
	pb_struct "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	pb "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"github.com/reusee/mmh3"
	"github.com/zentures/cityhash"

	"github.com/envoyproxy/ratelimit/src/config"
	"github.com/envoyproxy/ratelimit/src/utils"
)

type CacheKey struct {
	Key string
	// True if the key corresponds to a limit with a SECOND unit. False otherwise.
	PerSecond bool
}

func isPerSecondLimit(unit pb.RateLimitResponse_RateLimit_Unit) bool {
	return unit == pb.RateLimitResponse_RateLimit_SECOND
}

type CacheKeyGenerator interface {
	GenerateCacheKey(domain string, descriptor *pb_struct.RateLimitDescriptor, limit *config.RateLimit, now int64) CacheKey
}

type humanReadableCacheKeyGenerator struct {
	prefix string
	// bytes.Buffer pool used to efficiently generate cache keys.
	bufferPool sync.Pool
}

func NewHumanReadableCacheKeyGenerator(prefix string) *humanReadableCacheKeyGenerator {
	return &humanReadableCacheKeyGenerator{
		prefix: prefix,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

func makeHumanReadableCacheKey(b *bytes.Buffer,
	prefix, domain string, descriptor *pb_struct.RateLimitDescriptor, limit *config.RateLimit, now int64) {
	b.WriteString(prefix)
	b.WriteString(domain)
	b.WriteByte('_')

	for _, entry := range descriptor.Entries {
		b.WriteString(entry.Key)
		b.WriteByte('_')
		b.WriteString(entry.Value)
		b.WriteByte('_')
	}

	divider := utils.UnitToDivider(limit.Limit.Unit)
	b.WriteString(strconv.FormatInt((now/divider)*divider, 10))
}

// Generate a cache key for a limit lookup.
// @param domain supplies the cache key domain.
// @param descriptor supplies the descriptor to generate the key for.
// @param limit supplies the rate limit to generate the key for (may be nil).
// @param now supplies the current unix time.
// @return CacheKey struct.
func (g *humanReadableCacheKeyGenerator) GenerateCacheKey(
	domain string, descriptor *pb_struct.RateLimitDescriptor, limit *config.RateLimit, now int64) CacheKey {

	if limit == nil {
		return CacheKey{
			Key:       "",
			PerSecond: false,
		}
	}

	b := g.bufferPool.Get().(*bytes.Buffer)
	defer g.bufferPool.Put(b)
	b.Reset()

	makeHumanReadableCacheKey(b, g.prefix, domain, descriptor, limit, now)

	return CacheKey{
		Key:       b.String(),
		PerSecond: isPerSecondLimit(limit.Limit.Unit),
	}
}

type hashCacheKeyGenerator struct {
	prefix string
	// bytes.Buffer pool used to efficiently generate cache keys.
	bufferPool sync.Pool
	hash       func(*bytes.Buffer) string
}

func NewHashCacheKeyGenerator(prefix string, hash func(*bytes.Buffer) string) *hashCacheKeyGenerator {
	return &hashCacheKeyGenerator{
		prefix: prefix,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		hash: hash,
	}
}

// Generate a cache key for a limit lookup.
// @param domain supplies the cache key domain.
// @param descriptor supplies the descriptor to generate the key for.
// @param limit supplies the rate limit to generate the key for (may be nil).
// @param now supplies the current unix time.
// @return CacheKey struct.
func (g *hashCacheKeyGenerator) GenerateCacheKey(
	domain string, descriptor *pb_struct.RateLimitDescriptor, limit *config.RateLimit, now int64) CacheKey {
	if limit == nil {
		return CacheKey{
			Key:       "",
			PerSecond: false,
		}
	}

	b := g.bufferPool.Get().(*bytes.Buffer)
	defer g.bufferPool.Put(b)
	b.Reset()

	makeHumanReadableCacheKey(b, g.prefix, domain, descriptor, limit, now)
	return CacheKey{
		Key:       g.hash(b),
		PerSecond: isPerSecondLimit(limit.Limit.Unit),
	}
}

func MurMurHash3(b *bytes.Buffer) string {
	return hex.EncodeToString(mmh3.Sum128(b.Bytes()))
}

func CityHash(b *bytes.Buffer) string {
	return hex.EncodeToString(cityhash.CityHash128(b.Bytes(), uint32(b.Len())).Bytes())
}

func FarmHash(b *bytes.Buffer) string {
	lo, hi := farm.Hash128(b.Bytes())
	x := make([]byte, 16)
	binary.LittleEndian.PutUint64(x, lo)
	binary.LittleEndian.PutUint64(x[8:], hi)
	return hex.EncodeToString(x)
}

func CRC32Hash(b *bytes.Buffer) string {
	return hex.EncodeToString(crc32.NewIEEE().Sum(b.Bytes()))
}

func FNVHash(b *bytes.Buffer) string {
	x := fnv.New128()
	x.Write(b.Bytes())
	return hex.EncodeToString(x.Sum(nil))
}
