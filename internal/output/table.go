package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/jun-uen0/aws-orphan-finder/internal/ebs"
)

// WriteTable emits r as a column-aligned human-readable table followed by a
// single-line summary. Missing cost estimates render as "-".
func WriteTable(w io.Writer, r ebs.Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "VOLUME ID\tAZ\tSIZE\tTYPE\tAGE\tCOST/MO"); err != nil {
		return err
	}
	for _, o := range r.Orphans {
		cost := "-"
		if o.EstimatedMonthlyCostUSD != nil {
			cost = fmt.Sprintf("$%.2f", *o.EstimatedMonthlyCostUSD)
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%dGiB\t%s\t%dd\t%s\n",
			o.VolumeID, o.AvailabilityZone, o.SizeGiB, o.VolumeType, o.AgeDays, cost); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	totalCost := "-"
	if r.Summary.EstimatedMonthlyCostUSD != nil {
		totalCost = fmt.Sprintf("$%.2f", *r.Summary.EstimatedMonthlyCostUSD)
	}
	_, err := fmt.Fprintf(w, "\n%d orphan(s) in %s, %d GiB total, %s/month (%s)\n",
		r.Summary.Count, r.Region, r.Summary.TotalSizeGiB, totalCost, r.Summary.CostEstimateBasis)
	return err
}
