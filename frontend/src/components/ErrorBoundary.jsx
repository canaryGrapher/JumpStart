import { Component } from "react";

// Catches render errors so a single bad component can't blank the whole app.
export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error) {
    return { error };
  }

  componentDidCatch(error, info) {
    console.error("Unhandled render error:", error, info);
  }

  reset = () => this.setState({ error: null });

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="error-boundary">
        <h2>Something went wrong</h2>
        <p>{String(this.state.error?.message || this.state.error)}</p>
        <button className="btn primary" onClick={this.reset}>
          Dismiss
        </button>
      </div>
    );
  }
}
