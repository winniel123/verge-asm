import * as React from "react";
/** Modal on hard-offset shadow; ink border; overlay click closes.
 * @startingPoint section="Components" subtitle="Modal with title, body, footer actions" viewport="700x320"
 */
export interface DialogProps {
  open: boolean;
  title?: React.ReactNode;
  /** Footer action row (right-aligned Buttons: secondary then primary). */
  footer?: React.ReactNode;
  onClose?: () => void;
  /** Panel width px. @default 440 */
  width?: number;
  children?: React.ReactNode;
}
export declare function Dialog(props: DialogProps): React.ReactElement | null;
