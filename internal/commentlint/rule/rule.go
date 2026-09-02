package rule

import (
	"go/parser"
	"go/scanner"
	"go/token"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/winniel123/verge-asm/internal/commentlint/screen"
	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

type Class string

const (
	GeneratedHeader               Class = "generated-header"
	Directive                     Class = "directive"
	Todo                          Class = "todo"
	CommentedOutCode              Class = "commented-out-code"
	SectionDivider                Class = "section-divider"
	PackageDoc                    Class = "package-doc"
	DocstringExportedConventional Class = "docstring-exported-conventional"
	DocstringExportedOther        Class = "docstring-exported-other"
	DocstringUnexported           Class = "docstring-unexported"
	Citation                      Class = "citation"
	ExternalSpec                  Class = "external-spec"
	WhyNote                       Class = "why-note"
	ChangeNarration               Class = "change-narration"
	StepNarration                 Class = "step-narration"
	ShortLabel                    Class = "short-label"
	ProseOther                    Class = "prose-other"
)

const (
	RuleGoDeclComment       = "go-decl-comment"
	RuleChangeNarration     = "change-narration"
	RuleTodoMarker          = "todo-marker"
	RuleCitationOverOneLine = "citation-over-one-line"
)

const shortLabelWords = 6

const sectionDividerShare = 0.6

var (
	generatedRe = regexp.MustCompile(`Code generated .*DO NOT EDIT`)
	todoRe      = regexp.MustCompile(`^(TODO|FIXME|XXX|HACK|BUG)\b`)
	ruleRunRe   = regexp.MustCompile(`[-=*_~#+]{4,}`)
)

func Classify(b surface.Block) Class {
	payload := b.Payload()
	lines := b.PayloadLines()
	// One class per block, by the §2.1 precedence order. The delete decision
	// needs exactly one answer (SPEC §6.6).
	switch {
	case generatedRe.MatchString(payload):
		return GeneratedHeader
	case b.Directive:
		return Directive
	case todoRe.MatchString(firstField(lines)):
		return Todo
	case commentedOutCode(payload):
		return CommentedOutCode
	case sectionDivider(lines):
		return SectionDivider
	}
	// §2.1 rules 6 to 9 reach a package clause, a named identifier and a struct
	// field. A `const (` opener or an import spec declares none of those, so it
	// falls through to the content classes and an agent judges it.
	if b.Declaration && !b.DeclGroup {
		switch {
		case b.PackageDoc:
			return PackageDoc
		case exported(b.DeclName) && firstWord(lines) == b.DeclName:
			return DocstringExportedConventional
		case exported(b.DeclName):
			return DocstringExportedOther
		}
		return DocstringUnexported
	}
	switch {
	case screen.HasCitation(payload):
		return Citation
	case screen.HasExternalSpec(payload):
		return ExternalSpec
	case screen.HasWhyMarker(payload):
		return WhyNote
	case screen.HasHistoryMarker(payload):
		return ChangeNarration
	case screen.HasLooseNarration(payload):
		return StepNarration
	case b.Lines() == 1 && len(strings.Fields(payload)) <= shortLabelWords:
		return ShortLabel
	}
	return ProseOther
}

var deleteSet = map[Class]bool{
	SectionDivider:                true,
	CommentedOutCode:              true,
	ShortLabel:                    true,
	DocstringExportedConventional: true,
	DocstringUnexported:           true,
}

func InDeleteSet(c Class) bool {
	return deleteSet[c]
}

// DeleteSet lets the §3.9 gate name a class that no block reached, because a
// class with no verdict is a hole in the gate rather than a pass.
func DeleteSet() []Class {
	out := make([]Class, 0, len(deleteSet))
	for c := range deleteSet {
		out = append(out, c)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

func Deletable(b surface.Block) (Class, string, bool) {
	// §3.4 keeps every trailing comment with the agent, so a caller hands the
	// delete pass own-line blocks alone.
	c := Classify(b)
	signal := screen.Signal(b.Payload())
	return c, signal, deleteSet[c] && signal == ""
}

type Finding struct {
	Line  int
	Rule  string
	Class Class
}

func Lint(res surface.Result, testFile bool) []Finding {
	var out []Finding
	// Flagging is advisory, so a block that trips several rules reports each
	// one (SPEC §6.6).
	for _, b := range res.Blocks {
		out = append(out, flags(b, false, testFile)...)
	}
	for _, b := range res.Trailing {
		out = append(out, flags(b, true, testFile)...)
	}
	return out
}

func flags(b surface.Block, trailing, testFile bool) []Finding {
	c := Classify(b)
	if c == Directive || c == GeneratedHeader {
		return nil
	}
	// Ruling 12 forbids the tool from guessing at intent, and both classes
	// need intent to judge (SPEC §3.5).
	if c == StepNarration || c == ProseOther {
		return nil
	}

	var out []Finding
	add := func(id string) {
		out = append(out, Finding{Line: b.StartLine, Rule: id, Class: c})
	}
	// The id is the class name wherever a rule maps to a class (SPEC §6.6).
	// Ratchet rule 2 is the only rule that reaches a trailing comment, and it
	// reaches short-label alone (SPEC §3.5).
	if deleteSet[c] && (!trailing || c == ShortLabel) {
		add(string(c))
	}
	if b.Declaration && !trailing {
		add(RuleGoDeclComment)
	}
	if c == ChangeNarration {
		add(RuleChangeNarration)
	}
	if c == Todo {
		add(RuleTodoMarker)
	}
	if c == Citation && !testFile && b.Lines() > 1 {
		add(RuleCitationOverOneLine)
	}
	return out
}

func exported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper([]rune(name)[0])
}

func firstWord(lines []string) string {
	return strings.TrimRight(firstField(lines), ":.,")
}

func firstField(lines []string) string {
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		return fields[0]
	}
	return ""
}

func sectionDivider(lines []string) bool {
	total, rules := 0, 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		total++
		if ruleLine(line) {
			rules++
		}
	}
	if total == 0 {
		return false
	}
	return float64(rules)/float64(total) >= sectionDividerShare
}

func ruleLine(line string) bool {
	if ruleRunRe.MatchString(line) {
		return true
	}
	run := 0
	for _, r := range line {
		if r >= 0x2500 && r <= 0x257F {
			run++
			if run >= 4 {
				return true
			}
			continue
		}
		run = 0
	}
	return false
}

var codeTokens = map[token.Token]bool{
	token.DEFINE: true, token.LBRACE: true, token.RBRACE: true,
	token.RETURN: true, token.FUNC: true, token.IF: true, token.ELSE: true,
	token.FOR: true, token.RANGE: true, token.GO: true, token.DEFER: true,
	token.SWITCH: true, token.SELECT: true, token.VAR: true, token.CONST: true,
	token.TYPE: true, token.IMPORT: true, token.PACKAGE: true, token.ARROW: true,
	token.INC: true, token.DEC: true, token.ASSIGN: true, token.ADD_ASSIGN: true,
	token.SUB_ASSIGN: true, token.MUL_ASSIGN: true, token.QUO_ASSIGN: true,
}

func commentedOutCode(payload string) bool {
	if strings.TrimSpace(payload) == "" {
		return false
	}
	// The parse alone is not enough, because `see the note above` parses as an
	// expression statement (SPEC §6.6).
	return codeOnlyToken(payload) && parsesAsGo(payload)
}

func codeOnlyToken(payload string) bool {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(payload))
	var s scanner.Scanner
	s.Init(file, []byte(payload), func(token.Position, string) {}, 0)
	for {
		_, tok, _ := s.Scan()
		if tok == token.EOF {
			return false
		}
		if codeTokens[tok] {
			return true
		}
	}
}

func parsesAsGo(payload string) bool {
	if _, err := parser.ParseExpr(payload); err == nil {
		return true
	}
	fset := token.NewFileSet()
	body := "package p\nfunc _() {\n" + payload + "\n}\n"
	if _, err := parser.ParseFile(fset, "", body, parser.SkipObjectResolution); err == nil {
		return true
	}
	top := "package p\n" + payload + "\n"
	_, err := parser.ParseFile(fset, "", top, parser.SkipObjectResolution)
	return err == nil
}
