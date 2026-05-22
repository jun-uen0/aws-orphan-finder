# aws-orphan-finder

Find orphaned AWS EBS volumes that are quietly costing you money.

[![CI](https://github.com/jun-uen0/aws-orphan-finder/actions/workflows/ci.yml/badge.svg)](https://github.com/jun-uen0/aws-orphan-finder/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jun-uen0/aws-orphan-finder)](https://goreportcard.com/report/github.com/jun-uen0/aws-orphan-finder)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Why this exists

In multi-account AWS environments, EBS volumes in the `available` state (i.e. not attached to any instance) accumulate quietly over time and continue to bill at full per-GB-month rates. `aws-orphan-finder` scans a region, lists every such volume with size, age, and an on-demand cost estimate pulled live from the AWS Pricing API, and emits the result as JSON or a human-readable table. The output is designed to pipe cleanly into `jq` or a spreadsheet so you can use it as the starting point for a cleanup ticket, an inventory audit, or a periodic cost-review job.

### Where this fits among existing tools

- **AWS Trusted Advisor** surfaces the same "underutilized EBS volumes" finding with cost estimates, but requires Business Support or higher (USD 100+/month). If you are on Basic or Developer support, `aws-orphan-finder` covers the same ground from a single binary.
- **AWS Config rule `ec2-volume-inuse-check`** flags `available` volumes too, but does not estimate cost and requires Config to be enabled and to have evaluated the volume.
- **`cloud-nuke` / `aws-nuke`** can both detect and delete resources, which is useful when you are deliberately tearing down. `aws-orphan-finder` is read-only by design — it produces a list, not an outcome.
- **Steampipe / Powerpipe** offer SQL-over-AWS for arbitrary inventory queries including cost. The niche here is one-off CLI usage without a persistent daemon, a plugin install, or a SQL learning curve.
- **`aws ec2 describe-volumes --filters Name=status,Values=available`** is a one-liner, but does not estimate cost. The Pricing API integration is what makes the output actionable rather than informational.

### Scope and limits (v0.1)

- **Orphan = `state=available`**. Stopped EC2 instances with EBS still attached are **not** flagged — those volumes report `in-use` even though they continue to bill at the same per-GB-month rate. A flag for that case is on the roadmap.
- **Cost is GB-Mo only.** The per-GB storage rate is summed; **provisioned IOPS surcharges (io1/io2) and gp3 baseline-over IOPS/throughput are not yet added**. For an io2 volume with 50k provisioned IOPS, GB cost can be a small fraction of the bill, so the v0.1 estimate is conservative — see v0.1.1 in the roadmap.
- **Snapshots are not surfaced.** Orphan volumes frequently leave behind orphan snapshots that keep billing at `$0.05/GB-Mo`; listing them lands in v0.1.1.
- **Single region per invocation.** Cross-region scanning lands in v0.2.

## Install

```bash
go install github.com/jun-uen0/aws-orphan-finder/cmd/aws-orphan-finder@latest
```

This drops a single static binary into `$(go env GOBIN)` (defaults to `$GOPATH/bin` or `~/go/bin`). No runtime dependencies.

## Usage

```bash
# Scan the Tokyo region, emit JSON (default)
aws-orphan-finder --region ap-northeast-1

# Scan us-east-1 and only show volumes older than 30 days, as a table
aws-orphan-finder --region us-east-1 --min-age-days 30 --output table

# Skip the AWS Pricing API entirely (cost will be null in the output)
aws-orphan-finder --region eu-west-1 --no-pricing
```

Authentication uses the standard AWS SDK v2 credential chain: environment variables, shared config (`~/.aws/credentials`), IMDS, and SSO are all supported, in that order. No extra configuration file is needed.

### Required IAM permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DescribeEBSVolumes",
      "Effect": "Allow",
      "Action": "ec2:DescribeVolumes",
      "Resource": "*"
    },
    {
      "Sid": "FetchEBSPrices",
      "Effect": "Allow",
      "Action": [
        "pricing:GetProducts",
        "pricing:DescribeServices"
      ],
      "Resource": "*"
    }
  ]
}
```

When invoked with `--no-pricing` only `ec2:DescribeVolumes` is required; the `pricing:*` permissions can be omitted.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--region` | (required) | AWS region to scan for EBS volumes, e.g. `ap-northeast-1` |
| `--min-age-days` | `0` | Skip volumes whose `CreateTime` is more recent than N days ago |
| `--output` | `json` | Output format: `json` or `table` |
| `--no-pricing` | `false` | Skip the AWS Pricing API lookup; `estimatedMonthlyCostUSD` will be `null` |
| `--pricing-region` | `us-east-1` | AWS Pricing API endpoint region (must be `us-east-1`, `ap-south-1`, or `eu-central-1`) |

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Scan completed successfully |
| `1` | Scan failed (e.g. `ec2:DescribeVolumes` error) |
| `2` | Configuration or permission error (e.g. Pricing API call failed and `--no-pricing` was not set) |

## Output schema

```json
{
  "scannedAt": "2026-05-22T14:30:00Z",
  "region": "ap-northeast-1",
  "resourceType": "ebs-volume",
  "orphans": [
    {
      "volumeId": "vol-0123456789abcdef0",
      "availabilityZone": "ap-northeast-1a",
      "sizeGiB": 100,
      "volumeType": "gp3",
      "state": "available",
      "createTime": "2025-12-01T10:00:00Z",
      "ageDays": 172,
      "estimatedMonthlyCostUSD": 8,
      "tags": {"Name": "old-ec2-data"}
    }
  ],
  "summary": {
    "count": 1,
    "totalSizeGiB": 100,
    "estimatedMonthlyCostUSD": 8,
    "costEstimateBasis": "AWS Pricing API on-demand list price (ap-northeast-1)"
  }
}
```

Piping into `jq` is the intended common case:

```bash
aws-orphan-finder --region ap-northeast-1 \
  | jq '.orphans[] | select(.ageDays > 90) | {volumeId, sizeGiB, estimatedMonthlyCostUSD}'
```

## Design decisions

- **Go, single binary** — cross-platform distribution is trivial, AWS SDK v2 is a first-class option, and the resulting tool drops cleanly into CI environments without a Python runtime.
- **Live Pricing API, not a baked-in table** — list prices change. Fetching them at scan time keeps cost estimates honest and removes the maintenance burden of a hard-coded rate table. The Pricing endpoint is only hosted in `us-east-1` / `ap-south-1` / `eu-central-1`, so the tool transparently opens a second SDK config against `--pricing-region` while the EBS scan still runs in the resource region.
- **Pricing failures are fatal by default** — if `pricing:GetProducts` is denied, the tool exits 2 with an actionable error message. Use `--no-pricing` to opt into a degraded scan that still emits orphan IDs and sizes without cost numbers. The intent is to make missing cost data a deliberate choice rather than a silent fallback.
- **Interface-first AWS clients** — `internal/awsclient` defines narrow `EC2VolumesAPI` and `PricingAPI` interfaces that the SDK clients satisfy via duck typing. This is what makes the table-driven unit tests possible without LocalStack or a recorded HTTP fixture.
- **One Pricing fetch per scan** — `internal/pricing.EBSPricer` caches the entire region's rate map after the first `Rate` call. A scan of 1,000 volumes still makes one Pricing call, not 1,000.

## Roadmap

- **v0.1.1 — accurate EBS cost.** Sum the provisioned-IOPS surcharge for io1/io2 (`IOPS-Mo` dimension) and the gp3 baseline-over surcharge; look up the snapshots associated with each orphan volume (`ec2:DescribeSnapshots --filters volume-id`) and add their `$/GB-Mo` storage. Without these, the v0.1 estimate is systematically low for io-class volumes and silent about snapshot tail.
- **v0.2 — broader orphan classes.** Elastic IP, NAT Gateway, idle ALB/NLB, and orphan ENI detection; opt-in flag for "stopped EC2 with attached EBS" (the volumes are technically `in-use` but billed). Multi-region scan in a single invocation. Expanded CI matrix (`os: [ubuntu-latest, macos-latest, windows-latest] x go: ['1.24']`) with `staticcheck`, `golangci-lint`, and `-race`.
- **v0.3 — observability.** Age-via-CloudTrail (when was the volume last detached, not when was it created); CloudWatch Logs retention auditing; old AMI plus dependent snapshots; coverage upload to Codecov.
- **v0.4 — distribution.** Homebrew tap; pre-built GitHub Release binaries via `goreleaser`.

Issues and pull requests welcome.

## License

MIT. See [LICENSE](LICENSE).
