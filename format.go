package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

func cmpOne(x, y string) int {
	switch {
	case x == y:
		return 0

	case y == "Incomplete":
		return -1
	case x == "Incomplete":
		return 1
	case y == "Inv" || y == "Invalid":
		return -1
	case x == "Inv" || x == "Invalid":
		return 1
	case y == "Abs" || y == "Abstain":
		return -1
	case x == "Abs" || x == "Abstain":
		return 1
	case y == "Exhausted":
		return -1
	case x == "Exhausted":
		return 1
	case y == "Write-in":
		return -1
	case x == "Write-in":
		return 1

	case x == "Yes":
		return -1
	case y == "Yes":
		return 1
	case x == "No":
		return -1
	case y == "No":
		return 1

	case x < y:
		return -1
	default:
		return 1
	}
}

func less(x, y string) bool {
	if x == y {
		return false
	}
	xw := strings.Split(x, "|")
	yw := strings.Split(y, "|")
	xw = nonempty(xw)
	yw = nonempty(yw)
	return slices.CompareFunc(xw, yw, cmpOne) == -1
}

func formatResults[T numeric](results map[string]T) string {
	keys := make([]string, 0, len(results))
	total := T(0)
	w := len("Total")
	for k, v := range results {
		keys = append(keys, k)
		total += v
		if w < len(k) {
			w = len(k)
		}
	}
	slices.SortFunc(keys, less)

	f := "%" + strconv.Itoa(w) + "v"
	lines := make([]string, len(results)+1)
	for i, k := range keys {
		lines[i] = fmt.Sprintf(f+": %7v (%4.1f%%)", k, int(results[k]), float64(100*results[k])/float64(total))
	}
	lines[len(results)] = fmt.Sprintf(f+": %7v", "Total", int(total))
	return strings.Join(lines, "\n") + "\n"
}

func formatCSV(results map[string]int) string {
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, less)
	cols := len(nonempty(strings.Split(keys[0], "|"))) + 1

	cells := make([][]string, len(results))
	for i, k := range keys {
		cells[i] = make([]string, cols)
		copy(cells[i], nonempty(strings.Split(k, "|")))
		// (there will be a gap between these for incomplete)
		cells[i][cols-1] = strconv.Itoa(results[k])
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := w.WriteAll(cells)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func formatGrid[T any](grid [][]T) string {
	strings := map1(func(row []T) []string {
		return map1(func(cell T) string {
			if interface{}(cell) == nil {
				return ""
			}
			return fmt.Sprint(cell)
		}, row)
	}, grid)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := w.WriteAll(strings)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

type color [3]uint8

var (
	green = color{87, 187, 138}
	white = color{255, 255, 255}
)

func formatGridHTML[T any](grid [][]T) string {
	var maxVal float64
	for _, row := range grid {
		for _, cell := range row {
			if v, ok := any(cell).(float64); ok {
				if v > maxVal {
					maxVal = v
				}
			}
		}
	}

	var b bytes.Buffer
	b.WriteString("<table>\n<thead>\n")
	inHead := true
	for _, row := range grid {
		// assume strings are headers
		if inHead {
			_, isHead := any(row[len(row)-1]).(string)
			if !isHead {
				b.WriteString("</thead>\n<tbody>\n")
				inHead = false
			}
		}
		b.WriteString("<tr>")
		for _, cell := range row {
			switch cell := any(cell).(type) {
			case nil:
				b.WriteString("<td/>")
			case string:
				// don't html inject me SFDOE!
				fmt.Fprintf(&b, "<th>%s</th>", cell)
			case float64:
				var c color
				f := cell / maxVal
				for i := 0; i < 3; i++ {
					c[i] = uint8(f*float64(green[i]) + (1-f)*float64(white[i]))
				}
				fmt.Fprintf(&b,
					`<td style="background-color: #%x%x%x;">%.2f%%</td>`,
					c[0], c[1], c[2], 100*cell)
			default:
				fmt.Fprintf(&b, "<td>%v</td>", cell)
			}
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	return b.String()
}
