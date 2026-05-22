package pricing

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/pricing"
)

type fakePricing struct {
	pages [][]string
	err   error
	calls int
}

func (f *fakePricing) GetProducts(ctx context.Context, in *pricing.GetProductsInput, opts ...func(*pricing.Options)) (*pricing.GetProductsOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.calls >= len(f.pages) {
		return &pricing.GetProductsOutput{}, nil
	}
	page := f.pages[f.calls]
	f.calls++
	out := &pricing.GetProductsOutput{PriceList: page}
	if f.calls < len(f.pages) {
		token := "next"
		out.NextToken = &token
	}
	return out, nil
}

func TestEBSPricer_RateAndCache(t *testing.T) {
	fake := &fakePricing{
		pages: [][]string{{samplePriceListGp3, samplePriceListGp2}},
	}
	p := NewEBSPricer(fake, "ap-northeast-1")
	rate, err := p.Rate(context.Background(), "gp3")
	if err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if rate != 0.08 {
		t.Errorf("rate: got %f, want 0.08", rate)
	}
	if _, err := p.Rate(context.Background(), "gp2"); err != nil {
		t.Fatalf("Rate gp2: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("expected 1 API call (cached after), got %d", fake.calls)
	}
}

func TestEBSPricer_Pagination(t *testing.T) {
	fake := &fakePricing{
		pages: [][]string{
			{samplePriceListGp3},
			{samplePriceListGp2},
		},
	}
	p := NewEBSPricer(fake, "us-east-1")
	if _, err := p.Rate(context.Background(), "gp2"); err != nil {
		t.Fatalf("Rate gp2: %v", err)
	}
	if fake.calls != 2 {
		t.Errorf("expected 2 paginated API calls, got %d", fake.calls)
	}
}

func TestEBSPricer_Estimate(t *testing.T) {
	fake := &fakePricing{pages: [][]string{{samplePriceListGp3}}}
	p := NewEBSPricer(fake, "ap-northeast-1")
	cost, err := p.Estimate(context.Background(), 100, "gp3")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if cost != 8.0 {
		t.Errorf("cost: got %f, want 8.0", cost)
	}
}

func TestEBSPricer_UnknownVolumeType(t *testing.T) {
	fake := &fakePricing{pages: [][]string{{samplePriceListGp3}}}
	p := NewEBSPricer(fake, "us-east-1")
	_, err := p.Rate(context.Background(), "st1")
	if err == nil || !strings.Contains(err.Error(), "no rate found") {
		t.Errorf("expected no-rate-found error, got %v", err)
	}
}

func TestEBSPricer_EmptyPriceList(t *testing.T) {
	fake := &fakePricing{pages: [][]string{}}
	p := NewEBSPricer(fake, "us-east-1")
	_, err := p.Rate(context.Background(), "gp3")
	if err == nil || !strings.Contains(err.Error(), "no EBS rates") {
		t.Errorf("expected no-EBS-rates error, got %v", err)
	}
}

func TestEBSPricer_APIError(t *testing.T) {
	fake := &fakePricing{err: errors.New("denied")}
	p := NewEBSPricer(fake, "us-east-1")
	_, err := p.Rate(context.Background(), "gp3")
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Errorf("expected wrapped denied error, got %v", err)
	}
	if !strings.Contains(err.Error(), "pricing:GetProducts") {
		t.Errorf("expected IAM hint in error, got %v", err)
	}
}

func TestEBSPricer_NegativeSize(t *testing.T) {
	p := NewEBSPricer(&fakePricing{pages: [][]string{{samplePriceListGp3}}}, "us-east-1")
	_, err := p.Estimate(context.Background(), -1, "gp3")
	if err == nil {
		t.Fatal("expected error for negative size")
	}
}

func TestEBSPricer_NilClient(t *testing.T) {
	_, err := fetchEBSRates(context.Background(), nil, "us-east-1")
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestEBSPricer_Basis(t *testing.T) {
	p := NewEBSPricer(&fakePricing{}, "ap-northeast-1")
	if !strings.Contains(p.Basis(), "ap-northeast-1") {
		t.Errorf("Basis missing region: %q", p.Basis())
	}
}
