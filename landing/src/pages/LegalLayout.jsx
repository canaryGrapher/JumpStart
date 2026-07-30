import PageShell from "../components/PageShell";

// Legal pages: the shared page chrome wrapped around a centered prose column.
export default function LegalLayout(props) {
  return (
    <PageShell>
      <main class="legal-doc container narrow">
        <h1>{props.title}</h1>
        <p class="legal-updated">Last updated: {props.updated}</p>
        {props.children}
      </main>
    </PageShell>
  );
}
