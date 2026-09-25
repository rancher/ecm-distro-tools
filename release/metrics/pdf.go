package metrics

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// column widths, out of a 13-col grid.
const (
	colImage    = 8
	colCritical = 1
	colHigh     = 1
	colMedium   = 1
	colLow      = 1
	colTotal    = 1
)

// buildReportPDF renders a single project's CVE report into a PDF document
// and returns its raw bytes
func buildReportPDF(minSeverity string, project ProjectReport) ([]byte, error) {
	cfg := config.NewBuilder().
		WithOrientation(orientation.Vertical).
		WithPageNumber().
		Build()

	m := maroto.New(cfg)
	m.AddRows(reportHeaderRows(minSeverity, project)...)

	for _, release := range project.Releases {
		if len(release.CVEs) == 0 {
			continue
		}
		m.AddRows(releaseSectionRows(release)...)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate pdf: %w", err)
	}

	return doc.GetBytes(), nil
}

func reportHeaderRows(minSeverity string, project ProjectReport) []core.Row {
	return []core.Row{
		row.New(14).Add(
			text.NewCol(
				12,
				fmt.Sprintf("%s CVE Report", project.Name),
				props.Text{
					Size:  18,
					Style: fontstyle.Bold,
					Align: align.Center,
				},
			),
		),
		row.New(8).Add(
			text.NewCol(
				12,
				fmt.Sprintf("Minimum severity: %s", minSeverity),
				props.Text{
					Size:  10,
					Align: align.Center,
				},
			),
		),
		row.New(8).Add(
			text.NewCol(
				12,
				fmt.Sprintf("Totals — Critical: %d  High: %d  Medium: %d  Low: %d  Other: %d",
					project.Totals.Critical, project.Totals.High, project.Totals.Medium, project.Totals.Low, project.Totals.Other),
				props.Text{
					Size:  10,
					Align: align.Center,
				},
			),
		),
	}
}

func releaseSectionRows(release ReleaseReport) []core.Row {
	rows := []core.Row{
		row.New(10).Add(
			text.NewCol(12, fmt.Sprintf("%s · %s", release.ProjectName, release.Release), props.Text{
				Size:  13,
				Style: fontstyle.Bold,
			}),
		),
		row.New(6).Add(
			text.NewCol(12,
				fmt.Sprintf("%d CVEs — Critical: %d  High: %d  Medium: %d  Low: %d",
					release.Counts.Total(), release.Counts.Critical, release.Counts.High,
					release.Counts.Medium, release.Counts.Low),
				props.Text{Size: 9, Style: fontstyle.Italic},
			),
		),
		tableHeaderRow(),
	}

	for _, img := range imagesForRelease(release.CVEs) {
		rows = append(rows, imageRow(img))
	}

	return rows
}

func tableHeaderRow() core.Row {
	headerProps := props.Text{
		Size:  8,
		Style: fontstyle.Bold,
		Align: align.Left,
	}
	return row.New(6).Add(
		text.NewCol(colImage, "Image", headerProps),
		text.NewCol(colCritical, "Critical", headerProps),
		text.NewCol(colHigh, "High", headerProps),
		text.NewCol(colMedium, "Medium", headerProps),
		text.NewCol(colLow, "Low", headerProps),
		text.NewCol(colTotal, "Total", headerProps),
	)
}

func imageRow(img ImageReport) core.Row {
	cellProps := props.Text{Size: 8}
	countProps := props.Text{
		Size:  8,
		Align: align.Left,
	}
	return row.New(5).Add(
		text.NewCol(colImage, img.Image, cellProps),
		text.NewCol(
			colCritical,
			fmt.Sprintf("%d", img.Counts.Critical),
			countProps,
		),
		text.NewCol(
			colHigh,
			fmt.Sprintf("%d", img.Counts.High),
			countProps,
		),
		text.NewCol(
			colMedium,
			fmt.Sprintf("%d", img.Counts.Medium),
			countProps,
		),
		text.NewCol(
			colLow,
			fmt.Sprintf("%d", img.Counts.Low),
			countProps,
		),
		text.NewCol(
			colTotal,
			fmt.Sprintf("%d", img.Counts.Total()),
			countProps,
		),
	)
}
