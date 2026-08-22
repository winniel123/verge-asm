Report KPI block — micro-label + mono period + Stat + chart slot. Compose the reports grid from these.
```jsx
<ReportCard title="Open signals" period="Aug 8 – Aug 22" value="47" delta="+3" deltaTone="bad">
  <Sparkline data={trend} width={999} style={{width:"100%"}} />
</ReportCard>
```
