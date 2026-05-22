package ebs

import "time"

// BuildResult assembles a finished Result from an orphan list plus scan
// context. Per-orphan EstimatedMonthlyCostUSD must already be populated by
// the caller (or left nil when --no-pricing was used); BuildResult will
// only sum the non-nil entries into the summary total.
func BuildResult(orphans []Orphan, region, basis string, scannedAt time.Time) Result {
	if orphans == nil {
		orphans = []Orphan{}
	}
	var (
		totalSize int32
		totalCost float64
		anyCost   bool
	)
	for _, o := range orphans {
		totalSize += o.SizeGiB
		if o.EstimatedMonthlyCostUSD != nil {
			totalCost += *o.EstimatedMonthlyCostUSD
			anyCost = true
		}
	}
	summary := Summary{
		Count:             len(orphans),
		TotalSizeGiB:      totalSize,
		CostEstimateBasis: basis,
	}
	if anyCost {
		summary.EstimatedMonthlyCostUSD = &totalCost
	}
	return Result{
		ScannedAt:    scannedAt.UTC(),
		Region:       region,
		ResourceType: "ebs-volume",
		Orphans:      orphans,
		Summary:      summary,
	}
}
