// Sample CSV + short guide for bulk task import. Keep columns aligned with
// internal/taskcsv.Header so a filled-in download round-trips cleanly.

export const SAMPLE_CSV_FILENAME = "jumpstart-tasks-import-sample.csv";

export const SAMPLE_CSV = [
  "id,title,type,status,priority,description,assignee,labels,storyPoints,parentId,sprintId,sprint,done,subtasks,acceptance,createdAt,updatedAt,milestone,issueType,parentKey,reviewers,linkedPrs",
  ',Set up CI pipeline,task,todo,high,"Wire GitHub Actions for build + test",,"ci,infra",3,,,Sprint 1,false,"Install runners|Add workflow",Must pass on main,,,,,,,,,,,,,,,',
  ',Fix login redirect,bug,inprogress,medium,Session cookie drops on refresh,alice,bug,2,,,Sprint 1,false,,Redirects to dashboard after login,,,,,,,,,,,,,,,',
  ',User onboarding story,story,backlog,low,First-run experience for new accounts,,ux,5,,,,"",false,Welcome modal|Sample project,User completes first project in <10m,,,,,,,,,,,,,,,',
  ',Empty row template,task,todo,,,,"",,,,,,,,false,,,,,,,,,,,,,,,,',
].join("\n");

export const SAMPLE_GUIDE = [
  "How to bulk-import tasks",
  "",
  "1. Download the sample CSV and open it in Numbers, Excel, or Google Sheets.",
  "2. Keep the header row. Fill one task per row.",
  "3. Leave id blank for new tasks (JumpStart assigns ids on import).",
  "4. status: backlog | todo | inprogress | done",
  "5. type: task | bug | story",
  "6. priority: low | medium | high (optional)",
  "7. labels: comma-separated tags inside the cell (e.g. ci,infra)",
  "8. sprint: sprint name — unknown names create the sprint locally",
  "9. subtasks / acceptance: pipe-separated checklist items (Title|Next)",
  "10. Save as .csv, then use Import → Add (merge) or Replace (full board).",
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
