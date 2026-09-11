package act

// account.username has no format check and no reserved list, so an account named
// "setup token" would otherwise render as a GrantHolder does.
//
// Disjointness is structural rather than enumerated: we author every grant label and
// none begins with the mark, so a reserved-username list would buy nothing an install
// already holding the bad account could use.

const mark = "@"

func Mark(username string) string { return mark + username }

// A payload that renders a bare username marks it, or one row renders one person two ways (§3.5).

type markedSubject interface{ markedSubject() string }

func (p AccountRef) markedSubject() string { return Mark(p.Username) }

func (p AccountAtRole) markedSubject() string { return Mark(p.Username) + sep + p.Role }

func (p BindingRef) markedSubject() string { return p.Slug + sep + Mark(p.Username) }

// ActorCell and SubjectCell are the reader's two cells, and the mark stops here. Kind()
// also tells the caller the component: mono text for an account, an st-tag for a grant.

func ActorCell(a Actor) string {
	// Keyed on the union's own discriminator, so a pointer receiver cannot drop the mark.
	if a.Kind() == KindAccount {
		return Mark(a.Name())
	}
	return a.Name()
}

func SubjectCell(a Act) string {
	if m, ok := a.(markedSubject); ok {
		return m.markedSubject()
	}
	return a.Subject()
}
