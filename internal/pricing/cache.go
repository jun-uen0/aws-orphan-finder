package pricing

import "sync"

// rateCache holds USD/GiB-month rates per EBS volume type for a single region.
// Population is one-shot: ensureLoaded fills the map once and subsequent
// lookups are pure reads. Mutex protects against parallel callers racing
// during the initial load.
type rateCache struct {
	mu     sync.RWMutex
	region string
	loaded bool
	rates  map[string]float64
}

func newRateCache(region string) *rateCache {
	return &rateCache{
		region: region,
		rates:  map[string]float64{},
	}
}
