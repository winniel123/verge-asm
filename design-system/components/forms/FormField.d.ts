import * as React from "react";
export interface FormFieldProps {
  /** Also set this id on the control inside so label/summary focus work */
  id?: string;
  label?: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children?: React.ReactNode;
  style?: React.CSSProperties;
}
export function FormField(props: FormFieldProps): JSX.Element;
export interface FormError {
  label: string;
  message?: string;
  /** id of the control to focus on click */
  fieldId?: string;
}
export interface FormErrorSummaryProps {
  errors: FormError[];
  /** Default "N fields need attention" */
  title?: string;
  style?: React.CSSProperties;
}
export function FormErrorSummary(props: FormErrorSummaryProps): JSX.Element | null;
