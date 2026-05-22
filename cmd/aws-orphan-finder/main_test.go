package main

import (
	"io"
	"strings"
	"testing"
)

func TestParseFlags_Defaults(t *testing.T) {
	opts, err := parseFlags("test", []string{"--region", "ap-northeast-1"}, io.Discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.region != "ap-northeast-1" {
		t.Errorf("region: got %q", opts.region)
	}
	if opts.outputFormat != "json" {
		t.Errorf("outputFormat: got %q", opts.outputFormat)
	}
	if opts.pricingRegion != "us-east-1" {
		t.Errorf("pricingRegion: got %q, want us-east-1", opts.pricingRegion)
	}
	if opts.noPricing {
		t.Errorf("noPricing: got true, want false by default")
	}
}

func TestParseFlags_MissingRegion(t *testing.T) {
	_, err := parseFlags("test", []string{}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "--region") {
		t.Errorf("expected missing --region error, got %v", err)
	}
}

func TestParseFlags_BadOutput(t *testing.T) {
	_, err := parseFlags("test", []string{"--region", "x", "--output", "yaml"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "--output") {
		t.Errorf("expected bad --output error, got %v", err)
	}
}

func TestParseFlags_NegativeAge(t *testing.T) {
	_, err := parseFlags("test", []string{"--region", "x", "--min-age-days", "-1"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "min-age-days") {
		t.Errorf("expected min-age-days error, got %v", err)
	}
}

func TestParseFlags_AllOptions(t *testing.T) {
	opts, err := parseFlags("test", []string{
		"--region", "eu-west-1",
		"--min-age-days", "30",
		"--output", "table",
		"--no-pricing",
		"--pricing-region", "eu-central-1",
	}, io.Discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.minAgeDays != 30 || opts.outputFormat != "table" || !opts.noPricing || opts.pricingRegion != "eu-central-1" {
		t.Errorf("unexpected opts: %+v", opts)
	}
}
