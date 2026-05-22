// Package pricing fetches AWS Pricing API rates for EBS storage and exposes
// a small Estimate API that other packages use to fill in per-volume cost.
//
// One scan loads the entire rate map for a region in a single call (a single
// paginated GetProducts request), then answers per-volume queries from an
// in-memory cache. The default behaviour on Pricing API failure is to return
// the error to the caller; the CLI translates that into a non-zero exit
// unless the user opted into --no-pricing.
package pricing

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	pricingtypes "github.com/aws/aws-sdk-go-v2/service/pricing/types"

	"github.com/jun-uen0/aws-orphan-finder/internal/awsclient"
)

const (
	serviceCodeEC2   = "AmazonEC2"
	productFamilyEBS = "Storage"

	// BasisListPrice is written into Result.Summary.CostEstimateBasis when
	// the AWS Pricing API was successfully consulted.
	BasisListPrice = "AWS Pricing API on-demand list price"

	// BasisSkipped is written into Result.Summary.CostEstimateBasis when
	// the user passed --no-pricing.
	BasisSkipped = "skipped (--no-pricing)"
)

// EBSPricer resolves EBS USD/GiB-month rates for a single AWS region.
type EBSPricer struct {
	client awsclient.PricingAPI
	region string
	cache  *rateCache
}

// NewEBSPricer constructs an EBSPricer for the region the EBS volumes live in.
// Note that this region is the *resource* region (e.g. ap-northeast-1), not
// the Pricing API endpoint region (us-east-1 / ap-south-1 / eu-central-1).
func NewEBSPricer(client awsclient.PricingAPI, region string) *EBSPricer {
	return &EBSPricer{
		client: client,
		region: region,
		cache:  newRateCache(region),
	}
}

// Rate returns the on-demand USD/GiB-month rate for a given EBS volume type
// (e.g. "gp3", "io2"). The first call triggers a single Pricing API fetch
// that loads all volume-type rates for the region into cache.
func (p *EBSPricer) Rate(ctx context.Context, volumeType string) (float64, error) {
	if err := p.ensureLoaded(ctx); err != nil {
		return 0, err
	}
	p.cache.mu.RLock()
	defer p.cache.mu.RUnlock()
	rate, ok := p.cache.rates[volumeType]
	if !ok {
		return 0, fmt.Errorf("pricing: no rate found for EBS volume type %q in region %s", volumeType, p.region)
	}
	return rate, nil
}

// Estimate returns the estimated monthly cost (USD) for a single volume of
// the given size and type.
func (p *EBSPricer) Estimate(ctx context.Context, sizeGiB int32, volumeType string) (float64, error) {
	if sizeGiB < 0 {
		return 0, fmt.Errorf("pricing: negative sizeGiB %d", sizeGiB)
	}
	rate, err := p.Rate(ctx, volumeType)
	if err != nil {
		return 0, err
	}
	return rate * float64(sizeGiB), nil
}

// Basis returns the human-readable cost-basis string for the scan output.
func (p *EBSPricer) Basis() string {
	return fmt.Sprintf("%s (%s)", BasisListPrice, p.region)
}

func (p *EBSPricer) ensureLoaded(ctx context.Context) error {
	p.cache.mu.RLock()
	loaded := p.cache.loaded
	p.cache.mu.RUnlock()
	if loaded {
		return nil
	}

	rates, err := fetchEBSRates(ctx, p.client, p.region)
	if err != nil {
		return err
	}

	p.cache.mu.Lock()
	defer p.cache.mu.Unlock()
	p.cache.rates = rates
	p.cache.loaded = true
	return nil
}

// fetchEBSRates pulls all per-GB EBS storage rates for a region in a single
// (possibly paginated) Pricing API request and returns volumeType -> USD/GiB-mo.
// The first sample wins on duplicate volumeType rows.
func fetchEBSRates(ctx context.Context, client awsclient.PricingAPI, region string) (map[string]float64, error) {
	if client == nil {
		return nil, errors.New("pricing: client is nil")
	}
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String(serviceCodeEC2),
		Filters: []pricingtypes.Filter{
			{
				Type:  pricingtypes.FilterTypeTermMatch,
				Field: aws.String("productFamily"),
				Value: aws.String(productFamilyEBS),
			},
			{
				Type:  pricingtypes.FilterTypeTermMatch,
				Field: aws.String("regionCode"),
				Value: aws.String(region),
			},
		},
	}

	rates := map[string]float64{}
	for {
		out, err := client.GetProducts(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("pricing:GetProducts: %w; required IAM action: pricing:GetProducts", err)
		}
		for _, item := range out.PriceList {
			volType, rate, ok, perr := parsePriceListItem(item)
			if perr != nil {
				return nil, fmt.Errorf("pricing: parse PriceList: %w", perr)
			}
			if !ok {
				continue
			}
			if _, exists := rates[volType]; !exists {
				rates[volType] = rate
			}
		}
		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		input.NextToken = out.NextToken
	}

	if len(rates) == 0 {
		return nil, fmt.Errorf("pricing: no EBS rates returned for region %s (check IAM permissions for pricing:GetProducts and that the region code is valid)", region)
	}
	return rates, nil
}
