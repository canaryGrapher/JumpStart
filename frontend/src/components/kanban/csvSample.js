import { buildSampleCSV, CSV_HEADER, TASK_FIELD_DOCS } from "../../taskFields.js";

// Sample CSV + short guide for bulk task import. Columns come from
// taskFields.js (kept in sync with internal/taskcsv.Header).

export const SAMPLE_CSV_FILENAME = "jumpstart-tasks-import-sample.csv";

export const SAMPLE_CSV = buildSampleCSV();

export const SAMPLE_GUIDE = [
  "How to bulk-import tasks",
  "",
  "1. Download the sample CSV and open it in Numbers, Excel, or Google Sheets.",
  "2. Keep the header row exactly as written. Fill one task per row.",
  `3. Columns (in order): ${CSV_HEADER.join(", ")}.`,
  "4. Leave id blank for new tasks (JumpStart assigns ids on import).",
  ...TASK_FIELD_DOCS.filter((f) => f.key !== "id").map(
    (f, i) => `${i + 5}. ${f.key}: ${f.note}`
  ),
  `${TASK_FIELD_DOCS.length + 4}. Save as .csv, then use Import → Add (merge) or Replace (full board).`,
].join("\n");

export function downloadSampleCSV() {
  const blob = new Blob([SAMPLE_CSV], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = SAMPLE_CSV_FILENAME;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export function downloadSampleGuide() {
  const blob = new Blob([SAMPLE_GUIDE + "\n\n---\n\n" + SAMPLE_CSV], {
    type: "text/plain;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "jumpstart-bulk-import-guide.txt";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
