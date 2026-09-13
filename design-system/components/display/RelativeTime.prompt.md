Terse relative time + ISO 8601 tooltip — use for every timestamp instead of hand-rolling.
```jsx
<RelativeTime value="4m" iso="2026-08-22T14:02:11Z" />
```
Not for a label that has to part two otherwise identical controls: the relative form buckets siblings together, and the tooltip needs a mouse. Such a label states the instant in full. See [ADR-1875](../../../docs/adr/1875-an-undo-control-renders-the-exact-lookup-instant-against-the-relative-timestamp-convention.md).
