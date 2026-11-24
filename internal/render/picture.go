package render

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/vsilvestre/deslop-please-go/internal/report"
)

// barWidth is the longest bar, in cells: with a name and a length, a line
// fits in 80 columns. A longer bar ends in `⋯`.
const barWidth = 40

func draw(picture report.Picture, p paint) []string {
	switch picture := picture.(type) {
	case report.Circle:
		return circle(picture, p)
	case report.Spread:
		return spread(picture, p)
	case report.Lists:
		rows := []row{{"shows", picture.Shown, true}, {"hides", picture.Hidden, false}}
		for _, long := range picture.Long {
			rows = append(rows, row{long.Name, long.N, true})
		}
		return bars(rows, report.HeldInMind, p)
	}
	return nil
}

// circle draws each member under the last, the tie to the next between them,
// and the last tie back up to the first. The first tie, the cheapest, is heavy.
func circle(c report.Circle, p paint) []string {
	sites := shortSites(c.Steps)
	width := 0
	for _, site := range sites {
		width = max(width, utf8.RuneCountInString(site))
	}
	var lines []string
	for i, step := range c.Steps {
		into, bar := "│   ", "│"
		if i == 0 {
			into, bar = "┌─▶ ", p.red("┃")
		}
		lines = append(lines, into+p.bold(step.Member))
		lines = append(lines, fmt.Sprintf("│     %s %-*s  %s", bar, width, sites[i], step.Uses))
		if i+1 < len(c.Steps) {
			down := "▼"
			if i == 0 {
				down = p.red(down)
			}
			lines = append(lines, "│     "+down)
		}
	}
	lines = append(lines, "└─────┘")
	if c.Rest != "" {
		lines = append(lines, c.Rest)
	}
	return lines
}

// shortSites are each step's `file:line`, from the directory all share.
func shortSites(steps []report.Step) []string {
	common := directory(steps[0].At.File)
	for _, step := range steps {
		dir := directory(step.At.File)
		n := 0
		for n < len(common) && n < len(dir) && common[n] == dir[n] {
			n++
		}
		common = common[:n]
	}
	skip := 0
	for _, part := range common {
		skip += len(part) + 1
	}
	sites := make([]string, len(steps))
	for i, step := range steps {
		sites[i] = fmt.Sprintf("%s:%d", step.At.File[skip:], step.At.Line)
	}
	return sites
}

func directory(file string) []string {
	parts := strings.Split(file, "/")
	return parts[:len(parts)-1]
}

// spread draws the type, then the field with an arm from each package that
// works on it, then its other fields.
func spread(s report.Spread, p paint) []string {
	lines := []string{s.Owner}
	digits := 0
	for _, pkg := range s.Packages {
		digits = max(digits, len(fmt.Sprint(pkg.N)))
	}
	under := strings.Repeat(" ", utf8.RuneCountInString(s.Field)+3)
	for i, pkg := range s.Packages {
		lead, arm := under, "  ┣━"
		switch {
		case i == 0 && len(s.Packages) == 1:
			lead, arm = "  "+p.bold(s.Field)+" ", "◀━━━"
		case i == 0:
			lead, arm = "  "+p.bold(s.Field)+" ", "◀━┳━"
		case i+1 == len(s.Packages):
			arm = "  ┗━"
		}
		lines = append(lines, fmt.Sprintf("%s%s %*d  %s", lead, p.red(arm), digits, pkg.N, pkg.Name))
	}
	for _, other := range s.Others {
		lines = append(lines, "  "+other)
	}
	return lines
}

type row struct {
	name   string
	length int
	judged bool
}

// bars draws a bar for each row, with a line at most; past it, a judged
// row's bar is heavy. A cell is one, or as many as put the line halfway along
// the longest bar, and part of one fills one.
func bars(rows []row, most int, p paint) []string {
	unit := max((most+barWidth/2-1)/(barWidth/2), 1)
	line := (most + unit - 1) / unit
	width := 0
	for _, r := range rows {
		width = max(width, utf8.RuneCountInString(r.name))
	}
	lines := []string{fmt.Sprintf("%*s  %*d", width, "", line+1, most)}
	for _, r := range rows {
		cells := (r.length + unit - 1) / unit
		inside := min(cells, line)
		past := cells - inside
		beyond := strings.Repeat("█", min(past, barWidth-line))
		if past > barWidth-line {
			beyond += "⋯"
		}
		if r.judged && past > 0 {
			beyond = p.red(beyond)
		}
		lines = append(lines, fmt.Sprintf("%-*s  %s%s│%s %d", width, r.name,
			strings.Repeat("█", inside), strings.Repeat(" ", line-inside), beyond, r.length))
	}
	return lines
}
