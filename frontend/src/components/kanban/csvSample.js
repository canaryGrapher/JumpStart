import { SaveTextFile } from "../../api";
import { buildSampleCSV, CSV_HEADER, TASK_FIELD_DOCS } from "../../taskFields.js";

// Sample CSV + short guide for bulk task import. Columns come from
// taskFields.js (kept in sync with internal/taskcsv.Header).
// Saves go through the native Save dialog (Wails SaveFileDialog) so the
// user picks the destination — WebView <a download> does not land in
// ~/Downloads reliably in the desktop app.

export const SAMPLE_CSV_FILENAME = "jumpstart-tasks-import-sample.csv";
export const SAMPLE_GUIDE_FILENAME = "jumpstart-bulk-import-guide.txt";

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

// Returns the saved path, or "" / falsy if the user cancelled.
export function downloadSampleCSV() {
  return SaveTextFile(SAMPLE_CSV_FILENAME, SAMPLE_CSV, "Save sample CSV");
}

export function downloadSampleGuide() {
  return SaveTextFile(
    SAMPLE_GUIDE_FILENAME,
    SAMPLE_GUIDE + "\n\n---\n\n" + SAMPLE_CSV,
    "Save sample guide"
  );
}
