import React from "react";
import { Wizard } from "../../components/feedback/Wizard.jsx";
import { TagInput } from "../../components/forms/TagInput.jsx";
import { CadenceSelect } from "../../components/forms/CadenceSelect.jsx";
import { RadioCards } from "../../components/forms/RadioCards.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";

export function Onboarding({ open, onClose, onFinish }) {
  const [seeds, setSeeds] = React.useState([]);
  const [profile, setProfile] = React.useState("standard");
  const [cad, setCad] = React.useState("Daily · 08:00");
  const [cron, setCron] = React.useState("");
  const [channel, setChannel] = React.useState("");
  const micro = { font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" };
  return (
    <Wizard open={open} title="Set up this workspace" description="Three calls and the first scan runs." onClose={onClose} finishLabel="Start first scan"
      onFinish={() => onFinish && onFinish({ seeds, profile })}
      steps={[
        { id: "seeds", title: "Seeds", valid: seeds.length > 0, content: (
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <span style={micro}>What you own</span>
            <TagInput values={seeds} onChange={setSeeds} placeholder="acmecorp.io, 203.0.113.0/24" />
            <span style={{ font: "400 12px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>Domains or CIDR ranges. Discovery expands each seed into subjects — you never enumerate hosts by hand.</span>
          </div>
        ) },
        { id: "cadence", title: "Cadence", valid: cad !== "Custom…" || cron.trim().length > 0, content: (
          <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
            <RadioCards label="Scan profile" value={profile} onChange={setProfile} columns={1} options={[
              { value: "standard", title: "Standard", description: "Top 1,000 TCP ports, plus any port previously seen.", meta: "default" },
              { value: "passive", title: "Passive only", description: "Public datasets only — no active probing." },
            ]} />
            <CadenceSelect value={cad} customValue={cron} onChange={(v, c) => { setCad(v); setCron(c || ""); }} />
          </div>
        ) },
        { id: "channel", title: "Channel", valid: true, content: (
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <Input label="Delivery URL (optional)" mono placeholder="https://ops.example/hook" value={channel} onChange={(e) => setChannel(e.target.value)}
              hint="Signal and drift summaries post here. Add more channels later in Settings." />
          </div>
        ) },
        { id: "review", title: "Review", content: (
          <KeyValueList items={[
            { k: "Seeds", v: seeds.join(", ") || "—" },
            { k: "Profile", v: profile },
            { k: "Cadence", v: cad === "Custom…" ? cron : cad.toLowerCase() },
            { k: "Channel", v: channel.trim() || "none — inbox only" },
          ]} />
        ) },
      ]} />
  );
}
