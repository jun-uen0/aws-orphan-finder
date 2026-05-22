package pricing

import (
	"encoding/json"
	"strconv"
	"strings"
)

// priceListItem mirrors the relevant subset of a single AWS Pricing API
// PriceList JSON string. Many fields are intentionally omitted.
type priceListItem struct {
	Product struct {
		ProductFamily string `json:"productFamily"`
		Attributes    struct {
			VolumeAPIName string `json:"volumeApiName"`
		} `json:"attributes"`
	} `json:"product"`
	Terms struct {
		OnDemand map[string]struct {
			PriceDimensions map[string]struct {
				Unit         string            `json:"unit"`
				PricePerUnit map[string]string `json:"pricePerUnit"`
			} `json:"priceDimensions"`
		} `json:"OnDemand"`
	} `json:"terms"`
}

// parsePriceListItem extracts the EBS volume type (e.g. "gp3") and the
// on-demand USD GB-Mo rate from a single Pricing API PriceList JSON string.
//
// Returns (volumeType, rate, found, err). found=false signals the row is not
// a per-GB EBS storage rate we care about (e.g. snapshot or IOPS dimension);
// the caller should skip it without treating it as an error.
func parsePriceListItem(raw string) (string, float64, bool, error) {
	var item priceListItem
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return "", 0, false, err
	}
	if item.Product.ProductFamily != productFamilyEBS {
		return "", 0, false, nil
	}
	volType := item.Product.Attributes.VolumeAPIName
	if volType == "" {
		return "", 0, false, nil
	}
	for _, term := range item.Terms.OnDemand {
		for _, dim := range term.PriceDimensions {
			if !isPerGBMonthlyUnit(dim.Unit) {
				continue
			}
			usd, ok := dim.PricePerUnit["USD"]
			if !ok {
				continue
			}
			rate, err := strconv.ParseFloat(usd, 64)
			if err != nil {
				return "", 0, false, err
			}
			return volType, rate, true, nil
		}
	}
	return "", 0, false, nil
}

// isPerGBMonthlyUnit reports whether a Pricing API priceDimensions.unit
// string represents per-GB-per-month storage. AWS is inconsistent across
// SKUs: gp3 / gp2 / io1 / st1 / sc1 report "GB-Mo", while io2 reports
// "GB-month". Both forms (and case variants) mean the same thing.
func isPerGBMonthlyUnit(unit string) bool {
	u := strings.ToLower(strings.TrimSpace(unit))
	return strings.HasPrefix(u, "gb-mo")
}
