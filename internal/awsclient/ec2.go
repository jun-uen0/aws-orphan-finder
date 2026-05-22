// Package awsclient declares thin interfaces over the AWS SDK v2 clients used
// by this tool. Defining narrow interfaces here keeps the rest of the codebase
// decoupled from the SDK surface and lets tests substitute in-memory fakes.
package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// EC2VolumesAPI is the subset of the EC2 client used by the EBS orphan
// finder. The real *ec2.Client satisfies it via duck typing.
type EC2VolumesAPI interface {
	DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
}
