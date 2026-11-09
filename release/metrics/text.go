package metrics

import (
	"fmt"
	"strings"
)

// Slack recommends keeping message text well under 4000 chars.
const slackMessageLimit = 3500

// buildProjectText renders a project's report as one or more Slack mrkdwn
// messages. The first message carries the usergroup mention.
func buildProjectText(minSeverity string, project ProjectReport, usergroupID string) []string {
	var messages []string
	var b strings.Builder

	flush := func() {
		if b.Len() > 0 {
			messages = append(messages, b.String())
			b.Reset()
		}
	}
	write := func(line string) {
		if b.Len()+len(line) > slackMessageLimit {
			flush()
		}
		b.WriteString(line)
	}

	header := fmt.Sprintf("*%s CVE Report* | Min severity: %s | %d findings\n",
		project.Name, minSeverity, project.Totals.Total())
	if usergroupID != "" {
		header = fmt.Sprintf("<!subteam^%s> %s", usergroupID, header)
	}
	write(header)

	for _, release := range project.Releases {
		if len(release.CVEs) == 0 {
			continue
		}

		write(fmt.Sprintf("\n*%s*  %s\n", release.Release, countsLine(release.Counts)))

		for _, img := range imagesForRelease(release.CVEs) {
			// backticks stop Slack from auto-linking image names
			write(fmt.Sprintf("`%s`  %s\n", img.Image, countsLine(img.Counts)))
		}
	}

	flush()
	return messages
}

func countsLine(c SeverityCounts) string {
	return fmt.Sprintf("%s %d | %s %d | %s %d | %s %d",
		emojiCritical, c.Critical,
		emojiHigh, c.High,
		emojiMedium, c.Medium,
		emojiLow, c.Low,
	)
}
