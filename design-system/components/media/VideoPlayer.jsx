import React from "react";
import { Icon } from "./Icon.jsx";

const fmt = (s) => { s = Math.max(0, Math.floor(s || 0)); return Math.floor(s / 60) + ":" + String(s % 60).padStart(2, "0"); };
export function VideoPlayer({ src, poster, label, aspect = "16 / 9", style }) {
  const vref = React.useRef(null), wref = React.useRef(null), idleT = React.useRef(null);
  const [playing, setPlaying] = React.useState(false);
  const [time, setTime] = React.useState(0);
  const [dur, setDur] = React.useState(0);
  const [muted, setMuted] = React.useState(false);
  const [fs, setFs] = React.useState(false);
  const [pseudoFs, setPseudoFs] = React.useState(false);
  const [vol, setVol] = React.useState(1);
  const [volScrub, setVolScrub] = React.useState(false);
  const [idle, setIdle] = React.useState(false);
  const [scrub, setScrub] = React.useState(false);
  const [err, setErr] = React.useState(false);
  React.useEffect(() => {
    const onFs = () => setFs(document.fullscreenElement === wref.current);
    document.addEventListener("fullscreenchange", onFs);
    return () => { document.removeEventListener("fullscreenchange", onFs); clearTimeout(idleT.current); };
  }, []);
  React.useEffect(() => {
    if (!pseudoFs) return;
    const onEsc = (e) => { if (e.key === "Escape") setPseudoFs(false); };
    document.addEventListener("keydown", onEsc);
    return () => document.removeEventListener("keydown", onEsc);
  }, [pseudoFs]);
  const poke = () => {
    setIdle(false);
    clearTimeout(idleT.current);
    idleT.current = setTimeout(() => setIdle(true), 2200);
  };
  const toggle = () => {
    const v = vref.current;
    if (!v || err) return;
    if (v.paused) { v.play(); poke(); } else v.pause();
  };
  const seekBy = (d) => { const v = vref.current; if (v && dur) { v.currentTime = Math.max(0, Math.min(dur, v.currentTime + d)); poke(); } };
  const seekTo = (e) => {
    const v = vref.current;
    if (!v || !dur) return;
    const r = e.currentTarget.getBoundingClientRect();
    v.currentTime = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width)) * dur;
  };
  const trackDown = (e) => { e.currentTarget.setPointerCapture(e.pointerId); setScrub(true); seekTo(e); };
  const trackMove = (e) => { if (scrub) seekTo(e); };
  const onKey = (e) => {
    if (e.key === " " || e.key === "k") { e.preventDefault(); toggle(); }
    else if (e.key === "m") setMutedBoth(!muted);
    else if (e.key === "f") toggleFs();
    else if (e.key === "ArrowLeft") { e.preventDefault(); seekBy(-5); }
    else if (e.key === "ArrowRight") { e.preventDefault(); seekBy(5); }
    else if (e.key === "ArrowUp") { e.preventDefault(); applyVol(vol + 0.1); }
    else if (e.key === "ArrowDown") { e.preventDefault(); applyVol(vol - 0.1); }
  };
  const setMutedBoth = (m) => { setMuted(m); if (vref.current) vref.current.muted = m; };
  const applyVol = (v) => {
    v = Math.max(0, Math.min(1, v));
    setVol(v);
    if (vref.current) vref.current.volume = v;
    if (v > 0 && muted) setMutedBoth(false);
    if (v === 0 && !muted) setMutedBoth(true);
    poke();
  };
  const volTo = (e) => {
    const r = e.currentTarget.getBoundingClientRect();
    applyVol((e.clientX - r.left) / r.width);
  };
  const inFs = fs || pseudoFs;
  const toggleFs = () => {
    if (document.fullscreenElement === wref.current) { document.exitFullscreen(); return; }
    if (pseudoFs) { setPseudoFs(false); return; }
    const el = wref.current;
    if (el && el.requestFullscreen && document.fullscreenEnabled) {
      const p = el.requestFullscreen();
      if (p && p.catch) p.catch(() => setPseudoFs(true));
    } else setPseudoFs(true);
  };
  const pct = dur ? (time / dur) * 100 : 0;
  const showBar = !playing || !idle || scrub || volScrub;
  const volIcon = muted || vol === 0 ? "volume-x" : vol < 0.5 ? "volume-1" : "volume-2";
  const cbtn = { width: 30, height: 30, display: "inline-flex", alignItems: "center", justifyContent: "center", background: "transparent", border: "none", borderRadius: 8, color: "#fff", cursor: "pointer", padding: 0 };
  return (
    <div ref={wref} role="region" aria-label={label || "Video player"} tabIndex={0} onKeyDown={onKey} onPointerMove={poke}
      style={{ position: "relative", aspectRatio: pseudoFs ? "auto" : aspect, background: "#0d0c0a", borderRadius: pseudoFs ? 0 : 12, overflow: "hidden", border: pseudoFs ? "none" : "1px solid var(--border-default)", outlineOffset: 3, cursor: playing && idle && !scrub && !volScrub ? "none" : "default", ...(pseudoFs ? { position: "fixed", inset: 0, zIndex: 1200 } : null), ...style }}>
      <video ref={vref} src={src} poster={poster} playsInline preload="metadata"
        onPlay={() => { setPlaying(true); poke(); }} onPause={() => { setPlaying(false); setIdle(false); }} onEnded={() => setPlaying(false)}
        onTimeUpdate={(e) => setTime(e.currentTarget.currentTime)} onLoadedMetadata={(e) => setDur(e.currentTarget.duration)} onError={() => setErr(true)}
        onClick={toggle} style={{ position: "absolute", inset: 0, width: "100%", height: "100%", objectFit: "contain", display: "block" }} />
      {err && (
        <div style={{ position: "absolute", inset: 0, display: "flex", alignItems: "center", justifyContent: "center" }}>
          <span style={{ font: "400 12px var(--font-mono)", color: "rgba(255,255,255,0.72)" }}>video unavailable — check the source</span>
        </div>
      )}
      {!playing && !err && (
        <button type="button" aria-label="Play" onClick={toggle}
          style={{ position: "absolute", top: "50%", left: "50%", transform: "translate(-50%, -50%)", width: 54, height: 54, borderRadius: "50%", border: "none", cursor: "pointer", display: "flex", alignItems: "center", justifyContent: "center", background: "var(--accent)", color: "var(--on-accent)", boxShadow: "0 4px 18px rgba(0,0,0,0.35)" }}>
          <Icon name="play" size={22} style={{ marginLeft: 3 }} />
        </button>
      )}
      <div aria-hidden={showBar ? undefined : true}
        style={{ position: "absolute", left: 0, right: 0, bottom: 0, padding: "26px 12px 8px", background: "linear-gradient(transparent, rgba(9,8,6,0.72))", display: "flex", flexDirection: "column", gap: 6, opacity: showBar ? 1 : 0, pointerEvents: showBar ? "auto" : "none", transition: "opacity var(--dur-base) var(--ease-out)" }}>
        <div onPointerDown={trackDown} onPointerMove={trackMove} onPointerUp={() => setScrub(false)} onPointerCancel={() => setScrub(false)}
          role="slider" aria-label="Seek" aria-valuemin={0} aria-valuemax={Math.floor(dur)} aria-valuenow={Math.floor(time)} aria-valuetext={fmt(time) + " of " + fmt(dur)}
          style={{ padding: "5px 0", cursor: "pointer", touchAction: "none" }}>
          <div style={{ position: "relative", height: 4, borderRadius: 2, background: "rgba(255,255,255,0.28)" }}>
            <div style={{ position: "absolute", left: 0, top: 0, bottom: 0, width: pct + "%", borderRadius: 2, background: "var(--accent)" }} />
            <div style={{ position: "absolute", top: "50%", left: pct + "%", transform: "translate(-50%, -50%)", width: 10, height: 10, borderRadius: "50%", background: "#fff", boxShadow: "0 1px 4px rgba(0,0,0,0.4)", opacity: scrub ? 1 : 0.85 }} />
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
          <button type="button" aria-label={playing ? "Pause" : "Play"} onClick={toggle} style={cbtn}><Icon name={playing ? "pause" : "play"} size={16} /></button>
          <button type="button" aria-label={muted ? "Unmute" : "Mute"} onClick={() => setMutedBoth(!muted)} style={cbtn}><Icon name={volIcon} size={16} /></button>
          <div onPointerDown={(e) => { e.currentTarget.setPointerCapture(e.pointerId); setVolScrub(true); volTo(e); }} onPointerMove={(e) => { if (volScrub) volTo(e); }} onPointerUp={() => setVolScrub(false)} onPointerCancel={() => setVolScrub(false)}
            role="slider" aria-label="Volume" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round((muted ? 0 : vol) * 100)}
            style={{ width: 56, padding: "9px 0", cursor: "pointer", touchAction: "none", flex: "0 0 auto" }}>
            <div style={{ position: "relative", height: 3, borderRadius: 2, background: "rgba(255,255,255,0.28)" }}>
              <div style={{ position: "absolute", left: 0, top: 0, bottom: 0, width: (muted ? 0 : vol) * 100 + "%", borderRadius: 2, background: "#fff" }} />
              <div style={{ position: "absolute", top: "50%", left: (muted ? 0 : vol) * 100 + "%", transform: "translate(-50%, -50%)", width: 8, height: 8, borderRadius: "50%", background: "#fff", boxShadow: "0 1px 3px rgba(0,0,0,0.4)" }} />
            </div>
          </div>
          <span style={{ font: "400 11px var(--font-mono)", color: "rgba(255,255,255,0.85)", marginLeft: 4 }}>{fmt(time)} / {fmt(dur)}</span>
          {label && <span style={{ marginLeft: 12, font: "400 11px var(--font-ui)", color: "rgba(255,255,255,0.6)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{label}</span>}
          <button type="button" aria-label={inFs ? "Exit fullscreen" : "Fullscreen"} onClick={toggleFs} style={{ ...cbtn, marginLeft: "auto" }}><Icon name={inFs ? "minimize" : "maximize"} size={15} /></button>
        </div>
      </div>
    </div>
  );
}
