// Command aws-orphan-finder scans an AWS region for orphaned (unattached)
// EBS volumes and prints them as JSON or a human-readable table, including
// an on-demand cost estimate fetched from the AWS Pricing API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/pricing"

	"github.com/jun-uen0/aws-orphan-finder/internal/ebs"
	"github.com/jun-uen0/aws-orphan-finder/internal/output"
	pkgpricing "github.com/jun-uen0/aws-orphan-finder/internal/pricing"
)

const (
	exitSuccess   = 0
	exitScanError = 1
	exitConfigErr = 2
)

type options struct {
	region        string
	minAgeDays    int
	outputFormat  string
	noPricing     bool
	pricingRegion string
}

func main() {
	opts, err := parseFlags("aws-orphan-finder", os.Args[1:], os.Stderr)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(exitConfigErr)
	}

	code := run(context.Background(), opts, os.Stdout, os.Stderr)
	os.Exit(code)
}

func parseFlags(progName string, args []string, errOut io.Writer) (options, error) {
	fs := flag.NewFlagSet(progName, flag.ContinueOnError)
	fs.SetOutput(errOut)
	var opts options
	fs.StringVar(&opts.region, "region", "", "AWS region to scan for EBS volumes (required, e.g. ap-northeast-1)")
	fs.IntVar(&opts.minAgeDays, "min-age-days", 0, "Skip volumes whose CreateTime is more recent than N days ago")
	fs.StringVar(&opts.outputFormat, "output", "json", "Output format: json or table")
	fs.BoolVar(&opts.noPricing, "no-pricing", false, "Skip AWS Pricing API lookup (estimatedMonthlyCostUSD will be null)")
	fs.StringVar(&opts.pricingRegion, "pricing-region", "us-east-1", "AWS Pricing API endpoint region (us-east-1, ap-south-1, or eu-central-1)")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if opts.region == "" {
		return opts, errors.New("--region is required")
	}
	if opts.outputFormat != "json" && opts.outputFormat != "table" {
		return opts, fmt.Errorf("--output must be json or table, got %q", opts.outputFormat)
	}
	if opts.minAgeDays < 0 {
		return opts, fmt.Errorf("--min-age-days must be non-negative, got %d", opts.minAgeDays)
	}
	return opts, nil
}

func run(ctx context.Context, opts options, stdout, stderr io.Writer) int {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		fmt.Fprintf(stderr, "error: AWS config load: %v\n", err)
		return exitConfigErr
	}

	finder := &ebs.Finder{
		Client:     ec2.NewFromConfig(cfg),
		Region:     opts.region,
		MinAgeDays: opts.minAgeDays,
	}
	orphans, err := finder.Find(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitScanError
	}

	basis := pkgpricing.BasisSkipped
	if opts.noPricing {
		fmt.Fprintln(stderr, "warning: --no-pricing set; estimatedMonthlyCostUSD will be null")
	} else {
		pCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.pricingRegion))
		if err != nil {
			fmt.Fprintf(stderr, "error: AWS config load for pricing: %v\n", err)
			return exitConfigErr
		}
		pricer := pkgpricing.NewEBSPricer(pricing.NewFromConfig(pCfg), opts.region)
		for i := range orphans {
			cost, err := pricer.Estimate(ctx, orphans[i].SizeGiB, orphans[i].VolumeType)
			if err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				return exitConfigErr
			}
			orphans[i].EstimatedMonthlyCostUSD = &cost
		}
		basis = pricer.Basis()
	}

	result := ebs.BuildResult(orphans, opts.region, basis, time.Now().UTC())

	switch opts.outputFormat {
	case "json":
		if err := output.WriteJSON(stdout, result); err != nil {
			fmt.Fprintf(stderr, "error: write JSON: %v\n", err)
			return exitScanError
		}
	case "table":
		if err := output.WriteTable(stdout, result); err != nil {
			fmt.Fprintf(stderr, "error: write table: %v\n", err)
			return exitScanError
		}
	}
	return exitSuccess
}
