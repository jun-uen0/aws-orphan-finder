package ebs

import "time"

// Orphan represents a single unattached EBS volume detected as orphaned.
// Region is intentionally absent: every Orphan in a Result shares the
// Result.Region value, so per-orphan duplication would be redundant.
type Orphan struct {
	VolumeID                string            `json:"volumeId"`
	AvailabilityZone        string            `json:"availabilityZone"`
	SizeGiB                 int32             `json:"sizeGiB"`
	VolumeType              string            `json:"volumeType"`
	State                   string            `json:"state"`
	CreateTime              time.Time         `json:"createTime"`
	AgeDays                 int               `json:"ageDays"`
	EstimatedMonthlyCostUSD *float64          `json:"estimatedMonthlyCostUSD"`
	Tags                    map[string]string `json:"tags"`
}

// Result is the aggregated scan output emitted by the CLI.
type Result struct {
	ScannedAt    time.Time `json:"scannedAt"`
	Region       string    `json:"region"`
	ResourceType string    `json:"resourceType"`
	Orphans      []Orphan  `json:"orphans"`
	Summary      Summary   `json:"summary"`
}

// Summary captures aggregate counts and totals for the scan.
type Summary struct {
	Count                   int      `json:"count"`
	TotalSizeGiB            int32    `json:"totalSizeGiB"`
	EstimatedMonthlyCostUSD *float64 `json:"estimatedMonthlyCostUSD"`
	CostEstimateBasis       string   `json:"costEstimateBasis"`
}
