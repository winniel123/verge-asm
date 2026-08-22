# IntegrationTile
One integration in the library grid: neutral letter mark, name, install state Badge (installed ok / available
neutral / needs-attention warn), 2-line description, mono category. The whole tile is a button.
Use in an auto-fill minmax(280px,1fr) grid; open details in a Drawer with ConsentList.
```jsx
<IntegrationTile name="PagerDuty" category="Notify" mark="PD" state="attention"
  description="Critical signals open incidents; withdrawn signals resolve them." onClick={openDetail} />
```
