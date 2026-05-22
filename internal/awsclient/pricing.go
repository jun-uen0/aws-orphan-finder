package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/pricing"
)

// PricingAPI is the subset of the AWS Pricing client used to look up EBS
// USD/GiB-month rates. The real *pricing.Client satisfies it via duck typing.
//
// The AWS Pricing service is only available in the us-east-1, ap-south-1,
// and eu-central-1 endpoints; the caller is responsible for constructing the
// client against one of those regions even when querying prices for a
// different region's resources.
type PricingAPI interface {
	GetProducts(ctx context.Context, params *pricing.GetProductsInput, optFns ...func(*pricing.Options)) (*pricing.GetProductsOutput, error)
}
