Before/after drift block — adds in ok-green, removals in danger-red with a true minus, always mono.
```jsx
<DiffView title="Open ports · drift" lines={[{type:"same",text:":443 https nginx/1.25.4"},{type:"add",text:":5900 vnc"},{type:"remove",text:":8080 http-alt"}]} />
```
