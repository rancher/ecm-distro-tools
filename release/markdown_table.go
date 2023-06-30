package release

import (
	"fmt"
	"strings"
)

type markdownTable struct {
	header    []string
	headerLen int
	values    [][]string
}

func NewMarkdownTable(header []string, values [][]string) (markdownTable, error) {
	headerLen := len(header)
	for row, rowValue := range values {
		rowLen := len(rowValue)
		if rowLen != headerLen {
			return markdownTable{}, fmt.Errorf("failed to create markdown table, row %d was expected to have length %d, but it has length %d", row, headerLen, rowLen)
		}
	}
	table := markdownTable{
		header:    header,
		values:    values,
		headerLen: headerLen,
	}
	return table, nil
}

func (m *markdownTable) maxWidths() []int {
	maxWidths := make([]int, m.headerLen)

	for _, rowValue := range m.values {
		for column, columnValue := range rowValue {
			if maxWidths[column] == 0 {
				maxWidths[column] = len(m.header[column])
			}
			if len(columnValue) > maxWidths[column] {
				maxWidths[column] = len(columnValue)
			}
		}
	}
	return maxWidths
}

func (m *markdownTable) String() string {
	var sb strings.Builder

	valueFormat := "|" + strings.Repeat(" %-*s |", m.headerLen) + "\n"
	separatorFormat := "|" + strings.Repeat(" %s |", m.headerLen) + "\n"
	headerFormatArguments := make([]interface{}, 0)
	separatorFormatArguments := make([]interface{}, 0)
	maxWidths := m.maxWidths()

	for i := 0; i < m.headerLen; i++ {
		headerFormatArguments = append(headerFormatArguments, maxWidths[i])
		headerFormatArguments = append(headerFormatArguments, m.header[i])
		separatorFormatArguments = append(separatorFormatArguments, strings.Repeat("-", maxWidths[i]))
	}

	header := fmt.Sprintf(valueFormat, headerFormatArguments...)
	sb.WriteString(header)
	separator := fmt.Sprintf(separatorFormat, separatorFormatArguments...)
	sb.WriteString(separator)

	for _, rowValue := range m.values {
		rowFormatArguments := make([]interface{}, 0)
		for column, columnValue := range rowValue {
			rowFormatArguments = append(rowFormatArguments, maxWidths[column])
			rowFormatArguments = append(rowFormatArguments, columnValue)
		}
		row := fmt.Sprintf(valueFormat, rowFormatArguments...)
		sb.WriteString(row)
	}

	return sb.String()
}
