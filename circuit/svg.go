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
	gateWidth  = 32
	gateHeight = 32
	gatePadX   = 16
	gatePadY   = 32
)

type tile struct {
	gate *Gate
	idx  int
	x    float64
	y    float64
}

type point struct {
	x, y float64
}

type wireType int

const (
	wireTypeNormal wireType = iota
	wireTypeZero
	wireTypeOne
)

type wire struct {
	t    wireType
	from point
	to   point
}

func (w *wire) svg(out io.Writer) {
	label := "1"

	switch w.t {
	case wireTypeNormal:
		fmt.Fprintf(out, `<path d="M %v %v C %v %v %v %v %v %v" />
`,
			w.from.x, w.from.y,
			w.from.x, w.from.y+10,
			w.to.x, w.to.y-10,
			w.to.x, w.to.y)

	case wireTypeZero:
		label = "0"
		fallthrough

	case wireTypeOne:
		fmt.Fprintf(out, `    <g fill="#000">
      <text x="%v" y="%v" text-anchor="middle">%v</text>
    </g>
`,
			w.to.x, w.to.y-2, label)
	}
}

type svgCtx struct {
	wireStarts []point
	zero       Wire
	one        Wire
}

func (ctx *svgCtx) setWireType(input Wire, w *wire) {
	if input == ctx.zero {
		w.t = wireTypeZero
	} else if input == ctx.one {
		w.t = wireTypeOne
	}
}

func (c *Circuit) Svg(out io.Writer) {
	c.AssignLevels()

	width := c.Stats[MaxWidth]
	height := c.Stats[NumLevels]
	count := len(c.Gates)

	// Compute level widths.

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

	// Assing tiles.

	tiles := make([]*tile, count)

	x = 0
	y = -1
	lastLevel = 0
	var leftPad int

	for idx, g := range c.Gates {
		if idx == 0 || g.Level != lastLevel {
			lastLevel = g.Level
			y++
			x = 0

			leftPad = int(float64(width-widths[y]) / 2 * (gateWidth + gatePadX))
		}
		tiles[idx] = &tile{
			gate: &c.Gates[idx],
			idx:  idx,
			x:    float64(leftPad + x*(gateWidth+gatePadX)),
			y:    float64(y * (gateHeight + gatePadY)),
		}
		x++
	}

	// Render circuit.

	ctx := &svgCtx{
		wireStarts: make([]point, c.NumWires),
	}

	fmt.Fprintf(out,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">
  <style><![CDATA[
  text {
    font: 10px Verdana, Helvetica, Arial, sans-serif;
  }
  ]]></style>
  <g fill="none" stroke="#000" stroke-width=".5">
`,
		width*(gateWidth+gatePadX), height*(gateHeight+gatePadY))

	var wires []*wire

	for _, tile := range tiles {
		wires = append(wires, tile.gate.svg(out, tile.x, tile.y, ctx)...)
	}

	for _, w := range wires {
		w.svg(out)
	}

	fmt.Fprintln(out, "  </g>\n</svg>")
}

func (g *Gate) svg(out io.Writer, x, y float64, ctx *svgCtx) []*wire {
	fmt.Fprintf(out, `    <g transform="translate(%v %v)">
`,
		x, y)

	tmpl := templates[g.Op]
	if tmpl == nil {
		tmpl = templates[Count]
	}
	out.Write([]byte(tmpl.Expand()))
	fmt.Fprintln(out, `    </g>`)

	ctx.wireStarts[g.Output] = point{
		x: x + gateWidth/2,
		y: y + gateHeight,
	}

	var wires []*wire

	switch g.Op {
	case XOR, XNOR, AND, OR:
		wire := &wire{
			from: ctx.wireStarts[g.Input1],
			to: point{
				x: x + intCvt(65),
				y: y,
			},
		}
		ctx.setWireType(g.Input1, wire)
		wires = append(wires, wire)
		fallthrough

	case INV:
		wire := &wire{
			from: ctx.wireStarts[g.Input0],
			to: point{
				x: x + intCvt(35),
				y: y,
			},
		}
		ctx.setWireType(g.Input0, wire)
		wires = append(wires, wire)
	}

	// Constant value gates.
	switch g.Op {
	case XOR:
		if g.Input0 == g.Input1 {
			ctx.zero = g.Output
			g.staticOutput(out, x, y, "0")
		}

	case XNOR:
		if g.Input0 == g.Input1 {
			ctx.one = g.Output
			g.staticOutput(out, x, y, "1")
		}
	}

	return wires
}

func (g *Gate) staticOutput(out io.Writer, x, y float64, label string) {
	fmt.Fprintf(out, `    <g fill="#000">
      <text x="%v" y="%v" text-anchor="middle">%v</text>
    </g>
`,
		x+gateWidth/2, y+gateHeight+10, label)
}

func scale(in int) float64 {
	return float64(in) * gateWidth / 100
}

func path(out io.Writer) {
	fmt.Fprintln(out, `  <path fill="none" stroke="#000" stroke-width=".5"`)
}

var templates [Count + 1]*Template

var intCvt IntCvt
var floatCvt FloatCvt

func init() {
	templates[XOR] = NewTemplate(`      <path
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

	floatCvt = func(v float64) float64 {
		return v * gateWidth / 100
	}
	intCvt = func(v int) float64 {
		return float64(v) * gateWidth / 100
	}
	for op := XOR; op < Count+1; op++ {
		if templates[op] != nil {
			templates[op].FloatCvt = floatCvt
			templates[op].IntCvt = intCvt
		}
	}
}
