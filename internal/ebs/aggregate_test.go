package ebs

import (
	"testing"
	"time"
)

func ptrFloat(f float64) *float64 { return &f }

func TestBuildResult_EmptyOrphans(t *testing.T) {
	r := BuildResult(nil, "us-east-1", "x", time.Now())
	if r.Summary.Count != 0 {
		t.Errorf("Count: got %d, want 0", r.Summary.Count)
	}
	if r.Summary.TotalSizeGiB != 0 {
		t.Errorf("TotalSizeGiB: got %d, want 0", r.Summary.TotalSizeGiB)
	}
	if r.Summary.EstimatedMonthlyCostUSD != nil {
		t.Errorf("cost: got non-nil, want nil")
	}
	if r.Orphans == nil {
		t.Error("Orphans should be non-nil empty slice")
	}
	if r.ResourceType != "ebs-volume" {
		t.Errorf("ResourceType: got %q, want ebs-volume", r.ResourceType)
	}
}

func TestBuildResult_PartialCost(t *testing.T) {
	orphans := []Orphan{
		{SizeGiB: 100, EstimatedMonthlyCostUSD: ptrFloat(8.0)},
		{SizeGiB: 50, EstimatedMonthlyCostUSD: nil},
		{SizeGiB: 200, EstimatedMonthlyCostUSD: ptrFloat(16.0)},
	}
	r := BuildResult(orphans, "eu-west-1", "basis", time.Now())
	if r.Summary.Count != 3 {
		t.Errorf("Count: got %d, want 3", r.Summary.Count)
	}
	if r.Summary.TotalSizeGiB != 350 {
		t.Errorf("TotalSize: got %d, want 350", r.Summary.TotalSizeGiB)
	}
	if r.Summary.EstimatedMonthlyCostUSD == nil {
		t.Fatal("expected non-nil cost")
	}
	if got := *r.Summary.EstimatedMonthlyCostUSD; got != 24.0 {
		t.Errorf("cost: got %f, want 24.0", got)
	}
}

func TestBuildResult_NoCost(t *testing.T) {
	orphans := []Orphan{{SizeGiB: 100}, {SizeGiB: 50}}
	r := BuildResult(orphans, "ap-south-1", "skipped", time.Now())
	if r.Summary.EstimatedMonthlyCostUSD != nil {
		t.Errorf("cost: got non-nil, want nil when no orphan has a cost")
	}
	if r.Summary.TotalSizeGiB != 150 {
		t.Errorf("TotalSize: got %d, want 150", r.Summary.TotalSizeGiB)
	}
}
