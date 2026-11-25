package lint

import (
	"fmt"
	"strings"

	"github.com/vasilvestre/deslop-please-go/internal/model"
	"github.com/vasilvestre/deslop-please-go/internal/report"
)

// tangles: things of one package that know each other in a circle, written
// in more than one file, or more of them than a reader holds in mind. Where
// a tangle is written is its members' declarations and the references that
// tie it, so moving a method to another file of its own changes nothing.
func tangles(x *index) []report.Diagnostic {
	ties := map[*model.Package]map[[2]*thing]*tie{}
	x.eachUse(func(u usage) {
		if u.target.pkg != u.pkg || u.target == u.user {
			return
		}
		if ties[u.pkg] == nil {
			ties[u.pkg] = map[[2]*thing]*tie{}
		}
		key := [2]*thing{u.user, u.target}
		if ties[u.pkg][key] == nil {
			ties[u.pkg][key] = &tie{}
		}
		ties[u.pkg][key].add(itemName(u.obj, u.target), u.holder, x.where(u.id.Pos()))
	})
	var diagnostics []report.Diagnostic
	for _, pkg := range x.model.Packages {
		diagnostics = append(diagnostics, packageTangles(x.things[pkg], ties[pkg])...)
	}
	return diagnostics
}

func packageTangles(things []*thing, ties map[[2]*thing]*tie) []report.Diagnostic {
	adj := make([][]int, len(things))
	for pair := range ties {
		adj[pair[0].id] = append(adj[pair[0].id], pair[1].id)
	}
	var diagnostics []report.Diagnostic
	for _, circle := range circles(adj) {
		members := map[*thing]bool{}
		files := map[string]bool{}
		for _, id := range circle {
			members[things[id]] = true
			files[things[id].at.File] = true
		}
		var inside []link
		for pair, t := range ties {
			if members[pair[0]] && members[pair[1]] {
				inside = append(inside, link{qualified(pair[0]), qualified(pair[1]), t})
				for _, at := range t.sites {
					files[at.File] = true
				}
			}
		}
		if len(files) > 1 || len(circle) > held {
			diagnostics = append(diagnostics, tangleDiagnostic(members, files, inside))
		}
	}
	return diagnostics
}

func qualified(t *thing) string { return t.pkg.Path + "." + t.name }

func tangleDiagnostic(members map[*thing]bool, files map[string]bool, inside []link) report.Diagnostic {
	names := map[string]bool{}
	for m := range members {
		names[qualified(m)] = true
	}
	list := sorted(names)
	at, ties, picture := circleReport("knows", "thing", len(members), inside)
	notes := []string{"members: " + listed(list)}
	where := "all in `" + sorted(files)[0] + "`"
	if len(files) > 1 {
		where = "across " + counted(len(files), "file")
		notes = append(notes, "written in: "+listed(sorted(files)))
	}
	notes = append(notes, ties...)
	return report.Diagnostic{
		Lint:    report.Tangle,
		Key:     strings.Join(list, ","),
		Message: fmt.Sprintf("%d things know each other in a circle, %s", len(members), where),
		At:      at,
		Notes:   notes,
		Picture: picture,
	}
}
