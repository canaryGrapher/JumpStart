// Projects v2 field data types, and the small helpers the editors share.
// GitHub's board exposes writable fields the user fills in and read-only
// rollups it computes; only the first group gets an editor.

export const WRITABLE = new Set([
  "TEXT",
  "NUMBER",
  "DATE",
  "SINGLE_SELECT",
  "ITERATION",
  "TITLE",
]);

// Fields JumpStart already shows natively on the card, so repeating them
// in the GitHub section would just be noise.
export const NATIVE = new Set(["TITLE", "ASSIGNEES", "LABELS"]);

export const isWritable = (f) => WRITABLE.has(f?.dataType);

export const LABELS = {
  TEXT: "Text",
  NUMBER: "Number",
  DATE: "Date",
  SINGLE_SELECT: "Select",
  ITERATION: "Iteration",
  TITLE: "Title",
  ASSIGNEES: "Assignees",
  LABELS: "Labels",
  MILESTONE: "Milestone",
  REPOSITORY: "Repository",
  REVIEWERS: "Reviewers",
  LINKED_PULL_REQUESTS: "Linked pull requests",
  PARENT_ISSUE: "Parent issue",
  SUB_ISSUES_PROGRESS: "Sub-issues",
  ISSUE_TYPE: "Issue type",
  TRACKED_BY: "Tracked by",
};

// GitHub option colours, mapped to the swatches the board uses.
export const OPTION_COLORS = {
  GRAY: "#8b949e",
  BLUE: "#388bfd",
  GREEN: "#3fb950",
  YELLOW: "#d29922",
  ORANGE: "#db6d28",
  RED: "#f85149",
  PINK: "#db61a2",
  PURPLE: "#a371f7",
};

// blankValue builds the value shape the Go side expects for a field.
export const blankValue = (field) => ({
  fieldId: field.id,
  name: field.name,
  dataType: field.dataType,
  text: "",
  number: null,
  date: "",
  optionId: "",
  optionKey: "",
  iterationId: "",
});

// summarize renders any field value as a short string for the card.
export const summarize = (value) => {
  if (!value) return "";
  if (value.display) return value.display;
  if (value.optionKey) return value.optionKey;
  if (value.text) return value.text;
  if (value.date) return value.date;
  if (value.number !== null && value.number !== undefined)
    return String(value.number);
  return "";
};
