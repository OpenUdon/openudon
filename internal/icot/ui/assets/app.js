"use strict";

const show = (id, value) => {
  document.getElementById(id).textContent = value ?? "—";
};

async function loadSnapshot() {
  const response = await fetch("api/v1/snapshot", {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  });
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload?.error?.message || `Snapshot request failed (${response.status})`);
  }
  const snapshot = payload.snapshot;
  const preview = snapshot.preview;
  show("connection", `Connected · revision ${payload.revision}`);
  show("example", payload.workspace?.example_dir);
  show("project", preview?.project_path || "Not rendered yet");
  show("intent", preview?.intent_path || "Not rendered yet");
  show("ready", snapshot.ready ? "Yes" : `No (${snapshot.readiness?.length || 0} issues)`);
  show("issue", snapshot.top_issue ? `${snapshot.top_issue.code}: ${snapshot.top_issue.message}` : "None");
  show("frontier", `${snapshot.frontier?.length || 0} questions`);
  show("preview", preview ? (preview.incomplete ? "Incomplete draft" : "Final available") : "Unavailable");
  show("completed", payload.completed ? "Frozen after approved write" : "No");
  show("snapshot", JSON.stringify(snapshot, null, 2));
}

loadSnapshot().catch((error) => {
  show("connection", "Could not load the local snapshot");
  show("snapshot", error.message);
});
