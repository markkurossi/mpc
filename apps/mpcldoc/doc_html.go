//
// doc.go
//
// Copyright (c) 2021-2022 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/markkurossi/text"
)

// HTMLDoc implements HTML document generator
type HTMLDoc struct {
	dir string
}

// NewHTMLDoc creates a new HTML documenter. The HTML documentation is
// created in the argument directory. The function will create the
// directory if it does not exist.
func NewHTMLDoc(dir string) (*HTMLDoc, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	return &HTMLDoc{
		dir: dir,
	}, nil
}

// New implements Documenter.New.
func (doc *HTMLDoc) New(name string) (Output, error) {
	file := path.Join(doc.dir, fmt.Sprintf("pkg_%s.html", name))
	f, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	err = header(f)
	if err != nil {
		f.Close()
		os.Remove(file)
		return nil, err
	}

	return &HTMLOutput{
		out: f,
	}, nil
}

// Index implements Documenter.Index.
func (doc *HTMLDoc) Index(pkgs []*Package) error {
	file := path.Join(doc.dir, "apidoc.html")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = index(f, pkgs)
	if err != nil {
		os.Remove(file)
		return err
	}
	return nil
}

func index(out io.Writer, pkgs []*Package) error {
	err := header(out)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, `
<h1>MPCL API Documentation</h1>
<h2>Packages</h2>
`)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		first := pkg.Annotations.FirstSentence()
		_, err := fmt.Fprintf(out, `<div class="index">
<a href="%s">Package %s</a>
</div>
`,
			fmt.Sprintf("pkg_%s.html", pkg.Name), pkg.Name)
		if err != nil {
			return err
		}
		if len(first) > 0 {
			_, err = fmt.Fprintf(out, "<p>%s</p>\n", first)
			if err != nil {
				return err
			}
		}
	}

	err = trailer(out)
	if err != nil {
		return err
	}

	return nil
}

// HTMLOutput implements HTML document output.
type HTMLOutput struct {
	out io.WriteCloser
}

// Close implements Output.Close.
func (out *HTMLOutput) Close() error {
	err := trailer(out.out)
	if err != nil {
		return err
	}
	return out.out.Close()
}

// H1 implements Output.H1.
func (out *HTMLOutput) H1(content *text.Text) error {
	_, err := fmt.Fprintf(out.out, "<h1>%s</h1>\n\n", content.HTML())
	return err
}

// H2 implements Output.H2.
func (out *HTMLOutput) H2(content *text.Text) error {
	_, err := fmt.Fprintf(out.out, "<h2>%s</h2>\n\n", content.HTML())
	return err
}

// Pre implements Output.Pre.
func (out *HTMLOutput) Pre(lines []*text.Text) error {
	return out.lines(lines, "pre")
}

// P implements Output.P.
func (out *HTMLOutput) P(lines []*text.Text) error {
	return out.lines(lines, "p")
}

func (out *HTMLOutput) lines(lines []*text.Text, tag string) error {
	var formatted []string
	for _, line := range lines {
		formatted = append(formatted, line.HTML())
	}

	// Trim leading empty lines.
	for i := 0; i < len(formatted); i++ {
		if len(strings.TrimSpace(formatted[i])) > 0 {
			formatted = formatted[i:]
			break
		}
	}
	// Trim trailing empty lines.
	for i := len(formatted) - 1; i >= 0; i-- {
		if len(strings.TrimSpace(formatted[i])) > 0 {
			formatted = formatted[0 : i+1]
			break
		}
	}
	_, err := fmt.Fprintf(out.out, "<%s>", tag)
	if err != nil {
		return err
	}
	for _, l := range formatted {
		_, err := fmt.Fprintln(out.out, l)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(out.out, "</%s>\n", tag)
	if err != nil {
		return err
	}
	return nil
}

// Type implements Output.Type.
func (out *HTMLOutput) Type(name *text.Text) error {
	_, err := fmt.Fprintf(out.out, `<span class="type">%s</span>`, name.HTML())
	return err
}

// Function implements Output.Function.
func (out *HTMLOutput) Function(name *text.Text) error {
	_, err := fmt.Fprintf(out.out, `<span class="functionName">%s</span>`,
		name.HTML())
	return err
}

// Empty implements Output.Empty.
func (out *HTMLOutput) Empty(name *text.Text) error {
	_, err := fmt.Fprintf(out.out, `<div class="empty">%s</div>`, name.HTML())
	return err
}

// Code implements Output.Code.
func (out *HTMLOutput) Code(id string, code *text.Text) error {
	var idLabel string
	if len(id) > 0 {
		idLabel = fmt.Sprintf("id=\"%s\" ", id)
	}
	_, err := fmt.Fprintf(out.out, `
<div %sclass="code">%s</div>
`,
		idLabel, code.HTML())
	return err
}

// Signature implements Output.Signature.
func (out *HTMLOutput) Signature(code *text.Text) error {
	_, err := fmt.Fprintf(out.out, `
<div class="signature">func %s</div>
`,
		code.HTML())

	return err
}

// Start implements Output.Start.
func (out *HTMLOutput) Start(section string) error {
	_, err := fmt.Fprintf(out.out, `<div class="%s">`, section)
	return err
}

// End implements Output.End.
func (out *HTMLOutput) End(section string) error {
	_, err := fmt.Fprintln(out.out, `</div>`)
	return err
}

// URL implements Output.URL.
func (out *HTMLOutput) URL(container, id string) string {
	return fmt.Sprintf("pkg_%s.html#%s", container, id)
}

func header(out io.Writer) error {
	_, err := fmt.Fprintf(out, `<!DOCTYPE html>
<html lang="en">
  <head>
    <link rel="icon" href="favicon.png" />
    <meta http-equiv="content-type" content="text/html;charset=UTF-8" />
    <meta name="viewport" content="width=device-width">

    <title>MPCL</title>
    <link href="woff/stylesheet.css" rel="stylesheet" type="text/css">
    <link href="index.css" rel="stylesheet" type="text/css">
  </head>
  <body>
    <div class="page-wrapper">
      <div class="row">
        <div class="left-column">
          <div style="text-align: center; display: inline-block;
                      padding: 10px;">
            <img src="mpcl.png" width="64" align="middle"><br>MPCL
          </div>
          <ul>
            <li><a href="index.html">MPCL Documentation</a>
            <li><a href="https://github.com/markkurossi/mpc">GitHub</a>
          </ul>
        </div>
        <div class="article-column">
`)
	return err
}

func trailer(out io.Writer) error {
	_, err := fmt.Fprint(out, `
        </div>
      </div>
    </div>
  </body>
</html>
`)
	return err
}
