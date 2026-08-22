import * as React from "react";
export interface Annotation {
  /** Reason prose — the whole record; no author, no timestamp, no state */
  reason: string;
}
export interface AnnotationControlProps {
  /** Present = annotated (accepted risk); absent = the entry form */
  annotation?: Annotation | null;
  onAnnotate?: (reason: string) => void;
  onRemove?: () => void;
  style?: React.CSSProperties;
}
export function AnnotationControl(props: AnnotationControlProps): JSX.Element;
