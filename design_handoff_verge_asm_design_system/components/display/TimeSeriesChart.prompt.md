Axed multi-series line chart for Reports; Sparkline stays the inline form. Chart tokens --chart-1..4 only, never severity colors. Hover shows a per-series readout.
```jsx
<TimeSeriesChart series={[{label:"All open",data:open},{label:"Critical + high",data:crit}]} labels={sparseDays} hoverLabels={days} />
```
