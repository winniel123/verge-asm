import * as React from "react";
export interface Mapping { from: string; to: string; }
export interface MappingEditorProps {
  /** Controlled rows. */
  mappings: Mapping[];
  onChange: (next: Mapping[]) => void;
  /** Column headings. Defaults: "IdP claim" / "Verge field". */
  fromLabel?: string;
  toLabel?: string;
  fromPlaceholder?: string;
  /** Closed set for the target side. */
  toOptions: string[];
  addLabel?: string;
  style?: React.CSSProperties;
}
export declare function MappingEditor(props: MappingEditorProps): JSX.Element;
