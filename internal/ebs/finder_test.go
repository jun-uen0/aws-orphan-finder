package ebs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type fakeEC2 struct {
	pages [][]ec2types.Volume
	err   error
	calls int
}

func (f *fakeEC2) DescribeVolumes(ctx context.Context, in *ec2.DescribeVolumesInput, opts ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.calls >= len(f.pages) {
		return &ec2.DescribeVolumesOutput{}, nil
	}
	page := f.pages[f.calls]
	f.calls++
	out := &ec2.DescribeVolumesOutput{Volumes: page}
	if f.calls < len(f.pages) {
		token := "next"
		out.NextToken = &token
	}
	return out, nil
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrInt32(i int32) *int32        { return &i }
func ptrStr(s string) *string        { return &s }

func TestFinder_Find_FiltersAndPaginates(t *testing.T) {
	now := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	page1 := []ec2types.Volume{
		{
			VolumeId:         ptrStr("vol-0123456789abcdef0"),
			AvailabilityZone: ptrStr("ap-northeast-1a"),
			Size:             ptrInt32(100),
			VolumeType:       ec2types.VolumeTypeGp3,
			State:            ec2types.VolumeStateAvailable,
			CreateTime:       ptrTime(now.AddDate(0, 0, -60)),
			Tags: []ec2types.Tag{
				{Key: ptrStr("Name"), Value: ptrStr("old-data")},
			},
		},
		{
			VolumeId:         ptrStr("vol-0123456789abcdef1"),
			AvailabilityZone: ptrStr("ap-northeast-1c"),
			Size:             ptrInt32(50),
			VolumeType:       ec2types.VolumeTypeGp2,
			State:            ec2types.VolumeStateAvailable,
			CreateTime:       ptrTime(now.AddDate(0, 0, -5)),
		},
	}
	page2 := []ec2types.Volume{
		{
			VolumeId:         ptrStr("vol-0123456789abcdef2"),
			AvailabilityZone: ptrStr("ap-northeast-1d"),
			Size:             ptrInt32(200),
			VolumeType:       ec2types.VolumeTypeIo2,
			State:            ec2types.VolumeStateAvailable,
			CreateTime:       ptrTime(now.AddDate(0, 0, -90)),
		},
	}
	fake := &fakeEC2{pages: [][]ec2types.Volume{page1, page2}}
	finder := &Finder{
		Client:     fake,
		Region:     "ap-northeast-1",
		MinAgeDays: 30,
		Now:        func() time.Time { return now },
	}
	orphans, err := finder.Find(context.Background())
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got, want := len(orphans), 2; got != want {
		t.Fatalf("orphan count: got %d, want %d", got, want)
	}
	if fake.calls != 2 {
		t.Fatalf("DescribeVolumes calls: got %d, want 2", fake.calls)
	}
	for _, o := range orphans {
		if o.AgeDays < 30 {
			t.Errorf("volume %s slipped through MinAgeDays filter (age=%d)", o.VolumeID, o.AgeDays)
		}
		if o.Region != "ap-northeast-1" {
			t.Errorf("region: got %q, want ap-northeast-1", o.Region)
		}
	}
	if got := orphans[0].Tags["Name"]; got != "old-data" {
		t.Errorf("Tag Name: got %q, want old-data", got)
	}
}

func TestFinder_Find_NilClient(t *testing.T) {
	f := &Finder{}
	_, err := f.Find(context.Background())
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestFinder_Find_APIError(t *testing.T) {
	fake := &fakeEC2{err: errors.New("boom")}
	f := &Finder{Client: fake, Region: "us-east-1"}
	_, err := f.Find(context.Background())
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped boom, got %v", err)
	}
}

func TestFinder_Find_DefaultNow(t *testing.T) {
	fake := &fakeEC2{pages: [][]ec2types.Volume{{
		{VolumeId: ptrStr("vol-0"), Size: ptrInt32(10), CreateTime: ptrTime(time.Now().Add(-time.Hour))},
	}}}
	f := &Finder{Client: fake, Region: "us-east-1"}
	orphans, err := f.Find(context.Background())
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(orphans) != 1 {
		t.Errorf("expected 1 orphan with default Now, got %d", len(orphans))
	}
}

func TestFinder_toOrphan_SkipsNilVolumeID(t *testing.T) {
	f := &Finder{Region: "x"}
	_, ok := f.toOrphan(ec2types.Volume{}, time.Now())
	if ok {
		t.Fatal("expected skip for nil VolumeId")
	}
}

func TestFinder_toOrphan_ClockSkewClamped(t *testing.T) {
	f := &Finder{Region: "x"}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	future := now.AddDate(0, 0, 5)
	v := ec2types.Volume{
		VolumeId:   ptrStr("vol-future"),
		Size:       ptrInt32(1),
		CreateTime: ptrTime(future),
	}
	o, ok := f.toOrphan(v, now)
	if !ok {
		t.Fatal("expected orphan to be kept")
	}
	if o.AgeDays != 0 {
		t.Errorf("AgeDays for future CreateTime: got %d, want 0", o.AgeDays)
	}
}
