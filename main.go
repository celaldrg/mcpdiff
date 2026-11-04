// Command mcpdiff compares two MCP tools/list snapshots and reports breaking changes.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type schema struct {
	Type       any                `json:"type"`
	Properties map[string]*schema `json:"properties"`
	Required   []string           `json:"required"`
	Enum       []any              `json:"enum"`
}

type Change struct {
	Level string `json:"level"` // breaking | warning | info
	Tool  string `json:"tool"`
	Msg   string `json:"message"`
}

func load(path string) (map[string]Tool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parse(b)
}

func parse(b []byte) (map[string]Tool, error) {
	var wrap struct {
		Tools  []Tool `json:"tools"`
		Result *struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}
	var tools []Tool
	if err := json.Unmarshal(b, &wrap); err == nil && (wrap.Tools != nil || wrap.Result != nil) {
		tools = wrap.Tools
		if wrap.Result != nil {
			tools = wrap.Result.Tools
		}
	} else if err := json.Unmarshal(b, &tools); err != nil {
		return nil, fmt.Errorf("not a tools/list snapshot: %w", err)
	}
	m := map[string]Tool{}
	for _, t := range tools {
		m[t.Name] = t
	}
	return m, nil
}

func sch(raw json.RawMessage) *schema {
	s := &schema{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, s)
	}
	return s
}

func set(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func diffSchema(tool, path string, o, n *schema, out *[]Change) {
	add := func(l, f string, a ...any) { *out = append(*out, Change{l, tool, fmt.Sprintf(f, a...)}) }
	if fmt.Sprint(o.Type) != fmt.Sprint(n.Type) && o.Type != nil {
		add("breaking", "%stype changed %v -> %v", path, o.Type, n.Type)
	}
	if len(o.Enum) > 0 {
		ne := map[string]bool{}
		for _, v := range n.Enum {
			ne[fmt.Sprint(v)] = true
		}
		for _, v := range o.Enum {
			if len(n.Enum) > 0 && !ne[fmt.Sprint(v)] {
				add("breaking", "%senum value %v removed", path, v)
			}
		}
	}
	oreq, nreq := set(o.Required), set(n.Required)
	for _, p := range keys(o.Properties) {
		np, ok := n.Properties[p]
		if !ok {
			add("breaking", "param %s%s removed", path, p)
			continue
		}
		diffSchema(tool, path+p+".", o.Properties[p], np, out)
	}
	for _, p := range keys(n.Properties) {
		if _, ok := o.Properties[p]; !ok {
			if nreq[p] {
				add("breaking", "new required param %s%s", path, p)
			} else {
				add("info", "new optional param %s%s", path, p)
			}
		}
	}
	for _, p := range keys(nreq) {
		if !oreq[p] {
			if _, existed := o.Properties[p]; existed {
				add("breaking", "param %s%s became required", path, p)
			}
		}
	}
}

func Diff(old, nw map[string]Tool) []Change {
	var out []Change
	for _, name := range keys(old) {
		n, ok := nw[name]
		if !ok {
			out = append(out, Change{"breaking", name, "tool removed"})
			continue
		}
		if old[name].Description != n.Description {
			out = append(out, Change{"warning", name, "description changed (agents may pick tools differently)"})
		}
		diffSchema(name, "", sch(old[name].InputSchema), sch(n.InputSchema), &out)
	}
	for _, name := range keys(nw) {
		if _, ok := old[name]; !ok {
			out = append(out, Change{"info", name, "tool added"})
		}
	}
	return out
}

func run(args []string, w io.Writer) int {
	fs := flag.NewFlagSet("mcpdiff", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "output JSON")
	failOn := fs.String("fail-on", "breaking", "exit 1 on: breaking|warning|never")
	if err := fs.Parse(args); err != nil || fs.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: mcpdiff [-json] [-fail-on breaking|warning|never] OLD.json NEW.json")
		return 2
	}
	o, err := load(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	n, err := load(fs.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	ch := Diff(o, n)
	counts := map[string]int{}
	for _, c := range ch {
		counts[c.Level]++
	}
	if *asJSON {
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		_ = e.Encode(ch)
	} else {
		icon := map[string]string{"breaking": "BREAKING", "warning": "WARN    ", "info": "info    "}
		for _, c := range ch {
			fmt.Fprintf(w, "%s  %-20s %s\n", icon[c.Level], c.Tool, c.Msg)
		}
		fmt.Fprintf(w, "\n%d breaking, %d warnings, %d info\n", counts["breaking"], counts["warning"], counts["info"])
	}
	switch {
	case *failOn == "breaking" && counts["breaking"] > 0,
		*failOn == "warning" && counts["breaking"]+counts["warning"] > 0:
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout)) }
