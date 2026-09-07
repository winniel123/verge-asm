package main

import (
	"maps"

	"github.com/winniel123/verge-asm/internal/db"
)

const shellKey = "InShell"

func pageData(acct db.Account, title, navActive string, rest ...map[string]any) map[string]any {
	data := map[string]any{
		shellKey:    true,
		"Title":     title,
		"Account":   acct,
		"IsAdmin":   acct.Role == roleAdmin,
		"NavActive": navActive,
	}
	for _, m := range rest {
		maps.Copy(data, m)
	}
	return data
}

func barePageData(title string) map[string]any {
	return map[string]any{"Title": title}
}
