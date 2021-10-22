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
	"sort"
	"strings"
	"unicode"

	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/utils"
)

var (
	_ Documenter = &HTMLDoc{}
	_ Output     = &HTMLOutput{}
)

// Documenter implements document generator.
type Documenter interface {
	// New creates new output for the named document.
	New(name string) (Output, error)
}

// Output implements document output
type Output interface {
	// Close closes the output.
	Close() error

	// H1 outputs level 1 heading.
	H1(format string, a ...interface{}) error

	// H2 outputs level 2 heading.
	H2(format string, a ...interface{}) error

	// Pre outputs pre-formatted lines.
	Pre(lines []string) error

	// P outputs paragraph lines.
	P(lines []string) error

	// Type outputs type name.
	Type(name string) error

	// Function outputs function name.
	Function(name string) error

	// Empty outputs empty section content.
	Empty(name string) error

	// Code outputs program code.
	Code(code string) error

	// Signature outputs function signature.
	Signature(code string) error

	// Start starts a logical documentation section.
	Start(section string) error

	// End ends a logical documentation section.
	End(section string) error
}

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
	name = fmt.Sprintf("pkg_%s.html", name)
	file := path.Join(doc.dir, name)
	f, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(f, `<!DOCTYPE html>
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
	if err != nil {
		f.Close()
		os.Remove(file)
		return nil, err
	}

	return &HTMLOutput{
		out: f,
	}, nil
}

// HTMLOutput implements HTML document output.
type HTMLOutput struct {
	out io.WriteCloser
}

// Close implements Output.Close.
func (out *HTMLOutput) Close() error {
	_, err := fmt.Fprint(out.out, `
        </div>
      </div>
    </div>
  </body>
</html>
`)
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

var packages = make(map[string]*Package)

func documentation(files []string, doc Documenter) error {
	err := parseInputs(files)
	if err != nil {
		return err
	}
	var names []string
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		out, err := doc.New(name)
		if err != nil {
			return err
		}
		if err := documentPackage(out, name, packages[name]); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

func parseInputs(files []string) error {
	// Parse inputs.
	for _, file := range files {
		fi, err := os.Stat(file)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			entries, err := os.ReadDir(file)
			if err != nil {
				return err
			}
			var files []string
			for _, e := range entries {
				files = append(files, path.Join(file, e.Name()))
			}
			err = parseInputs(files)
			if err != nil {
				return err
			}
		} else {
			err = parseFile(file)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseFile(name string) error {
	if !strings.HasSuffix(name, ".mpcl") {
		return nil
	}

	params := utils.NewParams()
	params.NoCircCompile = true

	pkg, err := compiler.New(params).ParseFile(name)
	if err != nil {
		return err
	}
	p, ok := packages[pkg.Name]
	if !ok {
		p = &Package{}
		packages[pkg.Name] = p
	}
	p.Annotations = append(p.Annotations, pkg.Annotations...)
	p.Constants = append(p.Constants, pkg.Constants...)
	p.Variables = append(p.Variables, pkg.Variables...)
	p.Types = append(p.Types, pkg.Types...)

	for _, v := range pkg.Functions {
		p.Functions = append(p.Functions, v)
	}

	return nil
}

func documentPackage(out Output, name string, pkg *Package) error {
	if err := out.H1("Package %s", name); err != nil {
		return nil
	}
	err := annotations(out, pkg.Annotations)
	if err != nil {
		return err
	}
	if err := out.H2("Constants"); err != nil {
		return err
	}
	sort.Slice(pkg.Constants, func(i, j int) bool {
		return pkg.Constants[i].Name < pkg.Constants[j].Name
	})
	var hadConstants bool
	for _, c := range pkg.Constants {
		if !c.Exported() {
			continue
		}
		hadConstants = true
		if err := out.Code(c.String()); err != nil {
			return err
		}
		err = annotations(out, c.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadConstants {
		out.Empty("This section is empty.")
	}

	err = out.H2("Variables")
	if err != nil {
		return err
	}
	sort.Slice(pkg.Variables, func(i, j int) bool {
		return pkg.Variables[i].Names[0] < pkg.Variables[j].Names[0]
	})
	var hadVariables bool
	for _, v := range pkg.Variables {
		if !ast.IsExported(v.Names[0]) {
			continue
		}
		hadVariables = true
		if err := out.Code(v.String()); err != nil {
			return err
		}
		err = annotations(out, v.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadVariables {
		out.Empty("This section is empty.")
	}

	err = out.H2("Functions")
	if err != nil {
		return err
	}
	sort.Slice(pkg.Functions, func(i, j int) bool {
		return pkg.Functions[i].Name < pkg.Functions[j].Name
	})
	for _, f := range pkg.Functions {
		if err := out.Signature(f.Name); err != nil {
			return err
		}
		if err := out.Code(f.String()); err != nil {
			return err
		}
		err = annotations(out, f.Annotations)
		if err != nil {
			return err
		}
	}

	err = out.H2("Types")
	if err != nil {
		return err
	}
	sort.Slice(pkg.Types, func(i, j int) bool {
		return pkg.Types[i].TypeName < pkg.Types[j].TypeName
	})
	var hadTypes bool
	for _, t := range pkg.Types {
		if !ast.IsExported(t.TypeName) {
			continue
		}
		hadTypes = true
		if err := out.Code(t.Format()); err != nil {
			return err
		}
		err = annotations(out, t.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadTypes {
		out.Empty("This section is empty.")
	}

	return nil
}

func annotations(out Output, annotations ast.Annotations) error {
	if len(annotations) == 0 {
		return nil
	}
	prefixLen, _ := wsPrefix(annotations[0])
	var inPre bool
	var lines []string

	if err := out.Start("documentation"); err != nil {
		return err
	}
	for _, ann := range annotations {
		plen, empty := wsPrefix(ann)
		if plen > prefixLen {
			if !inPre {
				out.P(lines)
				lines = nil
				inPre = true
			}
			lines = append(lines, ann)
		} else if empty {
			if inPre {
				lines = append(lines, ann)
			} else {
				out.P(lines)
				lines = nil
			}
		} else {
			if inPre {
				out.Pre(lines)
				lines = nil
				inPre = false
			}
			lines = append(lines, ann)
		}
	}
	if inPre {
		out.Pre(lines)
	} else {
		out.P(lines)
	}
	return out.End("documentation")
}

func wsPrefix(str string) (int, bool) {
	runes := []rune(str)
	for idx, r := range runes {
		if !unicode.IsSpace(r) {
			return idx, false
		}
	}
	return len(runes), true
}

// Package contains all documentation items for a MPCL package.
type Package struct {
	Annotations ast.Annotations
	Constants   []*ast.ConstantDef
	Variables   []*ast.VariableDef
	Functions   []*ast.Func
	Types       []*ast.TypeInfo
}
