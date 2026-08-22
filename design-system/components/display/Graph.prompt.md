Asset graph canvas — typed nodes (domain/subdomain/ip/service), severity halos, drag to pan, wheel + controls to zoom. Pair with GraphLegend. Static coordinates; severity halos use the sev dot colors.
```jsx
<Graph nodes={[{id:"d",label:"acmecorp.io",type:"domain",x:100,y:200,sev:"medium"}]} edges={[]} selectedId={sel} onNodeSelect={setSel} />
<GraphLegend />
```
