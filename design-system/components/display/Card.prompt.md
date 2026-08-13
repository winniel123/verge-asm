Flat white panel — the container for dashboard modules and detail sections.

```jsx
<Card eyebrow="Latest findings" action={<a href="#">View all →</a>} pad={false}>
  <Table columns={cols} rows={rows} />
</Card>
```

`emphasized` swaps the hairline for a 1px ink outline — at most one per screen. Set `pad={false}` for flush Tables.
