import * as React from "react";
export interface Exclusion {
  kind: "name" | "subtree" | "address";
  value: string;
}
export interface ExclusionEditorProps {
  exclusions: Exclusion[];
  onAdd?: (kind: Exclusion["kind"], value: string) => void;
  onRemove?: (index: number) => void;
  style?: React.CSSProperties;
}
export function ExclusionEditor(props: ExclusionEditorProps): JSX.Element;
