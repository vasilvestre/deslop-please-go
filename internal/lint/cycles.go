package lint

import (
	"sort"
	"strings"

	"github.com/vsilvestre/deslop-please-go/internal/report"
)

// moduleCycles: sibling directories, each with its subtree of packages, that
// use each other in a circle, so there is no order to read them in. Go
// forbids import cycles between packages; a circle between two subtrees is
// the nearest thing it allows. A package and what is inside its directory
// make no circle.
func moduleCycles(x *index) []report.Diagnostic {
	byParent := map[string]map[[2]string]*tie{}
	x.eachUse(func(u usage) {
		if u.target.pkg == u.pkg {
			return
		}
		parent, user, used, ok := siblings(u.pkg.Path, u.target.pkg.Path)
		if !ok {
			return
		}
		if byParent[parent] == nil {
			byParent[parent] = map[[2]string]*tie{}
		}
		key := [2]string{user, used}
		if byParent[parent][key] == nil {
			byParent[parent][key] = &tie{}
		}
		byParent[parent][key].add(itemName(u.obj, u.target), u.holder, x.where(u.id.Pos()))
	})
	var diagnostics []report.Diagnostic
	for _, parent := range sortedKeys(byParent) {
		ties := byParent[parent]
		children := childrenOf(ties)
		adj := make([][]int, len(children))
		for pair := range ties {
			from := sort.SearchStrings(children, pair[0])
			adj[from] = append(adj[from], sort.SearchStrings(children, pair[1]))
		}
		for _, circle := range circles(adj) {
			members := make([]string, len(circle))
			for i, at := range circle {
				members[i] = children[at]
			}
			diagnostics = append(diagnostics, cycleDiagnostic(parent, members, ties))
		}
	}
	return diagnostics
}

// siblings are the directory holding both packages, and its children that
// hold each, unless one of the two holds the other.
func siblings(a, b string) (parent, childA, childB string, ok bool) {
	segA, segB := segments(a), segments(b)
	n := 0
	for n < len(segA) && n < len(segB) && segA[n] == segB[n] {
		n++
	}
	if n == len(segA) || n == len(segB) {
		return "", "", "", false
	}
	return join(segA[:n]), join(segA[:n+1]), join(segB[:n+1]), true
}

func segments(path string) []string {
	if path == "." {
		return nil
	}
	return strings.Split(path, "/")
}

func join(parts []string) string {
	if len(parts) == 0 {
		return "."
	}
	return strings.Join(parts, "/")
}

func childrenOf(ties map[[2]string]*tie) []string {
	set := map[string]bool{}
	for pair := range ties {
		set[pair[0]], set[pair[1]] = true, true
	}
	return sorted(set)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cycleDiagnostic(parent string, members []string, ties map[[2]string]*tie) report.Diagnostic {
	in := map[string]bool{}
	for _, member := range members {
		in[member] = true
	}
	var inside []link
	for pair, t := range ties {
		if in[pair[0]] && in[pair[1]] {
			inside = append(inside, link{pair[0], pair[1], t})
		}
	}
	at, notes, picture := circleReport("uses", "module", len(members), inside)
	notes = append([]string{"members: " + listed(members)}, notes...)
	return report.Diagnostic{
		Lint:    report.ModuleCycle,
		Key:     strings.Join(members, ","),
		Message: counted(len(members), "module") + " in `" + parent + "` use each other in a circle",
		At:      at,
		Notes:   notes,
		Picture: picture,
	}
}
