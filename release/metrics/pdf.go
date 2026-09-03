package metrics

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// column widths, out of a 12-col grid.
const (
	colSeverity = 1
	colID       = 2
	colImage    = 3
	colPackage  = 3
	colStatus   = 3
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

	for _, cve := range release.CVEs {
		rows = append(rows, cveRow(cve))
	}

	return rows
}

func tableHeaderRow() core.Row {
	headerProps := props.Text{Size: 8, Style: fontstyle.Bold}
	return row.New(6).Add(
		text.NewCol(colSeverity, "Sev", headerProps),
		text.NewCol(colID, "CVE ID", headerProps),
		text.NewCol(colImage, "Image", headerProps),
		text.NewCol(colPackage, "Package / Version", headerProps),
		text.NewCol(colStatus, "Status", headerProps),
	)
}

func cveRow(cve CVE) core.Row {
	cellProps := props.Text{Size: 8}
	return row.New(5).Add(
		col.New(colSeverity).Add(text.New(severityEmoji(cve.Severity), cellProps)),
		text.NewCol(colID, cve.VulnerabilityID, cellProps),
		text.NewCol(colImage, cve.Image, cellProps),
		text.NewCol(colPackage, fmt.Sprintf("%s %s", cve.PackageName, cve.PackageVersion), cellProps),
		text.NewCol(colStatus, cve.Status, cellProps),
	)
}
