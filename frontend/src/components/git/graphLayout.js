// Assigns each commit a column (lane) and derives parent edges so the
// timeline can be drawn as a git-style graph. Commits arrive newest-first
// (parents appear at a later row). Lane assignment is the classic
// "reserve a lane for the next expected parent" approach.

export function layoutGraph(commits) {
  const rowOf = new Map(commits.map((c, i) => [c.hash, i]));
  const colOf = new Array(commits.length).fill(0);
  const edges = [];
  let lanes = []; // lanes[i] = hash this lane is waiting to reach, or null

  const freeLane = (hash) => {
    let idx = lanes.indexOf(null);
    if (idx === -1) {
      idx = lanes.length;
      lanes.push(null);
    }
    lanes[idx] = hash;
    return idx;
  };

  commits.forEach((c, row) => {
    let col = lanes.indexOf(c.hash);
    if (col === -1) col = freeLane(c.hash);
    colOf[row] = col;

    // Release every lane that was waiting on this commit.
    lanes = lanes.map((l) => (l === c.hash ? null : l));

    const parents = (c.parents || []).filter((p) => rowOf.has(p));
    parents.forEach((p, pi) => {
      let pcol;
      const existing = lanes.indexOf(p);
      if (existing !== -1) {
        pcol = existing; // another child already reserved this parent
      } else if (pi === 0 && (lanes[col] == null || lanes[col] === p)) {
        pcol = col; // first parent continues this commit's lane
        lanes[pcol] = p;
      } else {
        pcol = freeLane(p);
      }
      edges.push({ fromRow: row, toRow: rowOf.get(p) });
      colOf[rowOf.get(p)] = pcol;
    });
  });

  const maxCol = colOf.reduce((m, c) => Math.max(m, c), 0);
  return { colOf, edges, maxCol };
}

// A small, theme-agnostic palette; lanes cycle through it by column.
export const LANE_COLORS = [
  "#3b82f6",
  "#10b981",
  "#f59e0b",
  "#ef4444",
  "#8b5cf6",
  "#ec4899",
  "#14b8a6",
];

export const laneColor = (col) => LANE_COLORS[col % LANE_COLORS.length];
