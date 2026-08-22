import * as React from "react";
export interface SavedView { id: string; label: string; count?: number; }
export interface SavedViewsProps {
  views: SavedView[];
  activeId?: string;
  onSelect?: (id: string) => void;
  /** Current filters differ from the active view — shows "Save view" */
  dirty?: boolean;
  onSave?: () => void;
  style?: React.CSSProperties;
}
export function SavedViews(props: SavedViewsProps): JSX.Element;
