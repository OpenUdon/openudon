"use strict";

const normalInterval = 2000;
const maximumBackoff = 30000;
let revision = "";
let timer = null;
let failures = 0;
let cachedPayload = null;

const show = (id, value) => {
  document.getElementById(id).textContent = value ?? "—";
};

const schedule = (delay) => {
  clearTimeout(timer);
  if (!document.hidden) {
    timer = setTimeout(refreshSnapshot, delay);
  }
};

const render = (payload, refreshedAt) => {
  cachedPayload = payload;
  revision = payload.revision;
  const snapshot = payload.snapshot;
  const preview = snapshot.preview;
  const externallyModified = Boolean(payload.workspace?.externally_modified);
  show("connection", externallyModified ? "Connected · restart required" : "Connected · polling every 2 seconds");
  show("example", payload.workspace?.example_dir);
  show("project", preview?.project_path || "Not rendered yet");
  show("intent", preview?.intent_path || "Not rendered yet");
  show("revision", payload.revision);
  show("refreshed", refreshedAt.toLocaleTimeString());
  show("sources", `${snapshot.selected_sources?.length || 0} selected`);
  show("actions", `${snapshot.proposed_file_actions?.length || 0} proposed`);
  show("ready", snapshot.ready ? "Yes" : `No (${snapshot.readiness?.length || 0} issues)`);
  show("issue", snapshot.top_issue ? `${snapshot.top_issue.code}: ${snapshot.top_issue.message}` : "None");
  show("frontier", `${snapshot.frontier?.length || 0} questions`);
  show("preview", preview ? (preview.incomplete ? "Incomplete draft" : "Final available") : "Unavailable");
  show("completed", payload.completed ? "Frozen after approved write" : "No");
  show("workspace", externallyModified ? "Externally modified" : "Unchanged");
  document.getElementById("workspace-warning").hidden = !externallyModified;
  show("snapshot", JSON.stringify(snapshot, null, 2));
};

async function refreshSnapshot() {
  if (document.hidden) return;
  try {
    const headers = { Accept: "application/json" };
    if (revision) headers["If-None-Match"] = `"${revision}"`;
    const response = await fetch("api/v2/snapshot", {
      credentials: "same-origin",
      headers,
    });
    const refreshedAt = new Date();
    if (response.status === 304) {
      failures = 0;
      show("connection", cachedPayload?.workspace?.externally_modified ? "Connected · restart required" : "Connected · no changes");
      show("refreshed", refreshedAt.toLocaleTimeString());
      schedule(normalInterval);
      return;
    }
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload?.error?.message || `Snapshot request failed (${response.status})`);
    }
    failures = 0;
    render(payload, refreshedAt);
    schedule(normalInterval);
  } catch (error) {
    failures += 1;
    const delay = Math.min(maximumBackoff, normalInterval * (2 ** Math.min(failures - 1, 4)));
    show("connection", `Refresh failed · retrying in ${Math.round(delay / 1000)}s`);
    if (!cachedPayload) show("snapshot", error.message);
    schedule(delay);
  }
}

document.addEventListener("visibilitychange", () => {
  if (document.hidden) {
    clearTimeout(timer);
    return;
  }
  refreshSnapshot();
});

refreshSnapshot();
