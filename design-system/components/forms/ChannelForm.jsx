import React from "react";
import { Input } from "./Input.jsx";
import { Checkbox } from "./Checkbox.jsx";
import { Button } from "./Button.jsx";
import { SecretInput } from "./SecretInput.jsx";

const CLASSES = ["signals", "drift", "coverage", "batches"];
export function ChannelForm({ channel, onSubmit, style }) {
  const [url, setUrl] = React.useState((channel && channel.url) || "");
  const [classes, setClasses] = React.useState((channel && channel.classes) || ["signals"]);
  const [secretSet, setSecretSet] = React.useState(!!(channel && channel.secretSet));
  const toggle = (c) => setClasses((cs) => (cs.indexOf(c) !== -1 ? cs.filter((x) => x !== c) : cs.concat(c)));
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14, fontFamily: "var(--font-ui)", ...style }}>
      <Input label="Channel URL" mono placeholder="https://ops.example/hook" value={url} onChange={(e) => setUrl(e.target.value)} hint="One-way https delivery" />
      <SecretInput isSet={secretSet} onSave={() => setSecretSet(true)} />
      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>Receives</span>
        <div style={{ display: "flex", gap: 18, flexWrap: "wrap" }}>
          {CLASSES.map((c) => <Checkbox key={c} label={c} checked={classes.indexOf(c) !== -1} onChange={() => toggle(c)} />)}
        </div>
        <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>Routing is by class and nothing finer.</span>
      </div>
      <div>
        <Button disabled={!url} onClick={() => onSubmit && onSubmit({ url, classes, secretSet })}>Save channel</Button>
      </div>
    </div>
  );
}
