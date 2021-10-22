//
// doc.go
//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"strings"
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
	file := path.Join(doc.dir, "index.html")
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
<h1>Multi-Party Computation Language (MPCL)</h1>
<h2>Packages</h2>
`)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		_, err := fmt.Fprintf(out, `<div class="index">
<a href="%s">Package %s</a>
</div>
`,
			fmt.Sprintf("pkg_%s.html", pkg.Name), pkg.Name)
		if err != nil {
			return err
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
func (out *HTMLOutput) H1(format string, a ...interface{}) error {
	text := fmt.Sprintf(format, a...)
	_, err := fmt.Fprintf(out.out, "<h1>%s</h1>\n", html.EscapeString(text))
	return err
}

// H2 implements Output.H2.
func (out *HTMLOutput) H2(format string, a ...interface{}) error {
	text := fmt.Sprintf(format, a...)
	_, err := fmt.Fprintf(out.out, "<h2>%s</h2>\n", html.EscapeString(text))
	return err
}

// Pre implements Output.Pre.
func (out *HTMLOutput) Pre(lines []string) error {
	for i := 0; i < len(lines); i++ {
		if len(strings.TrimSpace(lines[i])) > 0 {
			lines = lines[i:]
			break
		}
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if len(strings.TrimSpace(lines[i])) > 0 {
			lines = lines[0 : i+1]
			break
		}
	}
	_, err := fmt.Fprintf(out.out, "<pre>")
	if err != nil {
		return err
	}
	for _, l := range lines {
		_, err := fmt.Fprintln(out.out, html.EscapeString(l))
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(out.out, "</pre>")
	if err != nil {
		return err
	}
	return nil
}

// P implements Output.P.
func (out *HTMLOutput) P(lines []string) error {
	for i := 0; i < len(lines); i++ {
		if len(strings.TrimSpace(lines[i])) > 0 {
			lines = lines[i:]
			break
		}
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if len(strings.TrimSpace(lines[i])) > 0 {
			lines = lines[0 : i+1]
			break
		}
	}
	_, err := fmt.Fprintf(out.out, "<p>")
	if err != nil {
		return err
	}
	for _, l := range lines {
		_, err := fmt.Fprintln(out.out, html.EscapeString(l))
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(out.out, "</p>")
	if err != nil {
		return err
	}
	return nil
}

// Type implements Output.Type.
func (out *HTMLOutput) Type(name string) error {
	_, err := fmt.Fprintf(out.out, `<span class="type">%s</span>`,
		html.EscapeString(name))
	return err
}

// Function implements Output.Function.
func (out *HTMLOutput) Function(name string) error {
	_, err := fmt.Fprintf(out.out, `<span class="functionName">%s</span>`,
		html.EscapeString(name))
	return err
}

// Empty implements Output.Empty.
func (out *HTMLOutput) Empty(name string) error {
	_, err := fmt.Fprintf(out.out, `<div class="empty">%s</div>`,
		html.EscapeString(name))
	return err
}

// Code implements Output.Code.
func (out *HTMLOutput) Code(code string) error {
	_, err := fmt.Fprintf(out.out, `
<div class="code">%s</div>
`, html.EscapeString(code))
	return err
}

// Signature implements Output.Signature.
func (out *HTMLOutput) Signature(code string) error {
	_, err := fmt.Fprintf(out.out, `
<div class="signature">func %s</div>
`,
		html.EscapeString(code))

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

func header(out io.Writer) error {
	_, err := fmt.Fprintf(out, `<!DOCTYPE html>
<html>
  <head>
    <link rel="icon" href="favicon.png" />
    <meta http-equiv="content-type" content="text/html;charset=UTF-8" />
    <meta name="viewport" content="width=device-width">

    <title>MPCL</title>
    <link href="index.css" rel="stylesheet" type="text/css">
  </head>
  <body>
    <div class="page-wrapper">
      <div class="row">
        <div class="left-column">
          <div style="text-align: center; display: inline-block;
                      padding: 10px;">
            <img src="logo.png" width="64" align="middle"><br>MPCL
          </div>
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
