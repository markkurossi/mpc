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

// Documenter implements document output.
type Documenter interface {
	// H1 outputs level 1 heading.
	H1(format string, a ...interface{}) error

	// H2 outputs level 2 heading.
	H2(format string, a ...interface{}) error

	// Pre outputs pre-formatted lines.
	Pre(lines []string) error

	// Type outputs type name.
	Type(name string) error

	// Function outputs function name.
	Function(name string) error

	// Empty outputs empty section content.
	Empty(name string) error
}

// HTMLDoc implements HTML document output.
type HTMLDoc struct {
	out io.Writer
}

// H1 implements Document.H1.
func (doc *HTMLDoc) H1(format string, a ...interface{}) error {
	text := fmt.Sprintf(format, a...)
	_, err := fmt.Fprintf(doc.out, "<h1>%s</h1>\n", html.EscapeString(text))
	return err
}

// H2 implements Document.H2.
func (doc *HTMLDoc) H2(format string, a ...interface{}) error {
	text := fmt.Sprintf(format, a...)
	_, err := fmt.Fprintf(doc.out, "<h2>%s</h2>\n", html.EscapeString(text))
	return err
}

// Pre implements Documenter.Pre.
func (doc *HTMLDoc) Pre(lines []string) error {
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
	for _, l := range lines {
		_, err := fmt.Fprintln(doc.out, html.EscapeString(l))
		if err != nil {
			return err
		}
	}
	return nil
}

// Type implements Documenter.Type.
func (doc *HTMLDoc) Type(name string) error {
	_, err := fmt.Fprintf(doc.out, `<span class="type">%s</span>`,
		html.EscapeString(name))
	return err
}

// Function implements Documenter.Function.
func (doc *HTMLDoc) Function(name string) error {
	_, err := fmt.Fprintf(doc.out, `<span class="functionName">%s</span>`,
		html.EscapeString(name))
	return err
}

// Empty implements Documenter.Empty.
func (doc *HTMLDoc) Empty(name string) error {
	_, err := fmt.Fprintf(doc.out, `<div class="empty">%s</div>`,
		html.EscapeString(name))
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

	fmt.Print(`<!DOCTYPE html>
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

	for _, name := range names {
		doc.H1("Package %s", name)
		if err := documentPackage(doc, packages[name]); err != nil {
			return err
		}
	}
	fmt.Print(`
        </div>
      </div>
    </div>
  </body>
</html>
`)
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

func documentPackage(doc Documenter, pkg *Package) error {
	err := annotations(doc, pkg.Annotations)
	if err != nil {
		return err
	}
	err = doc.H2("Constants")
	if err != nil {
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
		fmt.Printf(`
<div class="code">%s</div>
`, html.EscapeString(c.String()))
		err = annotations(doc, c.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadConstants {
		doc.Empty("This section is empty.")
	}

	err = doc.H2("Variables")
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
		fmt.Printf(`
<div class="code">%s</div>
`, html.EscapeString(v.String()))
		err = annotations(doc, v.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadVariables {
		doc.Empty("This section is empty.")
	}

	err = doc.H2("Functions")
	if err != nil {
		return err
	}
	sort.Slice(pkg.Functions, func(i, j int) bool {
		return pkg.Functions[i].Name < pkg.Functions[j].Name
	})
	for _, f := range pkg.Functions {
		fmt.Printf(`
<div class="signature">func %s</div>
`,
			html.EscapeString(f.Name))
		fmt.Printf(`
<div class="code">%s</div>
`, html.EscapeString(f.String()))
		err = annotations(doc, f.Annotations)
		if err != nil {
			return err
		}
	}

	err = doc.H2("Types")
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
		fmt.Printf(`
<div class="code">%s</div>
`, html.EscapeString(t.Format()))
		err = annotations(doc, t.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadTypes {
		doc.Empty("This section is empty.")
	}

	return nil
}

func annotations(doc Documenter, annotations ast.Annotations) error {
	if len(annotations) == 0 {
		return nil
	}
	prefixLen, _ := wsPrefix(annotations[0])
	var inPre bool
	var preLines []string

	fmt.Println(`<div class="documentation">`)
	for _, ann := range annotations {
		plen, empty := wsPrefix(ann)
		if plen > prefixLen {
			if !inPre {
				fmt.Printf("<pre>")
				inPre = true
			}
			preLines = append(preLines, ann)
		} else if empty {
			if inPre {
				preLines = append(preLines, ann)
			} else {
				fmt.Println("<p>")
			}
		} else {
			if inPre {
				doc.Pre(preLines)
				preLines = nil
				fmt.Println("</pre>")
				inPre = false
			}
			fmt.Println(html.EscapeString(ann))
		}
	}
	if inPre {
		doc.Pre(preLines)
		fmt.Println("</pre>")
	}
	fmt.Println(`</div>`)

	return nil
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
