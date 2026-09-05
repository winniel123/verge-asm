import React from "react";

export function Sparkline({ data = [], width = 140, height = 36, color = "var(--chart-1)", area = true, strokeWidth = 1.75, style }) {
  const [on, setOn] = React.useState(false);
  React.useEffect(() => { const id = requestAnimationFrame(() => setOn(true)); return () => cancelAnimationFrame(id); }, []);
  if (data.length < 2) return <span style={{ display: "inline-block", width, height, ...style }} />;
  const min = Math.min(...data), max = Math.max(...data), span = max - min || 1;
  const pad = 3;
  const px = (i) => pad + (i / (data.length - 1)) * (width - pad * 2);
  const py = (v) => pad + (1 - (v - min) / span) * (height - pad * 2);
  const pts = data.map((v, i) => px(i).toFixed(1) + "," + py(v).toFixed(1)).join(" ");
  const areaD = "M" + pad + "," + (height - pad) + " L" + pts.split(" ").join(" L") + " L" + px(data.length - 1).toFixed(1) + "," + (height - pad) + " Z";
  return (
    <svg width={width} height={height} viewBox={"0 0 " + width + " " + height} style={{ display: "block", overflow: "visible", ...style }} aria-hidden="true">
      {area && <path d={areaD} fill={color} style={{ opacity: on ? 0.1 : 0, transition: "opacity 400ms var(--ease-out) 250ms" }}></path>}
      <polyline points={pts} fill="none" stroke={color} strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round" pathLength="100" strokeDasharray="100" style={{ strokeDashoffset: on ? 0 : 100, transition: "stroke-dashoffset 550ms var(--ease-out)" }}></polyline>
      <circle cx={px(data.length - 1)} cy={py(data[data.length - 1])} r="2.5" fill={color} style={{ opacity: on ? 1 : 0, transition: "opacity 250ms var(--ease-out) 480ms" }}></circle>
    </svg>
  );
}
