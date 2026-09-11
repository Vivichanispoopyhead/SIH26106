import React from 'react';

interface WorkspaceErrorBoundaryProps {
  children: React.ReactNode;
}

interface WorkspaceErrorBoundaryState {
  error: Error | null;
}

export class WorkspaceErrorBoundary extends React.Component<WorkspaceErrorBoundaryProps, WorkspaceErrorBoundaryState> {
  state: WorkspaceErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): WorkspaceErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    // Keep the diagnostic local to the UI; do not log raw email content or payloads.
    console.error('Investigation workspace rendering failed', error.name, info.componentStack);
  }

  render(): React.ReactNode {
    if (this.state.error) {
      return (
        <section className="workspace-render-error" role="alert" data-testid="workspace-render-error">
          <span className="section-eyebrow">Workspace render failure</span>
          <h2>The investigation workspace could not be rendered</h2>
          <p>A backend result was received, but one of the investigation views could not safely display it.</p>
          <code>{this.state.error.message || 'Unknown rendering error'}</code>
          <button type="button" onClick={() => this.setState({ error: null })}>Retry workspace</button>
        </section>
      );
    }
    return this.props.children;
  }
}
