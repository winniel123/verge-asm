package custody

// Output-affecting content moves the version; an operator act moves it not at all (ADR-0008).

const Version = "custody/v3" // a threshold move with no bump fails A6 (golden-corpus.md §10)
