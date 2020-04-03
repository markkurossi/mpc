//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"regexp"
	"strings"
)

type Rule struct {
	Name    string
	Source  string
	Pattern []Template
	Replace []Template
}

type Template struct {
	Op  Operand
	In  []string
	Out string
}

func (t Template) Expand(env map[string]Variable) (Instr, error) {
	var in []Variable
	var out Variable

	for _, i := range t.In {
		v, ok := env[i]
		if !ok {
			return Instr{}, fmt.Errorf("input variable %s not bound", i)
		}
		in = append(in, v)
	}
	out, ok := env[t.Out]
	if !ok {
		return Instr{}, fmt.Errorf("output variable %s not bound", t.Out)
	}

	return Instr{
		Op:  t.Op,
		In:  in,
		Out: &out,
	}, nil
}

func (rule Rule) Match(steps []Step) []Step {
	env := make(map[string]Variable)

	// Match all patterns
	for idx, p := range rule.Pattern {
		step := steps[idx]
		if step.Instr.Op != p.Op {
			return nil
		}
		if len(step.Instr.In) != len(p.In) {
			return nil
		}
		for i, in := range p.In {
			if !matchVar(env, in, step.Instr.In[i]) {
				return nil
			}
		}
		if step.Instr.Out == nil {
			return nil
		}
		if !matchVar(env, p.Out, *step.Instr.Out) {
			return nil
		}
	}

	var result []Step
	for _, r := range rule.Replace {
		instr, err := r.Expand(env)
		if err != nil {
			fmt.Printf("template expansion failed: %s\n", err)
			return nil
		}
		// XXX variable liveness in expansions
		result = append(result, Step{
			Instr: instr,
		})
	}

	return result
}

func matchVar(env map[string]Variable, pattern string, v Variable) bool {
	binding, ok := env[pattern]
	if ok {
		// Pattern already bound, must be equal binding.
		return v.Equal(&binding)
	}
	switch pattern[0] {
	case 'V':
	case 'C':
		if !v.Const {
			return false
		}
	case '$':
		if pattern != v.Name {
			return false
		}
	}
	env[pattern] = v

	return true
}

var rules = []*Rule{
	&Rule{
		Name: "Constant-shift-test => bts",
		Source: `rshift V1 C1 V2
                 band   V2 $1 V3
                 neq    V3 $0 V4
              =>
                 bts    V1 C1 V4`,
	},
	&Rule{
		Name: "Constant-shift-test => btc",
		Source: `rshift V1 C1 V2
                 band   V2 $1 V3
                 eq     V3 $0 V4
              =>
                 btc    V1 C1 V4`,
	},
	&Rule{
		Name: "Constant-shift-test => bts",
		Source: `rshift V1 C1 V2
                 band   V2 $1 V3
                 eq     V3 $1 V4
              =>
                 bts    V1 C1 V4`,
	},
	&Rule{
		Name: "Constant-shift-test => btc",
		Source: `rshift V1 C1 V2
                 band   V2 $1 V3
                 neq    V3 $1 V4
              =>
                 btc    V1 C1 V4`,
	},
	&Rule{
		Name: "Slice-cast => slice",
		Source: `slice V1 V2 V3 V4
                 mov   V4 V5
              =>
                 slice V1 V2 V3 V5`,
	},
}

var reSpace = regexp.MustCompilePOSIX(`[[:space:]]+`)

func init() {
	for _, r := range rules {
		var inReplace bool

		for _, line := range strings.Split(r.Source, "\n") {
			parts := reSpace.Split(strings.TrimSpace(line), -1)
			if len(parts) == 1 && parts[0] == "=>" {
				inReplace = true
				continue
			}
			if len(parts) < 3 {
				panic(fmt.Sprintf("unexpected pattern: %s", line))
			}
			var op Operand
			var found bool

			for k, v := range operands {
				if v == parts[0] {
					op = k
					found = true
					break
				}
			}
			if !found {
				panic(fmt.Sprintf("unknown operand '%s'", parts[0]))
			}
			tmpl := Template{
				Op:  op,
				In:  parts[1 : len(parts)-1],
				Out: parts[len(parts)-1],
			}
			if inReplace {
				r.Replace = append(r.Replace, tmpl)
			} else {
				r.Pattern = append(r.Pattern, tmpl)
			}
		}
	}
}

func (prog *Program) Peephole() error {

outer:
	for i := 0; i < len(prog.Steps); i++ {
		for _, rule := range rules {
			if len(rule.Pattern) > len(prog.Steps)-i {
				continue
			}
			match := rule.Match(prog.Steps[i : i+len(rule.Pattern)])
			if match == nil {
				continue
			}

			var n []Step
			n = append(n, prog.Steps[:i]...)
			n = append(n, match...)
			n = append(n, prog.Steps[i+len(rule.Pattern):]...)
			prog.Steps = n
			continue outer
		}
	}

	return nil
}
