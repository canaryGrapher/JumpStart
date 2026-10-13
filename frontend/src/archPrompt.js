// Builds the prompt users paste into their AI agent so it writes a detailed
// JumpStart architecture document to .jumpstart/architecture.xml in the
// project root. Unlike confPrompt, the agent MUST write the file.

export function buildArchPrompt(projectRoot) {
  const rootHint = projectRoot
    ? `The project root is:\n${projectRoot}\n\n`
    : "";

  return `You are generating a detailed architecture document for JumpStart, my local project manager.

${rootHint}Your job:
1. Inspect this repository thoroughly (code, configs, schemas, APIs, UI, data stores, process wiring).
2. Create the folder \`.jumpstart\` at the project root if it does not exist.
3. Write a single file: \`.jumpstart/architecture.xml\`
4. Do NOT write any other files. Do NOT print the full XML in chat unless writing the file failed — prefer writing the file and briefly confirming the path.

This is NOT a brief. It must be a very detailed visual knowledge map: graphs, charts, tables, data flows, and database structures so JumpStart can render an interactive Architecture tab.

XML schema (version 1). Use this exact structure and attribute names:

\`\`\`xml
<?xml version="1.0" encoding="UTF-8"?>
<architecture version="1" generatedAt="ISO-8601-timestamp">
  <project name="Project Name" summary="One-sentence product summary"/>
  <overview
    description="2–4 sentence architecture narrative"
    stack="Lang1,Lang2,Framework,…"
    principles="Principle one,Principle two,…"/>

  <section id="code-arch" title="Code Architecture" kind="graph">
    <summary>How the codebase is layered and packaged.</summary>
    <node id="unique-id" label="Display name" kind="package|service|ui|lib|worker" description="What this node owns"/>
    <!-- many nodes covering major packages/modules -->
    <edge from="id" to="id" label="imports|calls|owns" description="Why this dependency exists"/>
    <table title="Top-level packages" headers="Path|Role|Language">
      <row cells="path|role|lang"/>
    </table>
    <chart type="bar" title="Approximate size by area">
      <point label="Area" value="42"/>
    </chart>
  </section>

  <section id="data-flow" title="Data Flow" kind="flow">
    <summary>Primary request and event paths through the system.</summary>
    <step id="s1" label="Entry" description="Who starts the flow"/>
    <step id="s2" label="Next" description="What happens next"/>
    <!-- ordered steps; also add node/edge graphs for branching flows -->
    <node id="…" label="…" kind="…" description="…"/>
    <edge from="…" to="…" label="…" description="…"/>
    <table title="Key events / messages" headers="Name|Producer|Consumer|Payload">
      <row cells="…|…|…|…"/>
    </table>
  </section>

  <section id="db-arch" title="Database" kind="db">
    <summary>Persistent stores, schemas, and relationships.</summary>
    <entity name="table_or_collection" description="What it stores">
      <column name="id" type="uuid" pk="true" description="Primary key"/>
      <column name="…" type="…" description="…"/>
    </entity>
    <relation from="child" to="parent" type="N:1" description="FK meaning"/>
    <table title="Indexes / constraints" headers="Entity|Name|Columns|Notes">
      <row cells="…|…|…|…"/>
    </table>
  </section>

  <section id="features" title="Features" kind="mixed">
    <summary>Product capabilities mapped to code.</summary>
    <table title="Features" headers="Feature|Status|Entry points|Notes">
      <row cells="…|shipped|path::Symbol|…"/>
    </table>
    <chart type="pie" title="Feature status mix">
      <point label="shipped" value="10"/>
      <point label="partial" value="3"/>
    </chart>
    <node id="…" label="…" kind="feature" description="…"/>
    <edge from="…" to="…" label="uses" description="…"/>
  </section>

  <section id="apis" title="APIs &amp; Bindings" kind="table">
    <summary>External and internal API surfaces.</summary>
    <table title="Public / binding methods" headers="Name|Side|Purpose|Inputs">
      <row cells="…|go-binding|…|…"/>
    </table>
  </section>

  <section id="runtime" title="Runtime &amp; Processes" kind="flow">
    <summary>How the app boots, runs subprocesses, and shuts down.</summary>
    <step id="r1" label="Boot" description="…"/>
    <node id="…" label="…" kind="process" description="…"/>
    <edge from="…" to="…" label="spawns" description="…"/>
  </section>
</architecture>
\`\`\`

Rules:
- Fill every section above with real detail from THIS repo. Add more \`<section>\` blocks if useful (security, testing, deploy, MCP tools, sync, analytics) using the same element vocabulary: node, edge, step, entity, column, relation, table, chart, summary.
- Prefer many precise nodes/edges over a few vague ones. Descriptions on nodes/edges/columns/relations are required — JumpStart shows them on hover.
- Use stable ascii ids (kebab-case). Pipe-separate table cells and headers. Comma-separate stack/principles lists.
- chart type must be \`bar\` or \`pie\`; values are numbers.
- If there is no traditional database, still document config files, JSON stores, SQLite, keychains, or in-memory stores under db-arch.
- After writing the file, reply with one short confirmation including the absolute path to \`.jumpstart/architecture.xml\`.`;
}
