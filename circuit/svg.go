//
// Copyright (c) 2019-2022 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"io"
)

const (
	gateWidth  = 25
	gateHeight = 25
)

func (c *Circuit) Svg(out io.Writer) {
	c.AssignLevels()

	width := c.Stats[MaxWidth]
	height := c.Stats[NumLevels]

	widths := make([]uint64, height)

	var lastLevel Level
	var x, y int

	for _, g := range c.Gates {
		if g.Level != lastLevel {
			lastLevel = g.Level
			y++
		}
		widths[y]++
	}

	fmt.Fprintf(out,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">
  <g fill="none" stroke="#000" stroke-width=".5">
`,
		width*gateWidth, height*gateHeight)

	x = int((width - widths[0]) / 2)
	y = 0
	lastLevel = 0

	for _, g := range c.Gates {
		if g.Level != lastLevel {
			lastLevel = g.Level
			y++
			x = int((width - widths[y]) / 2)
		}
		g.Svg(out, x*gateWidth, y*gateHeight)
		x++
	}

	fmt.Fprintln(out, "  </g>\n</svg>")
}

func (g *Gate) Svg(out io.Writer, x, y int) {
	fmt.Fprintf(out, `<g transform="translate(%d %d)">
`,
		x, y)

	tmpl := templates[g.Op]
	if tmpl == nil {
		tmpl = templates[Count]
	}
	out.Write([]byte(tmpl.Expand()))
	fmt.Fprintln(out, "</g>")
}

func scale(in int) float64 {
	return float64(in) * gateWidth / 100
}

func path(out io.Writer) {
	fmt.Fprintln(out, `  <path fill="none" stroke="#000" stroke-width=".5"`)
}

var templates [Count + 1]*Template

func init() {
	templates[XOR] = NewTemplate(`<path
        d="M {{25}} {{20}}
           c {{10}} {{10}} {{40}} {{10}} {{50}} 0" />
  <path
        d="M {{25}} {{25}}
           c {{10}} {{10}} {{40}} {{10}} {{50}} 0" />

  <path
        d="M {{75}} {{25}}
           v {{25}}
           s 0 {{10}} {{-25}} {{25}}" />
  <path
        d="M {{25}} {{25}}
           v {{25}}
           s 0 {{10}} {{25}} {{25}}" />

  <!-- Wires -->
  <path
        d="M {{35}} 0
           v {{25}}
           z" />
  <path
        d="M {{65}} 0
           v {{25}}
           z" />
  <path
        d="M {{50}} {{75}}
           v {{25}}
           z" />
`)
	templates[XNOR] = NewTemplate(`    <path
          d="M {{25}} {{20}}
             c {{10}} {{10}} {{40}} {{10}} {{50}} 0" />
    <path
          d="M {{25}} {{25}}
             c {{10}} {{10}} {{40}} {{10}} {{50}} 0" />

    <path
          d="M {{75}} {{25}}
             v {{25}}
             s 0 {{10}} {{-25}} {{25}}" />
    <path
          d="M {{25}} {{25}}
             v {{25}}
             s 0 {{10}} {{25}} {{25}}" />

    <circle
            cx="{{50}}"
            cy="{{80}}"
            r="{{5}}" />

    <path
          d="M {{35}} 0
             v {{25}}
             z" />
    <path
          d="M {{65}} 0
             v {{25}}
             z" />
    <path
          d="M {{50}} {{85}}
             v {{15}}
             z" />
`)

	templates[AND] = NewTemplate(`    <path
          d="M {{25}} {{25}}
             h {{50}}
             v {{25}}
             a {{25}} {{25}} 0 1 1 {{-50}} 0
             v {{-25}}
             z" />
    <path d="M {{35}} 0
             v {{25}}
             z" />
    <path d="M {{65}} 0
             v {{25}}
             z" />
    <path d="M {{50}} {{75}}
             v {{25}}
             z" />
`)

	templates[OR] = NewTemplate(`    <path
          d="M {{25}} {{20}}
             c {{10}} {{10}} {{40}} {{10}} {{50}} 0" />

    <path
          d="M {{75}} {{20}}
             v {{30}}
             s 0 {{10}} {{-25}} {{25}}" />
    <path
          d="M {{25}} {{20}}
             v {{30}}
             s 0 {{10}} {{25}} {{25}}" />

    <path
          d="M {{35}} 0
             v {{25}}
             z" />
    <path
          d="M {{65}} 0
             v {{25}}
             z" />
    <path
          d="M {{50}} {{75}}
             v {{25}}
             z" />
`)

	templates[INV] = NewTemplate(`
    <path d="M {{25}} {{25}}
             h {{50}}
             l {{-25}} {{43}}
             z" />
    <circle cx="{{50}}"
            cy="{{73.5}}"
            r="{{5}}" />
    <path d="M {{35}} 0
             v {{25}}
             z" />
    <path d="M {{65}} 0
             v {{25}}
             z" />
    <path d="M {{50}} {{79}}
             v {{21}}
             z" />
`)

	templates[Count] = NewTemplate(`<path
        d="M {{25}} {{25}} h {{50}} v{{50}} h {{-50}} z" />`)

	floatCvt := func(v float64) interface{} {
		return v * gateWidth / 100
	}
	intCvt := func(v int) interface{} {
		return float64(v) * gateWidth / 100
	}
	for op := XOR; op < Count+1; op++ {
		if templates[op] != nil {
			templates[op].FloatCvt = floatCvt
			templates[op].IntCvt = intCvt
		}
	}
}
