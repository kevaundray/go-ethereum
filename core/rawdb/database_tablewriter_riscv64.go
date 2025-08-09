// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// TODO: Stub implementation for tablewriter -- likely can move this into a tablewriter package

//go:build riscv64

package rawdb

import (
	"fmt"
	"io"
	"strings"
)

type Table struct {
	out     io.Writer
	headers []string
	footer  []string
	rows    [][]string
}

func newTableWriter(w io.Writer) *Table {
	return &Table{out: w}
}

func (t *Table) SetHeader(headers []string) {
	t.headers = headers
}

func (t *Table) SetFooter(footer []string) {
	t.footer = footer
}

func (t *Table) AppendBulk(rows [][]string) {
	t.rows = rows
}

func (t *Table) Render() {
	// Calculate column widths based on content
	widths := t.calculateWidths()
	separator := t.buildSeparator(widths)

	// Headers
	if len(t.headers) > 0 {
		t.printRow(t.headers, widths)
		fmt.Fprintln(t.out, separator)
	}

	// Data rows
	for _, row := range t.rows {
		t.printRow(row, widths)
	}

	// Footer
	if len(t.footer) > 0 {
		fmt.Fprintln(t.out, separator)
		t.printRow(t.footer, widths)
	}
}

func (t *Table) calculateWidths() []int {
	if len(t.rows) == 0 && len(t.headers) == 0 {
		return nil
	}

	// Start with header widths
	var widths []int
	if len(t.headers) > 0 {
		widths = make([]int, len(t.headers))
		for i, h := range t.headers {
			widths[i] = len(h)
		}
	} else if len(t.rows) > 0 {
		widths = make([]int, len(t.rows[0]))
	}

	// Check all rows for max width
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Check footer widths
	for i, f := range t.footer {
		if i < len(widths) && len(f) > widths[i] {
			widths[i] = len(f)
		}
	}

	// Add some padding
	for i := range widths {
		widths[i] += 2
	}

	return widths
}

func (t *Table) buildSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("-", w)
	}
	return strings.Join(parts, "+")
}

func (t *Table) printRow(row []string, widths []int) {
	for i, cell := range row {
		if i > 0 {
			fmt.Fprint(t.out, "|")
		}
		if i < len(widths) {
			fmt.Fprintf(t.out, " %-*s ", widths[i]-2, cell)
		} else {
			fmt.Fprintf(t.out, " %s ", cell)
		}
	}
	fmt.Fprintln(t.out)
}
