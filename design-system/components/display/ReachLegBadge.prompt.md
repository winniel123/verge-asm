One `Reach` leg chip — reached / not reached / never looked / stopped looking. A leg holds one of two values and nothing else, so never render an `Exposure` word (`exposed`, `edge-only`, `firewalled`, `unreachable`) here. An `Exposure` value needs both legs, and it belongs in a chip of its own.

Tone keys on the value and the class together. Only the internet leg's `reached` is danger-toned, because that is the move the product alerts on. A `Gap` reads `stopped looking` in warn, and a never-configured leg reads `never looked` in a dashed neutral: the two absences keep their two statements.

```jsx
<ReachLegBadge state="reached" legClass="internet" />
<ReachLegBadge state="stopped-looking" legClass="internal" size="sm" />
```
