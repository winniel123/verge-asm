import * as React from "react";
export interface DialogProps {
  open: boolean;
  title: string;
  /** Secondary line under the title */
  description?: string;
  /** Right-aligned action row, e.g. <><Button variant="ghost">Cancel</Button><Button>Add seed</Button></> */
  footer?: React.ReactNode;
  /** Scrim click, Escape, and the close control all call this */
  onClose?: () => void;
  /** Panel width in px. Default 480 */
  width?: number;
  children?: React.ReactNode;
}
export function Dialog(props: DialogProps): JSX.Element | null;
