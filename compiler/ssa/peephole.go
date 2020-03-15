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

func (rule Rule) Match(instrs []Instr) []Instr {
	env := make(map[string]Variable)

	// Match all patterns
	for idx, p := range rule.Pattern {
		instr := instrs[idx]
		if instr.Op != p.Op {
			return nil
		}
		if len(instr.In) != len(p.In) {
			return nil
		}
		for i, in := range p.In {
			if !matchVar(env, in, instr.In[i]) {
				return nil
			}
		}
		if instr.Out == nil {
			return nil
		}
		if !matchVar(env, p.Out, *instr.Out) {
			return nil
		}
	}

	var result []Instr
	for _, r := range rule.Replace {
		instr, err := r.Expand(env)
		if err != nil {
			fmt.Printf("template expansion failed: %s\n", err)
			return nil
		}
		result = append(result, instr)
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

func (b *Block) Peephole(seen map[string]bool) error {
	if seen[b.ID] {
		return nil
	}
	seen[b.ID] = true

outer:
	for i := 0; i < len(b.Instr); i++ {
		for _, rule := range rules {
			if len(rule.Pattern) > len(b.Instr)-i {
				continue
			}
			match := rule.Match(b.Instr[i : i+len(rule.Pattern)])
			if match == nil {
				continue
			}

			var n []Instr
			n = append(n, b.Instr[:i]...)
			n = append(n, match...)
			n = append(n, b.Instr[i+len(rule.Pattern):]...)
			b.Instr = n
			continue outer
		}
	}

	if b.Next != nil {
		err := b.Next.Peephole(seen)
		if err != nil {
			return err
		}
	}
	if b.Branch != nil {
		err := b.Branch.Peephole(seen)
		if err != nil {
			return err
		}
	}

	return nil
}

func Peephole(block *Block) error {
	return block.Peephole(make(map[string]bool))
}
