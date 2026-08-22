Multi-step setup dialog (numbered progress, Back/Next, per-step valid gate). Constructive flows only — destructive confirmation is ConfirmDialog.
```jsx
<Wizard open={open} title="New report schedule" steps={[{id:"scope",title:"Scope",content:<.../>,valid:!!name},{id:"cadence",title:"Cadence",content:<CadenceSelect/>},{id:"review",title:"Review",content:<KeyValueList items={...}/>}]} onClose={close} onFinish={create} finishLabel="Create schedule" />
```
