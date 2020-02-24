//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"io"
)

type Border struct {
	H  string
	V  string
	TL string
	TM string
	TR string
	ML string
	MM string
	MR string
	BL string
	BM string
	BR string
}

var WhiteSpace = Border{}

var ASCII = Border{
	H:  "-",
	V:  "|",
	TL: "+",
	TM: "+",
	TR: "+",
	ML: "+",
	MM: "+",
	MR: "+",
	BL: "+",
	BM: "+",
	BR: "+",
}

var Unicode = Border{
	H:  "\u2501",
	V:  "\u2503",
	TL: "\u250F",
	TM: "\u2533",
	TR: "\u2513",
	ML: "\u2523",
	MM: "\u254B",
	MR: "\u252B",
	BL: "\u2517",
	BM: "\u253B",
	BR: "\u251B",
}

type Tabulate struct {
	Padding int
	Border  Border
	Headers []Column
	Rows    []*Row
}

func NewTabulateWS() *Tabulate {
	return &Tabulate{
		Padding: 2,
		Border:  WhiteSpace,
	}
}

func NewTabulateASCII() *Tabulate {
	return &Tabulate{
		Padding: 2,
		Border:  ASCII,
	}
}

func NewTabulateUnicode() *Tabulate {
	return &Tabulate{
		Padding: 2,
		Border:  Unicode,
	}
}

func (t *Tabulate) Header(align Align, data string) *Tabulate {
	t.Headers = append(t.Headers, Column{
		Align: align,
		Data:  data,
	})
	return t
}

func (t *Tabulate) Row() *Row {
	row := &Row{
		Tab: t,
	}
	t.Rows = append(t.Rows, row)
	return row
}

func (t *Tabulate) Print(o io.Writer) {
	widths := make([]int, len(t.Headers))

	for idx, hdr := range t.Headers {
		if len([]rune(hdr.Data)) > widths[idx] {
			widths[idx] = len([]rune(hdr.Data))
		}
	}
	for _, row := range t.Rows {
		for idx, col := range row.Columns {
			if idx >= len(widths) {
				widths = append(widths, 0)
			}
			if len([]rune(col.Data)) > widths[idx] {
				widths[idx] = len([]rune(col.Data))
			}
		}
	}

	// Header.
	if len(t.Border.H) > 0 {
		fmt.Fprint(o, t.Border.TL)
		for idx, width := range widths {
			for i := 0; i < width+t.Padding; i++ {
				fmt.Fprint(o, t.Border.H)
			}
			if idx+1 < len(widths) {
				fmt.Fprint(o, t.Border.TM)
			} else {
				fmt.Fprintln(o, t.Border.TR)
			}
		}
	}

	for idx, width := range widths {
		var hdr Column
		if idx < len(t.Headers) {
			hdr = t.Headers[idx]
		}
		t.PrintColumn(o, hdr, width)
	}
	fmt.Fprintln(o, t.Border.V)

	if len(t.Border.H) > 0 {
		fmt.Fprint(o, t.Border.ML)
		for idx, width := range widths {
			for i := 0; i < width+t.Padding; i++ {
				fmt.Fprint(o, t.Border.H)
			}
			if idx+1 < len(widths) {
				fmt.Fprint(o, t.Border.MM)
			} else {
				fmt.Fprintln(o, t.Border.MR)
			}
		}
	}

	// Data rows.
	for _, row := range t.Rows {
		for idx, width := range widths {
			var col Column
			if idx < len(row.Columns) {
				col = row.Columns[idx]
			}
			t.PrintColumn(o, col, width)
		}
		fmt.Fprintln(o, t.Border.V)
	}

	if len(t.Border.H) > 0 {
		fmt.Fprint(o, t.Border.BL)
		for idx, width := range widths {
			for i := 0; i < width+t.Padding; i++ {
				fmt.Fprint(o, t.Border.H)
			}
			if idx+1 < len(widths) {
				fmt.Fprint(o, t.Border.BM)
			} else {
				fmt.Fprintln(o, t.Border.BR)
			}
		}
	}
}

func (t *Tabulate) PrintColumn(o io.Writer, col Column, width int) {

	lPad := t.Padding / 2
	rPad := t.Padding - lPad

	pad := width - len([]rune(col.Data))
	switch col.Align {
	case AlignLeft:
		rPad += pad

	case AlignCenter:
		l := pad / 2
		r := pad - l
		lPad += l
		rPad += r

	case AlignRight:
		lPad += pad
	}

	fmt.Fprint(o, t.Border.V)
	for i := 0; i < lPad; i++ {
		fmt.Fprint(o, " ")
	}
	if col.Format != FmtNone {
		fmt.Fprint(o, col.Format.VT100())
	}
	fmt.Fprint(o, col.Data)
	if col.Format != FmtNone {
		fmt.Fprint(o, FmtNone.VT100())
	}
	for i := 0; i < rPad; i++ {
		fmt.Fprint(o, " ")
	}
}

type Row struct {
	Tab     *Tabulate
	Columns []Column
}

func (r *Row) Column(data string) {
	idx := len(r.Columns)
	var hdr Column
	if idx < len(r.Tab.Headers) {
		hdr = r.Tab.Headers[idx]
	}

	r.Columns = append(r.Columns, Column{
		Align:  hdr.Align,
		Data:   data,
		Format: hdr.Format,
	})
}

func (r *Row) ColumnAttrs(align Align, data string, format Format) {
	r.Columns = append(r.Columns, Column{
		Align:  align,
		Data:   data,
		Format: format,
	})
}

type Column struct {
	Align  Align
	Data   string
	Format Format
}

type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

type Format int

const (
	FmtNone Format = iota
	FmtBold
	FmtItalic
)

func (fmt Format) VT100() string {
	switch fmt {
	case FmtBold:
		return "\x1b[1m"
	case FmtItalic:
		return "\x1b[3m"
	default:
		return "\x1b[m"
	}
}
