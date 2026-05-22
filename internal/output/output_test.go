package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jun-uen0/aws-orphan-finder/internal/ebs"
)

func ptrFloat(f float64) *float64 { return &f }

func makeResult() ebs.Result {
	scanned, _ := time.Parse(time.RFC3339, "2026-05-22T00:00:00Z")
	created, _ := time.Parse(time.RFC3339, "2026-02-21T00:00:00Z")
	return ebs.Result{
		ScannedAt:    scanned,
		Region:       "ap-northeast-1",
		ResourceType: "ebs-volume",
		Orphans: []ebs.Orphan{
			{
				VolumeID:                "vol-0123456789abcdef0",
				Region:                  "ap-northeast-1",
				AvailabilityZone:        "ap-northeast-1a",
				SizeGiB:                 100,
				VolumeType:              "gp3",
				State:                   "available",
				CreateTime:              created,
				AgeDays:                 90,
				EstimatedMonthlyCostUSD: ptrFloat(8.0),
				Tags:                    map[string]string{"Name": "old"},
			},
		},
		Summary: ebs.Summary{
			Count:                   1,
			TotalSizeGiB:            100,
			EstimatedMonthlyCostUSD: ptrFloat(8.0),
			CostEstimateBasis:       "AWS Pricing API on-demand list price (ap-northeast-1)",
		},
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, makeResult()); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`"region": "ap-northeast-1"`,
		`"volumeId": "vol-0123456789abcdef0"`,
		`"estimatedMonthlyCostUSD": 8`,
		`"costEstimateBasis": "AWS Pricing API on-demand list price (ap-northeast-1)"`,
		`"resourceType": "ebs-volume"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON missing %q\n%s", want, out)
		}
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTable(&buf, makeResult()); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"VOLUME ID",
		"vol-0123456789abcdef0",
		"ap-northeast-1a",
		"$8.00",
		"1 orphan(s) in ap-northeast-1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q\n%s", want, out)
		}
	}
}

func TestWriteTable_NilCost(t *testing.T) {
	r := makeResult()
	r.Orphans[0].EstimatedMonthlyCostUSD = nil
	r.Summary.EstimatedMonthlyCostUSD = nil
	var buf bytes.Buffer
	if err := WriteTable(&buf, r); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "$") {
		t.Errorf("expected no dollar sign when cost is nil, got:\n%s", out)
	}
}

func TestWriteTable_EmptyOrphans(t *testing.T) {
	r := ebs.Result{
		Region:       "us-east-1",
		ResourceType: "ebs-volume",
		Orphans:      []ebs.Orphan{},
		Summary:      ebs.Summary{CostEstimateBasis: "x"},
	}
	var buf bytes.Buffer
	if err := WriteTable(&buf, r); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	if !strings.Contains(buf.String(), "0 orphan(s) in us-east-1") {
		t.Errorf("empty-orphan summary missing:\n%s", buf.String())
	}
}
