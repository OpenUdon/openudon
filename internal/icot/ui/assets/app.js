"use strict";

const normalInterval = 2000;
const maximumBackoff = 30000;

const state = {
  renderedPayload: null,
  latestPayload: null,
  serverRevision: "",
  dirty: false,
  pendingMutation: false,
  reconciliationRequired: false,
  unresolvedMutation: false,
  retryRequest: null,
  timer: null,
  failures: 0,
  pollGeneration: 0,
};

const byID = (id) => document.getElementById(id);

const showText = (id, value) => {
  byID(id).textContent = value ?? "—";
};

const showStatus = (id, value, tone = "") => {
  showText(id, value);
  const item = byID(id).closest(".status-item");
  if (!item) return;
  item.classList.remove("tone-positive", "tone-warning", "tone-danger", "tone-active");
  if (tone) item.classList.add(`tone-${tone}`);
};

const showConnection = (message, tone = "positive") => {
  showText("connection", message);
  byID("connection").dataset.tone = tone;
};

const announceMutation = (message) => {
  showText("mutation-status", message);
};

const clearNode = (node) => {
  while (node.firstChild) node.removeChild(node.firstChild);
};

const make = (tag, text, className) => {
  const node = document.createElement(tag);
  if (text !== undefined && text !== null) node.textContent = text;
  if (className) node.className = className;
  return node;
};

const countLabel = (count, singular, plural = `${singular}s`) => `${count} ${count === 1 ? singular : plural}`;

const schedule = (delay) => {
  clearTimeout(state.timer);
  if (!document.hidden && !state.pendingMutation) {
    state.timer = setTimeout(refreshSnapshot, delay);
  }
};

const currentSnapshot = () => state.renderedPayload?.snapshot || {};

const externallyModified = (payload = state.renderedPayload) => Boolean(payload?.workspace?.externally_modified);

const mutationLocked = () => Boolean(
  !state.renderedPayload ||
  state.pendingMutation ||
  state.reconciliationRequired ||
  state.unresolvedMutation ||
  state.renderedPayload?.completed ||
  externallyModified()
);

const answerInputs = () => Array.from(document.querySelectorAll("textarea[data-question-id]"));

const collectAnswers = () => answerInputs().map((input) => ({
  question_id: input.dataset.questionId,
  prompt: input.dataset.questionPrompt || input.dataset.questionId,
  value: input.value,
}));

const renderUnsentAnswers = (answers) => {
  const values = answers.filter((answer) => answer.value.trim() !== "");
  if (values.length === 0) return;
  const list = byID("unsent-answers");
  clearNode(list);
  for (const answer of values) {
    const row = make("div");
    row.append(make("dt", answer.prompt));
    row.append(make("dd", answer.value));
    list.append(row);
  }
  byID("unsent-section").hidden = false;
};

const preserveCurrentAnswers = () => {
  if (!state.dirty) return;
  renderUnsentAnswers(collectAnswers());
};

const clearError = () => {
  byID("error-banner").hidden = true;
  byID("retry-mutation").hidden = true;
  byID("error-request-row").hidden = true;
  state.retryRequest = null;
};

const showError = (message, requestID = "", retryRequest = null, focus = true) => {
  showText("error-message", message);
  showText("error-request-id", requestID);
  byID("error-request-row").hidden = requestID === "";
  state.retryRequest = retryRequest;
  byID("retry-mutation").hidden = retryRequest === null;
  byID("error-banner").hidden = false;
  if (focus) byID("error-banner").focus();
};

const showStateWarning = (payload, message, focus = false) => {
  state.latestPayload = payload;
  state.serverRevision = payload.revision;
  state.reconciliationRequired = true;
  showText("state-warning-message", message);
  byID("state-warning").hidden = false;
  if (focus) byID("state-warning").focus();
  updateControls();
};

const hideStateWarning = () => {
  state.reconciliationRequired = false;
  byID("state-warning").hidden = true;
};

const appendEmptyOrItems = (listID, values, emptyText, renderItem) => {
  const list = byID(listID);
  clearNode(list);
  if (!values || values.length === 0) {
    list.append(make("li", emptyText, "empty-state"));
    return;
  }
  values.forEach((value) => list.append(renderItem(value)));
};

const renderFrontier = (snapshot, revision) => {
  const frontier = snapshot.frontier || [];
  const fields = byID("frontier-fields");
  clearNode(fields);
  showText("frontier-revision", revision ? `Revision ${revision}` : "Waiting for a revision");

  for (const [index, question] of frontier.entries()) {
    const wrapper = make("fieldset", null, "question");
    const legend = make("legend", question.prompt || question.id || `Question ${index + 1}`);
    wrapper.append(legend);

    const badges = make("div");
    badges.append(make("span", `Priority ${question.priority || 0}`, "badge"));
    if (question.required) badges.append(make("span", "Required", "badge"));
    if (question.forced) badges.append(make("span", "Forced review", "badge"));
    wrapper.append(badges);

    const inputID = `frontier-answer-${index + 1}`;
    const errorID = `${inputID}-error`;
    const descriptionIDs = [];

    if (question.slots?.length) {
      const node = make("p", `Slots: ${question.slots.join(", ")}`, "question-meta");
      node.id = `${inputID}-slots`;
      descriptionIDs.push(node.id);
      wrapper.append(node);
    }
    if (question.rationale) {
      const node = make("p", `Why this is asked: ${question.rationale}`, "question-meta");
      node.id = `${inputID}-rationale`;
      descriptionIDs.push(node.id);
      wrapper.append(node);
    }
    if (question.recommendation) {
      const node = make("p", `Recommendation: ${question.recommendation}`, "question-meta");
      node.id = `${inputID}-recommendation`;
      descriptionIDs.push(node.id);
      wrapper.append(node);
    }

    const label = make("label", `Your answer for: ${question.prompt || question.id || `Question ${index + 1}`}`, "sr-only");
    label.htmlFor = inputID;
    wrapper.append(label);
    const textarea = make("textarea");
    textarea.id = inputID;
    textarea.name = question.id || inputID;
    textarea.required = true;
    textarea.dataset.questionId = question.id || "";
    textarea.dataset.questionPrompt = question.prompt || question.id || `Question ${index + 1}`;
    textarea.placeholder = "Write your answer…";
    textarea.setAttribute("aria-required", "true");
    descriptionIDs.push(errorID);
    textarea.setAttribute("aria-describedby", descriptionIDs.join(" "));
    wrapper.append(textarea);

    const error = make("p", "", "field-error");
    error.id = errorID;
    error.hidden = true;
    wrapper.append(error);

    if (question.recommendation) {
      const actions = make("div", null, "question-actions");
      const recommendation = make("button", "Use recommendation");
      recommendation.type = "button";
      recommendation.setAttribute("aria-label", `Use recommendation for ${question.prompt || question.id}`);
      recommendation.addEventListener("click", () => {
        textarea.value = question.recommendation;
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
        textarea.focus();
      });
      actions.append(recommendation);
      wrapper.append(actions);
    }
    fields.append(wrapper);
  }

  byID("round-form").hidden = frontier.length === 0;
  byID("frontier-empty").hidden = frontier.length !== 0;
};

const renderReadiness = (snapshot) => {
  const issues = snapshot.readiness || [];
  appendEmptyOrItems("readiness-list", issues, "No readiness issues.", (issue) => {
    const item = make("li");
    item.className = issue.severity === "blocking" ? "issue-blocking" : issue.severity === "warning" ? "issue-warning" : "";
    const title = make("strong", `${issue.severity || "issue"}: ${issue.code || "unspecified"}`);
    item.append(title, document.createTextNode(` — ${issue.message || "No detail provided."}`));
    if (issue.slot) item.append(make("div", `Slot: ${issue.slot}`, "question-meta"));
    if (issue.suggested_answer) item.append(make("div", `Suggested answer: ${issue.suggested_answer}`, "question-meta"));
    return item;
  });
  const panel = byID("readiness-panel");
  panel.classList.toggle("is-clear", issues.length === 0);
  panel.classList.toggle("has-danger", issues.some((issue) => issue.severity === "blocking"));
  panel.classList.toggle("has-warning", issues.length > 0 && !issues.some((issue) => issue.severity === "blocking"));
};

const renderSources = (snapshot) => {
  appendEmptyOrItems("sources-list", snapshot.selected_sources || [], "No sources selected.", (source) => {
    const target = source.target_path ? ` → ${source.target_path}` : "";
    const item = make("li");
    item.append(make("strong", `${source.kind || "source"}: ${source.id || "unnamed"}`));
    item.append(document.createTextNode(target));
    return item;
  });
};

const renderActions = (snapshot) => {
  const body = byID("actions-body");
  clearNode(body);
  const actions = snapshot.proposed_file_actions || [];
  for (const action of actions) {
    const row = make("tr");
    const actionName = action.action || "write";
    const actionCell = make("td", null, "action-cell");
    actionCell.dataset.label = "Action";
    const actionTone = actionName.startsWith("remove") ? "action-remove" : actionName === "write" ? "action-write" : "";
    actionCell.append(make("span", actionName, `action-badge ${actionTone}`.trim()));
    const path = make("td");
    path.dataset.label = "Path";
    path.append(make("code", action.path || ""));
    const reason = make("td", action.reason || "—");
    reason.dataset.label = "Reason";
    row.append(actionCell, path, reason);
    body.append(row);
  }
  byID("actions-empty").hidden = actions.length !== 0;
  byID("actions-body").parentElement.parentElement.hidden = actions.length === 0;
};

const renderConflicts = (snapshot) => {
  const conflicts = snapshot.write_conflicts || [];
  appendEmptyOrItems("conflicts-list", conflicts, "No accepted-baseline write conflicts.", (conflict) => {
    const item = make("li", `${conflict.action || "write"}: `, "issue-warning");
    item.append(make("code", conflict.path || ""));
    item.append(document.createTextNode(" requires explicit overwrite authorization."));
    return item;
  });
  byID("conflicts-panel").classList.toggle("is-clear", conflicts.length === 0);
  byID("conflicts-panel").classList.toggle("has-warning", conflicts.length > 0);
};

const renderPreview = (snapshot) => {
  const preview = snapshot.preview;
  showText("project-preview-path", preview?.project_path || "Not rendered yet");
  showText("intent-preview-path", preview?.intent_path || "Not rendered yet");
  showText("project-preview", preview?.project_md || "Preview unavailable.");
  showText("intent-preview", preview?.intent_hcl || "Preview unavailable.");
};

const renderWriteResult = (payload) => {
  const result = payload.write_result;
  byID("write-result-section").hidden = !result;
  if (!result) return;
  appendEmptyOrItems("written-list", result.written || [], "No paths written.", (path) => {
    const item = make("li");
    item.append(make("code", path));
    return item;
  });
  appendEmptyOrItems("removed-list", result.removed || [], "No paths removed.", (path) => {
    const item = make("li");
    item.append(make("code", path));
    return item;
  });
  const warnings = result.cleanup_warnings || [];
  byID("cleanup-warning-section").hidden = warnings.length === 0;
  appendEmptyOrItems("cleanup-warning-list", warnings, "No cleanup warnings.", (warning) => make("li", warning));
};

const updateControls = () => {
  const locked = mutationLocked();
  const snapshot = currentSnapshot();
  const preview = snapshot.preview;
  const conflicts = snapshot.write_conflicts || [];
  const reviewed = byID("review-confirmed").checked;
  const overwriteAllowed = conflicts.length === 0 || byID("allow-overwrite").checked;

  answerInputs().forEach((input) => { input.disabled = locked; });
  document.querySelectorAll(".question-actions button").forEach((button) => { button.disabled = locked; });
  byID("round-submit").disabled = locked || (snapshot.frontier || []).length === 0;

  byID("review-confirmed").disabled = locked || !snapshot.approval_required;
  byID("allow-overwrite").disabled = locked || conflicts.length === 0;
  byID("overwrite-row").hidden = conflicts.length === 0 || Boolean(state.renderedPayload?.completed);

  const finalAvailable = Boolean(snapshot.approval_required && preview && !preview.incomplete && snapshot.ready);
  const incompleteAvailable = Boolean(snapshot.approval_required && preview?.incomplete);
  byID("approve-final").hidden = !finalAvailable;
  byID("approve-incomplete").hidden = !incompleteAvailable;
  byID("approve-final").disabled = locked || !reviewed || !overwriteAllowed || !finalAvailable;
  byID("approve-incomplete").disabled = locked || !reviewed || !overwriteAllowed || !incompleteAvailable;

  let approvalState = "No proposal is currently available for approval.";
  if (state.renderedPayload?.completed) approvalState = "This approved session is frozen.";
  else if (externallyModified()) approvalState = "Restart before attempting another mutation.";
  else if (state.reconciliationRequired) approvalState = "Review the latest revision before continuing.";
  else if (state.unresolvedMutation) approvalState = "Reconciliation must succeed before continuing.";
  else if (snapshot.approval_required && !reviewed) approvalState = "Confirm that you reviewed the proposal to enable approval.";
  else if (conflicts.length > 0 && !overwriteAllowed) approvalState = "Explicitly authorize the listed overwrites to enable approval.";
  else if (finalAvailable) approvalState = "The final proposal is ready for explicit approval.";
  else if (incompleteAvailable) approvalState = "Only explicit incomplete-draft approval is available.";
  showText("approval-state", approvalState);
};

const renderPayload = (payload, refreshedAt = new Date()) => {
  state.renderedPayload = payload;
  state.latestPayload = payload;
  state.serverRevision = payload.revision;
  state.dirty = false;
  state.unresolvedMutation = false;
  hideStateWarning();

  const snapshot = payload.snapshot || {};
  const preview = snapshot.preview;
  const drifted = externallyModified(payload);
  showConnection(
    drifted ? "Connected · restart required" : payload.completed ? "Connected · session frozen" : "Connected · polling every 2 seconds",
    drifted ? "warning" : "positive",
  );
  showText("example", payload.workspace?.example_dir);
  showText("project", preview?.project_path || "Not rendered yet");
  showText("intent", preview?.intent_path || "Not rendered yet");
  showText("revision", payload.revision);
  showText("refreshed", refreshedAt.toLocaleTimeString());
  const sourceCount = snapshot.selected_sources?.length || 0;
  const actionCount = snapshot.proposed_file_actions?.length || 0;
  const conflictCount = snapshot.write_conflicts?.length || 0;
  const frontierCount = snapshot.frontier?.length || 0;
  showStatus("sources", countLabel(sourceCount, "selected source"));
  showStatus("actions", countLabel(actionCount, "proposed action"), actionCount > 0 ? "active" : "");
  showStatus("conflicts", countLabel(conflictCount, "write conflict"), conflictCount > 0 ? "danger" : "positive");
  showStatus("ready", snapshot.ready ? "Yes" : `No (${countLabel(snapshot.readiness?.length || 0, "issue")})`, snapshot.ready ? "positive" : "warning");
  showText("issue", snapshot.top_issue ? `${snapshot.top_issue.code}: ${snapshot.top_issue.message}` : "None");
  byID("issue-summary").classList.toggle("has-issue", Boolean(snapshot.top_issue));
  showStatus("frontier", countLabel(frontierCount, "question"), frontierCount > 0 ? "active" : "");
  showStatus("preview", preview ? (preview.incomplete ? "Incomplete draft" : "Final available") : "Unavailable", preview ? (preview.incomplete ? "warning" : "positive") : "");
  showStatus("completed", payload.completed ? "Frozen after approved write" : "No", payload.completed ? "positive" : "");
  showStatus("workspace", drifted ? "Externally modified" : "Unchanged", drifted ? "danger" : "positive");

  const reviewState = payload.completed ? "complete" : snapshot.approval_required ? "ready" : "working";
  byID("review-section").dataset.state = reviewState;
  showText("review-status", payload.completed ? "Approved · frozen" : snapshot.approval_required ? (preview?.incomplete ? "Draft review available" : "Ready for approval") : "Draft in progress");

  byID("workspace-warning").hidden = !drifted;
  byID("completion-banner").hidden = !payload.completed;
  byID("review-confirmed").checked = false;
  byID("allow-overwrite").checked = false;
  renderFrontier(snapshot, payload.revision);
  renderReadiness(snapshot);
  renderSources(snapshot);
  renderActions(snapshot);
  renderConflicts(snapshot);
  renderPreview(snapshot);
  renderWriteResult(payload);
  showText("snapshot", JSON.stringify(snapshot, null, 2));
  updateControls();
};

const receivePayload = (payload, refreshedAt = new Date(), focusStateWarning = false) => {
  state.serverRevision = payload.revision;
  state.latestPayload = payload;

  if (payload.completed || externallyModified(payload)) {
    preserveCurrentAnswers();
    renderPayload(payload, refreshedAt);
    if (externallyModified(payload)) byID("workspace-warning").focus();
    return;
  }

  if (state.renderedPayload && state.dirty && payload.revision !== state.renderedPayload.revision) {
    showText("refreshed", refreshedAt.toLocaleTimeString());
    showConnection("Connected · newer revision requires review", "warning");
    showStateWarning(
      payload,
      `The workspace advanced from ${state.renderedPayload.revision} to ${payload.revision}. Your unsent answers remain in the old form and will not be rebased or submitted automatically.`,
      focusStateWarning,
    );
    return;
  }

  renderPayload(payload, refreshedAt);
};

const decodePayload = async (response) => {
  try {
    return await response.json();
  } catch (_) {
    return null;
  }
};

async function fetchSnapshot(force = false) {
  const headers = { Accept: "application/json" };
  if (!force && state.serverRevision) headers["If-None-Match"] = `"${state.serverRevision}"`;
  const response = await fetch("api/v2/snapshot", { credentials: "same-origin", headers });
  if (response.status === 304) return { unchanged: true, refreshedAt: new Date() };
  const payload = await decodePayload(response);
  if (!response.ok) {
    const failure = new Error(payload?.error?.message || `Snapshot request failed (${response.status})`);
    failure.payload = payload;
    throw failure;
  }
  return { payload, refreshedAt: new Date() };
}

async function refreshSnapshot() {
  if (document.hidden || state.pendingMutation) return;
  const generation = ++state.pollGeneration;
  try {
    const result = await fetchSnapshot(false);
    if (generation !== state.pollGeneration || state.pendingMutation) return;
    state.failures = 0;
    if (result.unchanged) {
      showConnection(
        externallyModified() ? "Connected · restart required" : state.renderedPayload?.completed ? "Connected · session frozen" : "Connected · no changes",
        externallyModified() ? "warning" : "positive",
      );
      showText("refreshed", result.refreshedAt.toLocaleTimeString());
    } else {
      receivePayload(result.payload, result.refreshedAt);
    }
    schedule(normalInterval);
  } catch (error) {
    if (generation !== state.pollGeneration || state.pendingMutation) return;
    state.failures += 1;
    const delay = Math.min(maximumBackoff, normalInterval * (2 ** Math.min(state.failures - 1, 4)));
    showConnection(`Refresh failed · retrying in ${Math.round(delay / 1000)} seconds`, "danger");
    if (!state.renderedPayload) showText("snapshot", error.message);
    schedule(delay);
  }
}

const reconcileMutationFailure = async (request, failure, retryable) => {
  state.unresolvedMutation = true;
  updateControls();
  try {
    const result = await fetchSnapshot(true);
    if (result.payload) {
      const unchanged = result.payload.revision === request.body.revision && !result.payload.completed && !externallyModified(result.payload);
      if (unchanged) {
        state.latestPayload = result.payload;
        state.serverRevision = result.payload.revision;
        showText("refreshed", result.refreshedAt.toLocaleTimeString());
        showConnection("Connected · mutation failed, state reconciled", "warning");
        state.unresolvedMutation = false;
        if (retryable) {
          showError(`${failure.message} The current revision is unchanged; retry only after reviewing the request.`, failure.requestID, request);
        } else {
          state.unresolvedMutation = true;
          showError(`${failure.message} The mutation outcome is indeterminate. Restart and inspect the workspace before making another change.`, failure.requestID);
        }
      } else {
        receivePayload(result.payload, result.refreshedAt, true);
        state.unresolvedMutation = false;
        if (!result.payload.completed && !externallyModified(result.payload) && !state.reconciliationRequired) {
          showError(`${failure.message} The workspace now has a different revision; inspect it before trying another mutation.`, failure.requestID);
        }
      }
    }
  } catch (error) {
    showError(`${failure.message} Reconciliation also failed: ${error.message}. Mutations remain blocked until inspection succeeds.`, failure.requestID);
  }
  updateControls();
};

const handleMutationFailure = async (request, status, payload) => {
  const error = payload?.error || {};
  const failure = {
    code: error.code || "network_error",
    message: error.message || `Authoring request failed (${status || "network"}).`,
    requestID: error.request_id || "",
    retryable: status === 0 || Boolean(error.retryable),
  };
  announceMutation(`${request.route === "approve" ? "Approval" : "Round submission"} failed. Review the error before continuing.`);

  if (failure.code === "engine_rejected" || status === 422) {
    showError(failure.message, failure.requestID);
    return;
  }
  if (failure.code === "workspace_changed" || failure.code === "session_frozen" || failure.code === "stale_revision" || status >= 500 || status === 0) {
    await reconcileMutationFailure(request, failure, failure.retryable);
    return;
  }
  showError(failure.message, failure.requestID);
};

async function sendMutation(request) {
  if (mutationLocked()) return;
  let successFocusID = "";
  clearTimeout(state.timer);
  state.pollGeneration += 1;
  clearError();
  state.pendingMutation = true;
  announceMutation(request.route === "approve" ? "Approval is being committed." : "Authoring round is being submitted.");
  updateControls();
  try {
    const response = await fetch(`api/v2/${request.route}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(request.body),
    });
    const payload = await decodePayload(response);
    if (!response.ok) {
      await handleMutationFailure(request, response.status, payload);
      return;
    }
    clearError();
    renderPayload(payload, new Date());
    if (request.route === "approve") {
      announceMutation("Approval committed. Authoring is complete and the session is frozen.");
      successFocusID = "completion-banner";
    } else {
      const inputs = answerInputs();
      if (inputs.length > 0) {
        announceMutation("Round submitted. Continue with the next authoring question.");
        successFocusID = inputs[0].id;
      } else if (payload.snapshot?.approval_required) {
        announceMutation("Round submitted. The proposal is ready for review.");
        successFocusID = "review-heading";
      } else {
        announceMutation("Round submitted. Review the updated authoring state.");
        successFocusID = "frontier-heading";
      }
    }
  } catch (error) {
    await handleMutationFailure(request, 0, { error: { code: "network_error", message: error.message, retryable: true } });
  } finally {
    state.pendingMutation = false;
    updateControls();
    if (successFocusID) byID(successFocusID)?.focus();
    schedule(normalInterval);
  }
}

const validateRound = () => {
  let firstInvalid = null;
  for (const input of answerInputs()) {
    const error = byID(`${input.id}-error`);
    if (input.value.trim() === "") {
      error.textContent = "This frontier question requires an answer before the round can be submitted.";
      error.hidden = false;
      input.setAttribute("aria-invalid", "true");
      if (!firstInvalid) firstInvalid = input;
    } else {
      error.hidden = true;
      input.removeAttribute("aria-invalid");
    }
  }
  if (firstInvalid) firstInvalid.focus();
  return firstInvalid === null;
};

byID("round-form").addEventListener("submit", (event) => {
  event.preventDefault();
  if (mutationLocked() || !validateRound()) return;
  const answers = collectAnswers().map(({ question_id, value }) => ({ question_id, value: value.trim() }));
  sendMutation({ route: "round", body: { revision: state.renderedPayload.revision, answers } });
});

byID("frontier-fields").addEventListener("input", (event) => {
  if (!event.target.matches("textarea[data-question-id]")) return;
  state.dirty = true;
  const error = byID(`${event.target.id}-error`);
  if (event.target.value.trim() !== "") {
    error.hidden = true;
    event.target.removeAttribute("aria-invalid");
  }
  clearError();
  updateControls();
});

byID("approval-form").addEventListener("submit", (event) => {
  event.preventDefault();
  if (mutationLocked()) return;
  const mode = event.submitter?.dataset.approval;
  if (mode !== "final" && mode !== "incomplete") return;
  sendMutation({
    route: "approve",
    body: {
      revision: state.renderedPayload.revision,
      human_approved: true,
      allow_overwrite: byID("allow-overwrite").checked,
      approve_incomplete: mode === "incomplete",
    },
  });
});

byID("review-confirmed").addEventListener("change", () => {
  state.dirty = true;
  clearError();
  updateControls();
});
byID("allow-overwrite").addEventListener("change", () => {
  state.dirty = true;
  clearError();
  updateControls();
});

byID("adopt-latest").addEventListener("click", () => {
  if (!state.latestPayload) return;
  preserveCurrentAnswers();
  clearError();
  renderPayload(state.latestPayload, new Date());
  byID("frontier-heading").focus?.();
});

byID("retry-mutation").addEventListener("click", () => {
  const request = state.retryRequest;
  if (!request || state.serverRevision !== request.body.revision) return;
  state.unresolvedMutation = false;
  clearError();
  updateControls();
  sendMutation(request);
});

document.addEventListener("visibilitychange", () => {
  if (document.hidden) {
    clearTimeout(state.timer);
    return;
  }
  refreshSnapshot();
});

updateControls();
refreshSnapshot();
