Vertical event history (signal lifecycle: raised → drifted → resolved). Newest first; terse relative times. groups=[{label: batch, events}] renders collapsible per-batch sections; event content embeds a DiffView.
```jsx
<Timeline events={[{title:"withdrawn",time:"2m",dotColor:"var(--drift-loss-solid)"},{title:"Port re-opened",detail:":5900 answered again",time:"4m",tone:"danger",mono:true}]} />
```
