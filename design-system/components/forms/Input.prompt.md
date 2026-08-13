Single-line text field; mono variant for hostnames, IPs, CIDRs.

Text field with 1px ink border and square focus ring. Set `mono` for technical values.

```jsx
<Input label="Target" mono placeholder="acmecorp.io or 203.0.113.0/24"
       hint="Domain, subdomain, IP, or CIDR range" />
<Input label="Name" error="A target with this name exists" />
```
