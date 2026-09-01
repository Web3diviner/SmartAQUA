/**
 * Error boundary.
 *
 * 15_AQUADOC_FRONTEND.md section 10: avoid exposing stack traces. The stack is
 * logged to the console for the developer and never rendered into the page.
 */

import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Console only — never into the DOM.
    console.error('Unhandled UI error', error, info.componentStack)
  }

  render(): ReactNode {
    if (!this.state.hasError) return this.props.children

    return (
      <div className="error-boundary">
        <h2>Something failed while rendering this view</h2>
        <p>
          The details were written to the browser console. Reload the page to continue; if it
          recurs, the console output is what to report.
        </p>
        <button
          type="button"
          className="button"
          onClick={() => this.setState({ hasError: false })}
        >
          Try again
        </button>
      </div>
    )
  }
}
