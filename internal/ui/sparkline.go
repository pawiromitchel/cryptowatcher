package ui

import (
	"fmt"
	"math"
	"strings"
)

var sparklineBlocks = []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline renders a 1-line ASCII mini price trend chart.
func RenderSparkline(data []float64, isPositive bool) string {
	if len(data) == 0 {
		return ""
	}

	minVal := data[0]
	maxVal := data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	var sb strings.Builder
	diff := maxVal - minVal

	for _, v := range data {
		var idx int
		if diff == 0 {
			idx = 3
		} else {
			norm := (v - minVal) / diff
			idx = int(math.Floor(norm * float64(len(sparklineBlocks)-1)))
			if idx >= len(sparklineBlocks) {
				idx = len(sparklineBlocks) - 1
			}
			if idx < 0 {
				idx = 0
			}
		}
		sb.WriteRune(sparklineBlocks[idx])
	}

	chartStr := sb.String()
	if isPositive {
		return positiveStyle.Render(chartStr)
	}
	return negativeStyle.Render(chartStr)
}

// RenderRangeBar generates a visual 24-hour price range slider between Low and High.
func RenderRangeBar(current, low, high float64, barWidth int) string {
	if high <= low || barWidth <= 2 {
		return "[--------------------]"
	}

	ratio := (current - low) / (high - low)
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}

	filledLen := int(math.Round(ratio * float64(barWidth)))
	if filledLen > barWidth {
		filledLen = barWidth
	}

	filled := strings.Repeat("█", filledLen)
	unfilled := strings.Repeat("░", barWidth-filledLen)

	bar := fmt.Sprintf("[%s%s]", filled, unfilled)

	if ratio >= 0.5 {
		return positiveStyle.Render(bar)
	}
	return negativeStyle.Render(bar)
}
