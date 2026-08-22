import * as React from "react";
export interface WizardStep {
  id: string;
  title: string;
  content: React.ReactNode;
  /** false disables Next/Finish for this step */
  valid?: boolean;
}
export interface WizardProps {
  open: boolean;
  title: string;
  description?: string;
  steps: WizardStep[];
  onClose?: () => void;
  /** Called on the last step's primary action */
  onFinish?: () => void;
  /** Default "Finish" */
  finishLabel?: string;
  /** Default 560 */
  width?: number;
}
export function Wizard(props: WizardProps): JSX.Element;
