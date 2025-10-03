//
// doc.go
//
// Copyright (c) 2021-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"

	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
	"github.com/markkurossi/text"
)

var (
	_ Documenter = &HTMLDoc{}
	_ Output     = &HTMLOutput{}

	emptySection = "This section is empty."
)

// Documenter implements document generator.
type Documenter interface {
	// New creates new output for the named document.
	New(name string) (Output, error)

	// Index creates an index page for the documentation.
	Index(pkgs, mains []*Package) error
}

// Output implements document output
type Output interface {
	// Close closes the output.
	Close() error

	// H1 outputs level 1 heading.
	H1(text *text.Text) error

	// H2 outputs level 2 heading.
	H2(text *text.Text) error

	// Pre outputs pre-formatted lines.
	Pre(lines []*text.Text) error

	// P outputs paragraph lines.
	P(lines []*text.Text) error

	// Type outputs type name.
	Type(name *text.Text) error

	// Function outputs function name.
	Function(name *text.Text) error

	// Empty outputs empty section content.
	Empty(name *text.Text) error

	// Code outputs program code.
	Code(id string, code *text.Text) error

	// Signature outputs function signature.
	Signature(code *text.Text) error

	// Start starts a logical documentation section.
	Start(section string) error

	// End ends a logical documentation section.
	End(section string) error

	URL(container, id string) string
}

var packages = make(map[string]*Package)
var typeDefs = make(map[string]*Package)
var mains = make(map[string]*Package)

func documentation(params *utils.Params, files []string, doc Documenter) error {
	err := parseInputs(params, files)
	if err != nil {
		return err
	}

	var pkgs []*Package
	for _, pkg := range packages {
		pkgs = append(pkgs, pkg)

		// Collect types.
		for _, ti := range pkg.Types {
			typeDefs[ti.TypeName] = pkg
		}
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].Name < pkgs[j].Name
	})

	var mns []*Package
	for _, pkg := range mains {
		mns = append(mns, pkg)
	}
	sort.Slice(mns, func(i, j int) bool {
		return mns[i].Name < mns[j].Name
	})

	doc.Index(pkgs, mns)

	for _, pkg := range pkgs {
		out, err := doc.New("pkg_" + pkg.Docfile())
		if err != nil {
			return err
		}
		if err := documentPackage("Package", out, pkg); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	for _, pkg := range mns {
		out, err := doc.New("prg_" + pkg.Docfile())
		if err != nil {
			return err
		}
		if err := documentPackage("Program", out, pkg); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

func parseInputs(params *utils.Params, files []string) error {
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
				if e.Name() == "internal" {
					continue
				}
				files = append(files, path.Join(file, e.Name()))
			}
			err = parseInputs(params, files)
			if err != nil {
				return err
			}
		} else {
			err = parseFile(params, file)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseFile(params *utils.Params, name string) error {
	if !compiler.IsFilename(name) {
		return nil
	}

	pkg, err := compiler.New(params).ParseFile(name)
	if err != nil {
		return err
	}

	var p *Package
	var ok bool
	if pkg.Name == "main" {
		_, ok = mains[pkg.Source]
		if ok {
			return fmt.Errorf("file '%v' already processed", name)
		}
		p = &Package{
			Name: pkg.Source,
		}
		mains[pkg.Source] = p
	} else {
		p, ok = packages[pkg.Name]
		if !ok {
			p = &Package{
				Name: pkg.Name,
			}
			packages[pkg.Name] = p
		}
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

func documentPackage(kind string, out Output, pkg *Package) error {
	builtin := pkg.Name == "builtin"

	if err := out.H1(text.New().Plainf("%s %s", kind, pkg.Name)); err != nil {
		return nil
	}
	err := annotations(out, pkg.Annotations)
	if err != nil {
		return err
	}
	if err := out.H2(text.New().Plain("Constants")); err != nil {
		return err
	}
	sort.Slice(pkg.Constants, func(i, j int) bool {
		return pkg.Constants[i].Name < pkg.Constants[j].Name
	})
	var hadConstants bool
	for _, c := range pkg.Constants {
		if !builtin && !c.Exported() {
			continue
		}
		hadConstants = true
		if err := out.Code("", text.New().Plain(c.String())); err != nil {
			return err
		}
		err = annotations(out, c.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadConstants {
		out.Empty(text.New().Plain(emptySection))
	}

	err = out.H2(text.New().Plain("Variables"))
	if err != nil {
		return err
	}
	sort.Slice(pkg.Variables, func(i, j int) bool {
		return pkg.Variables[i].Names[0] < pkg.Variables[j].Names[0]
	})
	var hadVariables bool
	for _, v := range pkg.Variables {
		if !builtin && !ast.IsExported(v.Names[0]) {
			continue
		}
		hadVariables = true
		if err := out.Code("", text.New().Plain(v.String())); err != nil {
			return err
		}
		err = annotations(out, v.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadVariables {
		out.Empty(text.New().Plain(emptySection))
	}

	err = out.H2(text.New().Plain("Functions"))
	if err != nil {
		return err
	}
	sort.Slice(pkg.Functions, func(i, j int) bool {
		return pkg.Functions[i].Name < pkg.Functions[j].Name
	})
	for _, f := range pkg.Functions {
		if !builtin && f.Name != "main" && !ast.IsExported(f.Name) {
			continue
		}
		if err := out.Signature(text.New().Plain(f.Name)); err != nil {
			return err
		}
		if err := out.Code("", formatFunction(out, f)); err != nil {
			return err
		}
		err = annotations(out, f.Annotations)
		if err != nil {
			return err
		}
	}

	err = out.H2(text.New().Plain("Types"))
	if err != nil {
		return err
	}
	sort.Slice(pkg.Types, func(i, j int) bool {
		return pkg.Types[i].TypeName < pkg.Types[j].TypeName
	})
	var hadTypes bool
	for _, t := range pkg.Types {
		if !builtin && !ast.IsExported(t.TypeName) {
			continue
		}
		hadTypes = true
		if err := out.Code(t.TypeName, formatType(out, t, true)); err != nil {
			return err
		}
		err = annotations(out, t.Annotations)
		if err != nil {
			return err
		}
	}
	if !hadTypes {
		out.Empty(text.New().Plain(emptySection))
	}

	return nil
}

func formatFunction(out Output, f *ast.Func) *text.Text {
	txt := text.New()

	if f.This != nil {
		txt.Plainf("func (%s ", f.This.Name)
		txt.Append(formatType(out, f.This.Type, false))
		txt.Plainf(") %s(", f.Name)
	} else {
		txt.Plainf("func %s(", f.Name)
	}

	for idx, arg := range f.Args {
		if idx > 0 {
			txt.Plain(", ")
		}
		if idx+1 < len(f.Args) && arg.Type.Equal(f.Args[idx+1].Type) {
			txt.Plain(arg.Name)
		} else {
			txt.Plainf("%s ", arg.Name)
			txt.Append(formatType(out, arg.Type, false))
		}
	}
	txt.Plain(")")

	if len(f.Return) > 0 {
		if f.NamedReturn {
			txt.Plain(" (")
			for idx, ret := range f.Return {
				if idx > 0 {
					txt.Plain(", ")
				}
				if idx+1 < len(f.Return) &&
					ret.Type.Equal(f.Return[idx+1].Type) {
					txt.Plain(ret.Name)
				} else {
					txt.Plainf("%s ", ret.Name)
					txt.Append(formatType(out, ret.Type, false))
				}
			}
			txt.Plain(")")
		} else if len(f.Return) > 1 {
			txt.Plain(" (")
			for idx, ret := range f.Return {
				if idx > 0 {
					txt.Plain(", ")
				}
				txt.Append(formatType(out, ret.Type, false))
			}
			txt.Plain(")")
		} else {
			txt.Plain(" ")
			txt.Append(formatType(out, f.Return[0].Type, false))
		}
	}

	return txt
}

func formatType(out Output, ti *ast.TypeInfo, pp bool) *text.Text {
	txt := text.New()

	if pp {
		txt.Plain("type ")

		if strings.HasSuffix(ti.TypeName, "Size") {
			txt.Plain(ti.TypeName[:len(ti.TypeName)-4]).Oblique("Size")
		} else {
			txt.Plain(ti.TypeName)
		}
		txt.Plain(" ")
	}

	switch ti.Type {
	case ast.TypeName:
		// XXX ti.Name.Defined
		pkg, ok := typeDefs[ti.Name.Name]
		if ok {
			return txt.Link(out.URL(pkg.Name, ti.Name.Name),
				text.New().Plain(ti.Name.String()))
		}
		var typeName string
		info, err := types.Parse(ti.Name.Name)
		if err == nil {
			switch info.Type {
			case types.TInt:
				typeName = "intSize"
			case types.TUint:
				typeName = "uintSize"
			case types.TFloat:
				typeName = "floatSize"
			case types.TString:
				typeName = "stringSize"
			}
		}
		if len(typeName) > 0 {
			pkg, ok = typeDefs[typeName]
			if ok {
				return txt.Link(out.URL(pkg.Name, typeName),
					text.New().Plain(ti.Name.String()))
			}
		}
		return txt.Plain(ti.Name.String())

	case ast.TypeArray:
		return txt.
			Plainf("[%s]", ti.ArrayLength).
			Append(formatType(out, ti.ElementType, false))

	case ast.TypeSlice:
		return txt.Plainf("[]").Append(formatType(out, ti.ElementType, false))

	case ast.TypeStruct:
		txt.Plain("struct {")

		if pp {
			var width int
			for _, field := range ti.StructFields {
				if len(field.Name) > width {
					width = len(field.Name)
				}
			}
			for idx, field := range ti.StructFields {
				if idx == 0 {
					txt.Plain("\n")
				}
				txt.Plain("    ")
				txt.Plain(field.Name)
				for i := len(field.Name); i < width; i++ {
					txt.Plain(" ")
				}
				txt.Plain(" ").
					Append(formatType(out, field.Type, false)).
					Plain("\n")
			}
		} else {
			for idx, field := range ti.StructFields {
				if idx > 0 {
					txt.Plain(", ")
				}
				txt.Plainf("%s ", field.Name).
					Append(formatType(out, field.Type, false))
			}

		}
		return txt.Plain("}")

	case ast.TypeAlias:
		return txt.Plain("= ").Append(formatType(out, ti.AliasType, false))

	case ast.TypePointer:
		return txt.Plain("*").Append(formatType(out, ti.ElementType, false))

	default:
		return txt.Plainf("{TypeInfo %d}", ti.Type)
	}
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
				out.P(stringsToText(lines))
				lines = nil
				inPre = true
			}
			lines = append(lines, ann)
		} else if empty {
			if inPre {
				lines = append(lines, ann)
			} else {
				out.P(stringsToText(lines))
				lines = nil
			}
		} else {
			if inPre {
				out.Pre(trimWsPrefix(lines))
				lines = nil
				inPre = false
			}
			lines = append(lines, ann)
		}
	}
	if inPre {
		out.Pre(trimWsPrefix(lines))
	} else {
		out.P(stringsToText(lines))
	}
	return out.End("documentation")
}

func stringsToText(lines []string) []*text.Text {
	var result []*text.Text
	for _, line := range lines {
		result = append(result, text.New().Plain(line))
	}
	return result
}

func trimWsPrefix(lines []string) []*text.Text {
	l := -1
	for _, line := range lines {
		for idx, r := range line {
			if !unicode.IsSpace(r) {
				if l < 0 || idx < l {
					l = idx
				}
				break
			}
		}
	}
	var result []*text.Text
	for _, line := range lines {
		if len(line) > l {
			line = line[l:]
		}
		result = append(result, text.New().Plain(line))
	}
	return result
}

func wsPrefix(str string) (int, bool) {
	runes := []rune(str)
	var idx int
	for _, r := range runes {
		if !unicode.IsSpace(r) {
			return idx, false
		}
		if r == '\t' {
			idx += 8 - idx%8
		} else {
			idx++
		}
	}
	return len(runes), true
}

// Package contains all documentation items for a MPCL package.
type Package struct {
	Name        string
	Annotations ast.Annotations
	Constants   []*ast.ConstantDef
	Variables   []*ast.VariableDef
	Functions   []*ast.Func
	Types       []*ast.TypeInfo
}

// Docfile returns the package's documentation filename.
func (pkg *Package) Docfile() string {
	return strings.ReplaceAll(pkg.Name, "/", "_")
}
