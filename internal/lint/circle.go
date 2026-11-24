package lint

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vsilvestre/deslop-please-go/internal/report"
)

const held = report.HeldInMind

// tie is how one member of a circle knows another.
type tie struct {
	items   map[string]bool
	holders map[string]bool
	sites   []report.Location
}

func (t *tie) add(item, holder string, at report.Location) {
	if t.items == nil {
		t.items, t.holders = map[string]bool{}, map[string]bool{}
	}
	t.items[item] = true
	t.holders[holder] = true
	t.sites = append(t.sites, at)
}

// cost is how many items it ties by: what cutting it takes.
func (t *tie) cost() int { return len(t.items) }

func (t *tie) first() report.Location {
	first := t.sites[0]
	for _, at := range t.sites {
		if before(at, first) {
			first = at
		}
	}
	return first
}

func before(a, b report.Location) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Col < b.Col
}

type link struct {
	user, used string
	tie        *tie
}

// circleReport describes a circle of members by the ties between them: where
// its cheapest tie is, a note for each, and a picture of the shortest circle
// through the cheapest tie. Ties of fewest items are the cheapest to cut.
func circleReport(verb, noun string, members int, ties []link) (*report.Location, []string, report.Picture) {
	sort.Slice(ties, func(i, j int) bool {
		a, b := ties[i], ties[j]
		if a.tie.cost() != b.tie.cost() {
			return a.tie.cost() < b.tie.cost()
		}
		return a.user+"\x00"+a.used < b.user+"\x00"+b.used
	})
	var notes []string
	for _, l := range ties[:min(len(ties), held)] {
		notes = append(notes, fmt.Sprintf("%s: `%s` %s `%s` by %s: %s",
			l.tie.first(), l.user, verb, l.used, counted(len(l.tie.items), "item"), listed(sorted(l.tie.items))))
	}
	if len(ties) > held {
		notes = append(notes, "and "+counted(len(ties)-held, "more tie"))
	}
	circle := shortestCircle(ties)
	var steps []report.Step
	for _, at := range circle {
		l := ties[at]
		steps = append(steps, report.Step{Member: l.user, At: l.tie.first(), Uses: stepUses(l.tie)})
	}
	var left []string
	if n := members - len(circle); n > 0 {
		left = append(left, counted(n, "more "+noun))
	}
	if n := len(ties) - len(circle); n > 0 {
		left = append(left, counted(n, "more tie"))
	}
	rest := ""
	if len(left) > 0 {
		rest = "not drawn: " + strings.Join(left, ", ")
	}
	at := ties[0].tie.first()
	return &at, notes, report.Circle{Steps: steps, Rest: rest}
}

func stepUses(t *tie) string {
	items := strings.Join(sorted(t.items), ", ")
	if holders := sorted(t.holders); len(holders) == 1 {
		return holders[0] + " uses " + items
	}
	return "uses " + items
}

// shortestCircle is the indexes of the ties of the shortest circle through
// the first of ties, the first first.
func shortestCircle(ties []link) []int {
	start, end := ties[0].user, ties[0].used
	via := map[string]int{end: -1}
	queue := []string{end}
	for len(queue) > 0 && queue[0] != start {
		at := queue[0]
		queue = queue[1:]
		for i, l := range ties {
			if _, seen := via[l.used]; l.user == at && !seen {
				via[l.used] = i
				queue = append(queue, l.used)
			}
		}
	}
	circle := []int{0}
	var back []int
	for at := start; ; at = ties[via[at]].user {
		if i, ok := via[at]; !ok || i < 0 {
			break
		}
		back = append(back, via[at])
	}
	for i := len(back) - 1; i >= 0; i-- {
		circle = append(circle, back[i])
	}
	return circle
}

// circles are the strongly connected components of adj of more than one
// node, each sorted, in order of their first node (Tarjan).
func circles(adj [][]int) [][]int {
	var (
		index, low = make([]int, len(adj)), make([]int, len(adj))
		onStack    = make([]bool, len(adj))
		stack      []int
		next       = 1
		found      [][]int
		visit      func(v int)
	)
	visit = func(v int) {
		index[v], low[v] = next, next
		next++
		stack, onStack[v] = append(stack, v), true
		for _, w := range adj[v] {
			switch {
			case index[w] == 0:
				visit(w)
				low[v] = min(low[v], low[w])
			case onStack[w]:
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var component []int
		for {
			w := stack[len(stack)-1]
			stack, onStack[w] = stack[:len(stack)-1], false
			component = append(component, w)
			if w == v {
				break
			}
		}
		if len(component) > 1 {
			sort.Ints(component)
			found = append(found, component)
		}
	}
	for v := range adj {
		if index[v] == 0 {
			visit(v)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i][0] < found[j][0] })
	return found
}

func sorted(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// listed is `a, b, +2 more`: at most held names.
func listed(names []string) string {
	shown := strings.Join(names[:min(len(names), held)], ", ")
	if more := len(names) - held; more > 0 {
		return fmt.Sprintf("%s, +%d more", shown, more)
	}
	return shown
}

func counted(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
