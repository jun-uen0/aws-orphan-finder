// Package ebs implements detection of orphaned (unattached) EBS volumes.
package ebs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/jun-uen0/aws-orphan-finder/internal/awsclient"
)

// Finder scans a single region for orphaned EBS volumes.
//
// A volume is considered orphaned when its state is "available" (i.e.
// not attached to any instance). The MinAgeDays filter further restricts
// the result to volumes whose CreateTime is at least N days in the past.
type Finder struct {
	Client     awsclient.EC2VolumesAPI
	Region     string
	MinAgeDays int
	// Now is injected so tests can pin "current time". Defaults to time.Now().UTC().
	Now func() time.Time
}

// Find returns the list of orphaned volumes in the configured region.
// The returned slice is non-nil but may be empty.
func (f *Finder) Find(ctx context.Context) ([]Orphan, error) {
	if f.Client == nil {
		return nil, errors.New("ebs.Finder: Client is nil")
	}
	now := f.currentTime()

	input := &ec2.DescribeVolumesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("status"), Values: []string{"available"}},
		},
	}

	orphans := make([]Orphan, 0)
	for {
		out, err := f.Client.DescribeVolumes(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("ec2:DescribeVolumes: %w", err)
		}
		for _, v := range out.Volumes {
			o, ok := f.toOrphan(v, now)
			if !ok {
				continue
			}
			orphans = append(orphans, o)
		}
		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		input.NextToken = out.NextToken
	}
	return orphans, nil
}

// toOrphan converts a single SDK Volume into our domain Orphan, applying the
// MinAgeDays filter. Returns (_, false) if the volume should be skipped.
func (f *Finder) toOrphan(v ec2types.Volume, now time.Time) (Orphan, bool) {
	if v.VolumeId == nil {
		return Orphan{}, false
	}

	var createTime time.Time
	if v.CreateTime != nil {
		createTime = v.CreateTime.UTC()
	}

	ageDays := 0
	if !createTime.IsZero() {
		ageDays = int(now.Sub(createTime).Hours() / 24)
		if ageDays < 0 {
			ageDays = 0
		}
	}
	if ageDays < f.MinAgeDays {
		return Orphan{}, false
	}

	tags := map[string]string{}
	for _, t := range v.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}

	az := ""
	if v.AvailabilityZone != nil {
		az = *v.AvailabilityZone
	}
	var size int32
	if v.Size != nil {
		size = *v.Size
	}

	return Orphan{
		VolumeID:         *v.VolumeId,
		AvailabilityZone: az,
		SizeGiB:          size,
		VolumeType:       string(v.VolumeType),
		State:            string(v.State),
		CreateTime:       createTime,
		AgeDays:          ageDays,
		Tags:             tags,
	}, true
}

func (f *Finder) currentTime() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now().UTC()
}
