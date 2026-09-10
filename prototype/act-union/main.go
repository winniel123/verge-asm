package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.
//
//	go run ./prototype/act-union          writes act-union.html and prints the table
//	go test ./prototype/act-union         runs the three hazards

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"time"
)

type View struct {
	Limb        string
	Route       string
	Class       string
	Note        string
	When        string
	Actor       string
	Action      string
	Subject     string
	ActorKind   string
	ActorJSON   string
	SubjectJSON string
}

func build() []View {
	base := time.Now().Add(-3 * time.Hour)
	out := make([]View, 0, len(catalogue))
	for i, e := range catalogue {
		row, err := Encode(e.Act, e.Actor)
		if err != nil {
			panic(err)
		}
		row.CreatedAt = base.Add(time.Duration(i) * 137 * time.Second)
		r, err := Render(row)
		if err != nil {
			panic(err)
		}
		out = append(out, View{
			Limb: e.Limb, Route: e.Route, Class: e.Act.Class(), Note: e.Note,
			When: r.When, Actor: r.Actor, Action: r.Action, Subject: r.Subject,
			ActorKind: row.ActorKind, ActorJSON: string(row.Actor), SubjectJSON: string(row.Subject),
		})
	}
	return out
}

func main() {
	views := build()

	widest := map[string]int{}
	byLimb := map[string]int{}
	for _, v := range views {
		byLimb[v.Limb]++
		for col, s := range map[string]string{"When": v.When, "Actor": v.Actor, "Action": v.Action, "Subject": v.Subject} {
			if len(s) > widest[col] {
				widest[col] = len(s)
			}
		}
	}

	fmt.Printf("%-4s %-26s %-8s %-22s %-24s %s\n", "LIMB", "CLASS", "WHEN", "ACTOR", "ACTION", "SUBJECT")
	for _, v := range views {
		fmt.Printf("%-4s %-26s %-8s %-22s %-24s %s\n", v.Limb, v.Class, v.When, v.Actor, v.Action, v.Subject)
	}

	fmt.Printf("\n%d variants\n", len(views))
	limbs := make([]string, 0, len(byLimb))
	for k := range byLimb {
		limbs = append(limbs, k)
	}
	sort.Strings(limbs)
	for _, k := range limbs {
		fmt.Printf("  limb %-4s %d\n", k, byLimb[k])
	}
	fmt.Printf("widest cell: When %d · Actor %d · Action %d · Subject %d chars\n",
		widest["When"], widest["Actor"], widest["Action"], widest["Subject"])

	f, err := os.Create("act-union.html")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := page.Execute(f, map[string]any{
		"Views":  views,
		"Widest": widest,
		"Limbs":  byLimb,
	}); err != nil {
		panic(err)
	}
	fmt.Println("\nwrote act-union.html")
}

func hazard(classes ...string) func([]View) []View {
	want := map[string]bool{}
	for _, c := range classes {
		want[c] = true
	}
	return func(vs []View) []View {
		out := []View{}
		for _, v := range vs {
			if want[v.Class] {
				out = append(out, v)
			}
		}
		return out
	}
}

var page = template.Must(template.New("p").Funcs(template.FuncMap{
	"hazard1": hazard("seed.withdrawn", "account.removed", "sso.binding.removed", "scan.stopped"),
	"hazard2": hazard("sso.provider.secret.set", "channel.declared", "channel.updated", "token.minted"),
	"hazard3": hazard("restore.applied", "migration.applied"),
	"join":    func(s []string) string { return strings.Join(s, " ") },
}).Parse(pageHTML))
