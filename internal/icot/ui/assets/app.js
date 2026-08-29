"use strict";

const normalInterval = 2000;
const maximumBackoff = 30000;
const acquisitionRoutes = new Set(["journey", "source/stage", "source/remove", "browser/preflight", "capture/start"]);

const registrationActive = (authoring) => Boolean(authoring && !["review_ready", "adopted", "promoted", "canceled", "failed"].includes(authoring.state));

const state = {
  renderedPayload: null,
  latestPayload: null,
  serverRevision: "",
  etag: "",
  dirty: false,
  pendingMutation: false,
  reconciliationRequired: false,
  unresolvedMutation: false,
  retryRequest: null,
  timer: null,
  failures: 0,
  pollGeneration: 0,
	transactionRevision: "",
	registrationRowsInitialized: false,
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
	state.renderedPayload?.lifecycle !== "authoring" ||
	state.renderedPayload?.capture?.containment_failed ||
	(state.renderedPayload?.capture && !["staged", "canceled", "failed"].includes(state.renderedPayload.capture.state)) ||
	state.renderedPayload?.registration_authoring?.containment_failed ||
	registrationActive(state.renderedPayload?.registration_authoring) ||
  externallyModified()
);

const acquisitionLocked = () => mutationLocked() || state.dirty;

const answerInputs = () => Array.from(document.querySelectorAll(".question-answer[data-question-id]"));

const questionWrapper = (input) => input.closest(".question");

const deferralFields = (wrapper) => Array.from(wrapper?.querySelectorAll(".deferral-field") || []);

const collectAnswers = () => answerInputs().map((input) => {
  const wrapper = questionWrapper(input);
  const toggle = wrapper?.querySelector(".deferral-toggle");
  const base = {
    question_id: input.dataset.questionId,
    prompt: input.dataset.questionPrompt || input.dataset.questionId,
    value: input.value,
  };
  if (!toggle?.checked) return base;
  const deferral = {};
  for (const field of deferralFields(wrapper)) deferral[field.dataset.deferralField] = field.value;
  return {
    ...base,
    value: "",
    deferral,
    display_value: `Deferred — owner: ${deferral.owner || "—"}; impact: ${deferral.impact || "—"}; unblock condition: ${deferral.unblock_condition || "—"}; next action: ${deferral.suggested_next_action || "—"}`,
  };
});

const renderUnsentAnswers = (answers) => {
  const values = answers.filter((answer) => (answer.display_value || answer.value || "").trim() !== "");
  if (values.length === 0) return;
  const list = byID("unsent-answers");
  clearNode(list);
  for (const answer of values) {
    const row = make("div");
    row.append(make("dt", answer.prompt));
    row.append(make("dd", answer.display_value || answer.value));
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

const showQuestionError = (questionID, message) => {
  if (!questionID) return "";
  const input = answerInputs().find((candidate) => candidate.dataset.questionId === questionID);
  if (!input) return "";
  const error = byID(`${input.id}-error`);
  error.textContent = message;
  error.hidden = false;
  const wrapper = questionWrapper(input);
  const toggle = wrapper?.querySelector(".deferral-toggle");
  const focusTarget = toggle?.checked ? deferralFields(wrapper)[0] : input;
  focusTarget?.setAttribute("aria-invalid", "true");
  if (!focusTarget?.disabled) focusTarget?.focus();
  return focusTarget?.id || input.id;
};

const clearQuestionError = (wrapper) => {
  const input = wrapper?.querySelector(".question-answer");
  if (!input) return;
  const error = byID(`${input.id}-error`);
  error.hidden = true;
  for (const control of [input, ...deferralFields(wrapper)]) control.removeAttribute("aria-invalid");
};

const updateQuestionState = (wrapper, locked = mutationLocked()) => {
  const input = wrapper.querySelector(".question-answer");
  const toggle = wrapper.querySelector(".deferral-toggle");
  const panel = wrapper.querySelector(".deferral-panel");
  const deferred = Boolean(toggle?.checked);
  input.disabled = locked || deferred;
  input.required = !deferred;
  input.setAttribute("aria-required", String(!deferred));
  if (toggle) toggle.disabled = locked;
  if (panel) panel.hidden = !deferred;
  for (const field of deferralFields(wrapper)) {
    field.disabled = locked || !deferred;
    field.required = deferred;
    field.setAttribute("aria-required", String(deferred));
  }
  const recommendation = wrapper.querySelector(".question-actions button");
  if (recommendation) recommendation.disabled = locked || deferred;
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
  const controls = new Map((snapshot.question_controls || []).map((control) => [control.question_id, control]));
  const fields = byID("frontier-fields");
  clearNode(fields);
  showText("frontier-revision", revision ? `Revision ${revision}` : "Waiting for a revision");

  for (const [index, question] of frontier.entries()) {
    const wrapper = make("fieldset", null, "question");
    wrapper.dataset.questionId = question.id || "";
    const control = controls.get(question.id) || { input_kind: "text", options: [], deferrable: false };
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
    if (question.evidence_refs?.length) {
      const node = make("p", `Evidence: ${question.evidence_refs.join(", ")}`, "question-meta");
      node.id = `${inputID}-evidence`;
      descriptionIDs.push(node.id);
      wrapper.append(node);
    }
    if (question.recommendation) {
      const node = make("p", `Recommendation: ${question.recommendation}`, "question-meta");
      node.id = `${inputID}-recommendation`;
      descriptionIDs.push(node.id);
      wrapper.append(node);
    }
    if (control.syntax) {
      const node = make("p", `Format: ${control.syntax}`, "question-meta question-syntax");
      node.id = `${inputID}-syntax`;
      descriptionIDs.push(node.id);
      wrapper.append(node);
    }

    const label = make("label", `Your answer for: ${question.prompt || question.id || `Question ${index + 1}`}`, "sr-only");
    label.htmlFor = inputID;
    wrapper.append(label);
    let input;
    if (control.input_kind === "choice" && control.options?.length) {
      input = make("select", null, "question-answer");
      const placeholder = make("option", "Choose an answer…");
      placeholder.value = "";
      input.append(placeholder);
      for (const option of control.options) {
        const choice = make("option", option.label || option.value);
        choice.value = option.value || "";
        input.append(choice);
      }
    } else {
      input = make("textarea", null, "question-answer");
      input.placeholder = "Write your answer…";
    }
    input.id = inputID;
    input.name = question.id || inputID;
    input.required = true;
    input.dataset.questionId = question.id || "";
    input.dataset.questionPrompt = question.prompt || question.id || `Question ${index + 1}`;
    input.setAttribute("aria-required", "true");
    descriptionIDs.push(errorID);
    input.setAttribute("aria-describedby", descriptionIDs.join(" "));
    wrapper.append(input);

    const error = make("p", "", "field-error");
    error.id = errorID;
    error.hidden = true;
    wrapper.append(error);

    if (control.deferrable) {
      const toggleID = `${inputID}-defer`;
      const toggleRow = make("label", null, "check-row deferral-toggle-row");
      toggleRow.htmlFor = toggleID;
      const toggle = make("input", null, "deferral-toggle");
      toggle.id = toggleID;
      toggle.type = "checkbox";
      toggleRow.append(toggle, make("span", "Defer this decision with an explicit owner and unblock plan."));
      wrapper.append(toggleRow);

      const panel = make("fieldset", null, "deferral-panel");
      panel.hidden = true;
      panel.append(make("legend", "Deferral details"));
      const deferralLabels = [
        ["owner", "Owner"],
        ["impact", "Impact of deferring"],
        ["unblock_condition", "Unblock condition"],
        ["suggested_next_action", "Suggested next action"],
      ];
      for (const [fieldName, fieldLabel] of deferralLabels) {
        const fieldID = `${inputID}-deferral-${fieldName.replaceAll("_", "-")}`;
        const fieldWrapper = make("div", null, "deferral-field-row");
        const fieldLabelNode = make("label", fieldLabel);
        fieldLabelNode.htmlFor = fieldID;
        const field = make("input", null, "deferral-field");
        field.id = fieldID;
        field.type = "text";
        field.required = true;
        field.disabled = true;
        field.dataset.deferralField = fieldName;
        field.setAttribute("aria-describedby", errorID);
        fieldWrapper.append(fieldLabelNode, field);
        panel.append(fieldWrapper);
      }
      wrapper.append(panel);
    }

    const recommendationMatchesControl = control.input_kind !== "choice" || control.options?.some((option) => option.value === question.recommendation);
    if (question.recommendation && recommendationMatchesControl) {
      const actions = make("div", null, "question-actions");
      const recommendation = make("button", "Use recommendation");
      recommendation.type = "button";
      recommendation.setAttribute("aria-label", `Use recommendation for ${question.prompt || question.id}`);
      recommendation.addEventListener("click", () => {
        if (input.value.trim() !== "") {
          input.focus();
          return;
        }
        input.value = question.recommendation;
        input.dispatchEvent(new Event("input", { bubbles: true }));
        input.focus();
      });
      actions.append(recommendation);
      wrapper.append(actions);
    }
    fields.append(wrapper);
    updateQuestionState(wrapper, mutationLocked());
  }

  byID("round-form").hidden = frontier.length === 0;
  byID("frontier-empty").hidden = frontier.length !== 0;
};

const renderRevisableDecisions = (snapshot) => {
  const decisions = snapshot.revisable_decisions || [];
  const list = byID("revisions-list");
  clearNode(list);
  for (const decision of decisions) {
    const item = make("section", null, "revision-item");
    item.append(make("h4", decision.prompt || decision.question_id || "Settled answer"));
    const value = make("p", null, "revision-value");
    value.append(make("strong", "Current answer: "), document.createTextNode(decision.value || "—"));
    item.append(value);
    if (decision.slots?.length) item.append(make("p", `Slots: ${decision.slots.join(", ")}`, "question-meta"));
    item.append(make("p", decision.impact || "Reopening this answer will re-run authoring readiness.", "revision-impact"));
    const button = make("button", "Reopen answer", "reopen-decision");
    button.type = "button";
    button.dataset.questionId = decision.question_id || "";
    button.setAttribute("aria-label", `Reopen answer for ${decision.prompt || decision.question_id}`);
    button.addEventListener("click", () => {
      if (mutationLocked() || (currentSnapshot().frontier || []).length > 0) return;
      sendMutation({
        route: "reopen",
        body: { revision: state.renderedPayload.revision, question_id: decision.question_id },
      });
    });
    item.append(button);
    list.append(item);
  }
  byID("revisions-empty").hidden = decisions.length !== 0;
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

const renderCandidateWorkflows = (snapshot) => {
  appendEmptyOrItems("candidate-workflows-list", snapshot.candidate_workflows || [], "No deferred candidate workflows.", (candidate) => {
    const item = make("li");
    item.append(make("strong", candidate.title || "Untitled candidate"));
    if (candidate.outcome) item.append(document.createTextNode(` — ${candidate.outcome}`));
    if (candidate.deferral_reason) item.append(make("div", `Deferred: ${candidate.deferral_reason}`, "question-meta"));
    if (candidate.promotion_trigger) item.append(make("div", `Promotion trigger: ${candidate.promotion_trigger}`, "question-meta"));
    return item;
  });
};

const renderDecisionEvidence = (snapshot) => {
  appendEmptyOrItems("decision-evidence-list", snapshot.evidence || [], "No decision evidence has been recorded.", (evidence) => {
    const item = make("li");
    item.append(make("strong", evidence.kind || "evidence"));
    item.append(document.createTextNode(` — ${evidence.summary || "No summary provided."}`));
    if (evidence.source) item.append(make("div", `Source: ${evidence.source}`, "question-meta"));
    if (evidence.references?.length) item.append(make("div", `References: ${evidence.references.join(", ")}`, "question-meta"));
    return item;
  });
};

const renderSourceEvidence = (snapshot) => {
  const sourceCandidates = snapshot.source_candidates || {};
  const entries = [];
  for (const candidate of sourceCandidates.local?.candidates || []) {
    entries.push({ tone: "candidate", title: `Local ${candidate.kind || "API"}: ${candidate.title || candidate.id || "unnamed"}`, detail: `${candidate.operation_count || 0} reviewed operations · ${candidate.provenance || "local discovery"}` });
  }
  for (const candidate of sourceCandidates.browser?.candidates || []) {
    entries.push({ tone: "candidate", title: `Browser profile: ${candidate.title || candidate.id || "unnamed"}`, detail: `${candidate.action_count || 0} reviewed actions · ${candidate.status || "unknown status"}` });
  }
  for (const candidate of sourceCandidates.browser_registry || []) {
    entries.push({ tone: "candidate", title: `Browser registry: ${candidate.title || candidate.id || "unnamed"}`, detail: `${candidate.actions?.length || 0} reviewed actions · ${candidate.status || "unknown status"}` });
  }
  for (const candidate of sourceCandidates.remote || []) {
    entries.push({ tone: "candidate", title: `Remote hint: ${candidate.title || candidate.id || candidate.kind || "unnamed"}`, detail: candidate.provenance || "reviewed remote metadata" });
  }
  for (const diagnostic of sourceCandidates.local?.rejected || []) {
    entries.push({ tone: "blocker", title: `Local source rejected: ${diagnostic.code || diagnostic.kind || "rejected"}`, detail: diagnostic.message || "No detail provided." });
  }
  for (const diagnostic of sourceCandidates.local?.ambiguous || []) {
    entries.push({ tone: "blocker", title: "Local source ambiguous", detail: diagnostic.message || `Possible kinds: ${(diagnostic.possible_kinds || []).join(", ") || "unknown"}` });
  }
  for (const diagnostic of sourceCandidates.local?.diagnostics || []) {
    entries.push({ tone: "blocker", title: `Local discovery: ${diagnostic.code || diagnostic.severity || "diagnostic"}`, detail: diagnostic.message || "No detail provided." });
  }
  for (const group of [sourceCandidates.browser?.rejected || [], sourceCandidates.browser?.ambiguous || [], sourceCandidates.browser?.truncated || []]) {
    for (const diagnostic of group) {
      entries.push({ tone: "blocker", title: `Browser discovery: ${diagnostic.code || "diagnostic"}`, detail: diagnostic.detail || "No detail provided." });
    }
  }
  for (const blocker of sourceCandidates.browser_registry_blockers || []) {
    entries.push({ tone: "blocker", title: `Registry blocker: ${blocker.code || "blocked"}`, detail: blocker.message || "No detail provided." });
  }
  if (sourceCandidates.remote_blocker) {
    entries.push({ tone: "blocker", title: `Remote blocker: ${sourceCandidates.remote_blocker.code || "blocked"}`, detail: sourceCandidates.remote_blocker.message || "No detail provided." });
  }
  appendEmptyOrItems("source-evidence-list", entries, "No discovery candidates or blockers are present.", (entry) => {
    const item = make("li", null, entry.tone === "blocker" ? "issue-warning" : "");
    item.append(make("strong", entry.title));
    item.append(document.createTextNode(` — ${entry.detail}`));
    return item;
  });
};

const renderSources = (snapshot) => {
  appendEmptyOrItems("sources-list", snapshot.selected_sources || [], "No sources selected.", (source) => {
    const target = source.target_path ? ` → ${source.target_path}` : "";
    const item = make("li");
    item.append(make("strong", `${source.kind || "source"}: ${source.id || "unnamed"}`));
    item.append(document.createTextNode(target));
		const facts = [];
		if (source.sha256) facts.push(`digest sha256:${source.sha256}`);
		if (source.provenance) facts.push(`provenance ${source.provenance}`);
		if (source.origins?.length) facts.push(`origins ${source.origins.join(", ")}`);
		if (source.actions?.length) facts.push(`actions ${source.actions.join(", ")}`);
		if (source.expires_at) facts.push(`expires ${source.expires_at}`);
		if (facts.length) item.append(make("small", ` — ${facts.join("; ")}`));
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

const renderTransactionFacts = (id, facts) => {
  const list = byID(id);
  clearNode(list);
  for (const [label, value] of facts) {
    const row = make("div");
    row.append(make("dt", label), make("dd", value || "—"));
    list.append(row);
  }
};

const transactionResource = () => state.renderedPayload?.browser_transaction || null;
const transactionAllowed = (operation) => (transactionResource()?.allowed_operations || []).includes(operation);

const renderBrowserTransaction = (payload, now = new Date()) => {
  const resource = payload.browser_transaction;
  const section = byID("browser-transaction-section");
  section.hidden = !resource;
  if (!resource) return;
  const transaction = resource.transaction;
  const review = resource.review;
  const revisionChanged = resource.revision !== state.transactionRevision;
  state.transactionRevision = resource.revision || "";
  if (revisionChanged) {
    for (const id of ["transaction-review-confirmed", "transaction-prepare-confirmed", "transaction-promote-confirmed", "transaction-recover-confirmed", "transaction-cancel-confirmed"]) byID(id).checked = false;
  }
  if (!transaction) {
    showText("browser-transaction-state", "Waiting for candidate");
    showText("browser-transaction-status", "No browser transaction is active. Candidate and review bodies remain outside this API.");
    byID("browser-transaction-content").hidden = true;
    return;
  }
  byID("browser-transaction-content").hidden = false;
  const expiresAt = new Date(transaction.provenance?.expires_at || "");
  const expired = Number.isFinite(expiresAt.getTime()) && now >= expiresAt;
  section.dataset.expired = expired ? "true" : "false";
  showText("browser-transaction-state", `${review?.composition || transaction.kind} · ${transaction.state}`);
  const failure = resource.last_failure;
  showText("browser-transaction-status", failure
    ? `${failure.operation} reported ${failure.code}${failure.promotion_state ? ` (${failure.promotion_state})` : ""}. Review the current state before continuing.`
    : expired && ["candidate", "reviewed"].includes(transaction.state)
      ? "Candidate freshness expired. Review and prepare are blocked; cancel or restart with a newly adopted candidate."
      : `Exact ${transaction.state} transaction. Runtime execution is unavailable.`);

  renderTransactionFacts("browser-transaction-provenance", [
    ["Composition", review?.composition], ["Transaction ID", transaction.id], ["Transaction digest", resource.transaction_sha256],
    ["Producer result", `${transaction.provenance?.result_version || "—"} · ${transaction.provenance?.result_sha256 || "—"}`],
    ["Observed", transaction.provenance?.observed_at], ["Expires", `${transaction.provenance?.expires_at || "—"}${expired ? " · expired" : " · engine rechecks before review and prepare"}`],
    ["Revision", resource.revision],
  ]);
  appendEmptyOrItems("browser-transaction-origins", review?.origins || transaction.provenance?.origins || [], "No approved origins are present.", (origin) => make("li", origin));
  const symbols = (review?.credential_bindings || transaction.credential_bindings || []).map((binding) => `${binding.slot} → ${binding.binding}`);
  if (review?.session) symbols.push(`browser session → ${review.session}`);
  appendEmptyOrItems("browser-transaction-symbols", symbols, "No credential values are present; this transaction uses no symbolic bindings.", (symbol) => make("li", symbol));
  const authority = [
    `Allowed now: ${(resource.allowed_operations || []).join(", ") || "observe only"}.`,
    "Candidate review is separate from prepare/qualification.",
    "Prepare/qualification is separate from atomic promotion.",
    "Promotion selects package bytes only; it grants no browser or workflow runtime authority.",
  ];
  if (resource.inspection?.execution_policy) authority.push(resource.inspection.execution_policy.side_effectful ? "Selected package declares downstream side effects requiring a separate trusted-runner approval." : "Selected package declares a read-only downstream posture.");
  appendEmptyOrItems("browser-transaction-authority", authority, "No transaction authority is available.", (item) => make("li", item));

  const virtualCandidates = payload.snapshot?.source_candidates?.virtual_browser?.candidates || [];
  const tbody = byID("browser-transaction-candidates");
  clearNode(tbody);
  for (const candidate of transaction.candidates || []) {
    const adopted = virtualCandidates.find((item) => item.transaction_id === transaction.id && item.kind === candidate.kind);
    const output = adopted?.target_path || "Output appears after exact private candidate adoption";
    const cleanup = candidate.kind === "registration" ? (adopted?.cleanup_disposition || "Cleanup remains separately review-bound") : "No registration cleanup";
    const row = make("tr");
    for (const [label, value] of [["Profile", candidate.kind], ["Schema", candidate.schema], ["Source digest", candidate.source_sha256], ["Review digest", candidate.review_sha256], ["Output and cleanup", `${output} · ${cleanup}`]]) {
      const cell = make("td", value || "—"); cell.dataset.label = label; row.append(cell);
    }
    tbody.append(row);
  }

  const registration = review?.registration_authoring;
  byID("browser-registration-disclosure").hidden = !registration;
  if (registration) {
    showText("browser-registration-label-disclosure", registration.accessibility_labels);
    appendEmptyOrItems("browser-registration-policy", [
      `Observation: ${registration.observation_status}; freshness is bounded by the exact observed/expires timestamps above.`,
      `Network: ${(registration.network_methods || []).join("/")} only; mutation requests allowed: ${registration.mutation_requests_allowed ? "yes" : "no"}.`,
      `Submit: ${registration.submit_supported ? "supported" : "not supported"}; account attempt: ${registration.account_attempt_supported ? "supported" : "not supported"}; session establishment: ${registration.session_establishment_supported ? "supported" : "not supported"}.`,
      `Symbol ${registration.approval_symbol} is descriptive and ${registration.approval_symbol_is_authority ? "does" : "does not"} grant execution authority.`,
    ], "No registration policy is present.", (item) => make("li", item));
  }

  const preparation = resource.preparation;
  renderTransactionFacts("browser-transaction-package", preparation ? [
    ["Preparation", preparation.preparation_sha256], ["Input", preparation.input_sha256], ["Package", preparation.package_sha256],
    ["Handoff", preparation.handoff_sha256], ["Quality", preparation.quality_sha256], ["Qualification", preparation.qualification_sha256],
  ] : [["Preparation", "Not prepared"]]);
  const promotion = resource.promotion;
  const report = resource.recovery?.report;
  const reconciliation = resource.recovery?.reconciliation;
  renderTransactionFacts("browser-transaction-recovery", [
    ["Generation", promotion?.generation_sha256], ["Selection", promotion?.selection_sha256], ["Baseline selection", promotion?.baseline_selection_sha256],
    ["Selected generation", promotion?.selected_generation_sha256], ["Prior generation", promotion?.prior_generation_sha256],
    ["Recovery resolution", report?.resolution], ["Recovery report", report?.recovery_sha256], ["Safe target", report?.target_generation_sha256 || failure?.target_generation_sha256],
    ["Accepted reconciliation", reconciliation?.observed_recovery_sha256],
  ]);
  updateControls();
};

const updateControls = () => {
	const locked = mutationLocked();
	const acquisitionIsLocked = acquisitionLocked();
  const snapshot = currentSnapshot();
  const preview = snapshot.preview;
  const conflicts = snapshot.write_conflicts || [];
  const reviewed = byID("review-confirmed").checked;
  const overwriteAllowed = conflicts.length === 0 || byID("allow-overwrite").checked;

  document.querySelectorAll(".question").forEach((wrapper) => updateQuestionState(wrapper, locked));
  byID("round-submit").disabled = locked || (snapshot.frontier || []).length === 0;
  document.querySelectorAll(".reopen-decision").forEach((button) => {
    button.disabled = locked || (snapshot.frontier || []).length > 0;
  });

  byID("review-confirmed").disabled = locked || !snapshot.approval_required;
  byID("allow-overwrite").disabled = locked || conflicts.length === 0;
  byID("overwrite-row").hidden = conflicts.length === 0 || Boolean(state.renderedPayload?.completed);

  const finalAvailable = Boolean(snapshot.approval_required && preview && !preview.incomplete && snapshot.ready);
  const incompleteAvailable = Boolean(snapshot.approval_required && preview?.incomplete);
  byID("approve-final").hidden = !finalAvailable;
  byID("approve-incomplete").hidden = !incompleteAvailable;
  byID("approve-final").disabled = locked || !reviewed || !overwriteAllowed || !finalAvailable;
	byID("approve-incomplete").disabled = locked || !reviewed || !overwriteAllowed || !incompleteAvailable;
	document.querySelectorAll('#journey-form input, #journey-form textarea, #journey-form button, #upload-form input, #upload-form button, #uploaded-sources-list button, #staged-sources-list button').forEach((control) => {
		control.disabled = acquisitionIsLocked;
	});
	byID("browser-preflight").disabled = acquisitionIsLocked;
	const capture = state.renderedPayload?.capture;
	const captureActive = capture && !["configuring", "staged", "canceled", "failed"].includes(capture.state);
	const captureFormLocked = Boolean(captureActive) || Boolean(capture?.containment_failed) || state.renderedPayload?.lifecycle !== "authoring" || state.dirty || state.pendingMutation || state.reconciliationRequired || state.unresolvedMutation || externallyModified();
	Array.from(byID("capture-form").elements).forEach((control) => { control.disabled = captureFormLocked; });
	const transaction = transactionResource();
	const transactionLocked = !transaction?.transaction || state.pendingMutation || state.dirty || state.reconciliationRequired || state.unresolvedMutation || externallyModified();
	const expiresAt = new Date(transaction?.transaction?.provenance?.expires_at || "");
	const transactionExpired = Number.isFinite(expiresAt.getTime()) && new Date() >= expiresAt;
	for (const [operation, rowID, confirmationID, buttonID] of [
		["review", "transaction-review-row", "transaction-review-confirmed", "transaction-review"],
		["prepare", "transaction-prepare-row", "transaction-prepare-confirmed", "transaction-prepare"],
		["promote", "transaction-promote-row", "transaction-promote-confirmed", "transaction-promote"],
		["recover", "transaction-recover-row", "transaction-recover-confirmed", "transaction-recover"],
		["cancel", "transaction-cancel-row", "transaction-cancel-confirmed", "transaction-cancel"],
	]) {
		const available = transactionAllowed(operation);
		byID(rowID).hidden = !available;
		byID(buttonID).hidden = !available;
		byID(confirmationID).disabled = transactionLocked || !available || (transactionExpired && ["review", "prepare"].includes(operation));
		byID(buttonID).disabled = transactionLocked || !available || !byID(confirmationID).checked || (transactionExpired && ["review", "prepare"].includes(operation));
	}
	byID("transaction-inspect-recovery").hidden = !transactionAllowed("inspect_recovery");
	byID("transaction-inspect-recovery").disabled = transactionLocked || !transactionAllowed("inspect_recovery");
	byID("transaction-inspect-selected").hidden = !transactionAllowed("inspect_selected");
	byID("transaction-inspect-selected").disabled = transactionLocked || !transactionAllowed("inspect_selected");

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
  if (state.renderedPayload) {
    byID("package-build").disabled = state.renderedPayload.lifecycle !== "authored" || !byID("package-confirmed").checked || state.pendingMutation;
    byID("package-confirmed").disabled = state.renderedPayload.lifecycle !== "authored" || state.pendingMutation;
  }
};

const renderPayload = (payload, refreshedAt = new Date()) => {
  state.renderedPayload = payload;
  state.latestPayload = payload;
  state.serverRevision = payload.revision;
  if (payload.__etag) state.etag = payload.__etag;
  state.dirty = false;
  state.unresolvedMutation = false;
  hideStateWarning();

  const snapshot = payload.snapshot || {};
  const preview = snapshot.preview;
  const drifted = externallyModified(payload);
  showConnection(
    drifted ? "Connected · restart required" : payload.completed ? "Connected · handoff-ready" : `Connected · ${payload.lifecycle || "authoring"}`,
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
  showStatus("completed", payload.completed ? "Handoff-ready" : (payload.lifecycle || "authoring"), payload.completed ? "positive" : "");
  showStatus("workspace", drifted ? "Externally modified" : "Unchanged", drifted ? "danger" : "positive");

  const reviewState = payload.lifecycle !== "authoring" ? "complete" : snapshot.approval_required ? "ready" : "working";
  byID("review-section").dataset.state = reviewState;
  showText("review-status", payload.lifecycle !== "authoring" ? `Authoring ${payload.lifecycle}` : snapshot.approval_required ? (preview?.incomplete ? "Draft review available" : "Ready for approval") : "Draft in progress");

  byID("workspace-warning").hidden = !drifted;
  byID("completion-banner").hidden = !payload.completed;
  byID("review-confirmed").checked = false;
  byID("allow-overwrite").checked = false;
  renderFrontier(snapshot, payload.revision);
  renderRevisableDecisions(snapshot);
  renderReadiness(snapshot);
  renderSources(snapshot);
  renderCandidateWorkflows(snapshot);
  renderDecisionEvidence(snapshot);
  renderSourceEvidence(snapshot);
  renderActions(snapshot);
  renderConflicts(snapshot);
  renderPreview(snapshot);
  renderWriteResult(payload);
  renderAcquisition(payload);
  renderWorkflowHelpers(snapshot);
  renderPackage(payload);
  renderBrowserTransaction(payload, refreshedAt);
  showText("snapshot", JSON.stringify(snapshot, null, 2));
  updateControls();
};

const receivePayload = (payload, refreshedAt = new Date(), focusStateWarning = false) => {
  state.serverRevision = payload.revision;
  state.latestPayload = payload;
  if (payload.__etag) state.etag = payload.__etag;

  if (payload.completed || externallyModified(payload)) {
    preserveCurrentAnswers();
    renderPayload(payload, refreshedAt);
    if (externallyModified(payload)) byID("workspace-warning").focus();
    return;
  }

	if (state.renderedPayload && state.dirty) {
		const revisionChanged = payload.revision !== state.renderedPayload.revision;
		showText("refreshed", refreshedAt.toLocaleTimeString());
		showConnection(revisionChanged ? "Connected · newer revision requires review" : "Connected · capture update requires review", "warning");
		showStateWarning(
			payload,
			revisionChanged
				? `The workspace advanced from ${state.renderedPayload.revision} to ${payload.revision}. Your unsent answers remain in the old form and will not be rebased or submitted automatically.`
				: "Capture or readiness state changed while you have unsent answers. The old form remains intact; review the update before discarding it.",
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
  if (!force && state.etag) headers["If-None-Match"] = state.etag;
  const response = await fetch("api/v4/snapshot", { credentials: "same-origin", headers });
  if (response.status === 304) return { unchanged: true, refreshedAt: new Date() };
  const payload = await decodePayload(response);
  if (!response.ok) {
    const failure = new Error(payload?.error?.message || `Snapshot request failed (${response.status})`);
    failure.payload = payload;
    throw failure;
  }
  payload.__etag = response.headers.get("ETag") || "";
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
      renderBrowserTransaction(state.renderedPayload, result.refreshedAt);
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
  const action = request.route === "author/approve" ? "Approval" : request.route === "reopen" ? "Answer reopening" : "Authoring mutation";
  announceMutation(`${action} failed. Review the error before continuing.`);

  if (failure.code === "engine_rejected" || status === 422) {
    const fieldFocusID = showQuestionError(error.question_id || "", failure.message);
    showError(failure.message, failure.requestID, null, !fieldFocusID);
    return fieldFocusID;
  }
  if (error.question_id && status > 0 && status < 500) {
    const fieldFocusID = showQuestionError(error.question_id, failure.message);
    showError(failure.message, failure.requestID, null, !fieldFocusID);
    return fieldFocusID;
  }
  if (failure.code === "workspace_changed" || failure.code === "session_frozen" || failure.code === "stale_revision" || status >= 500 || status === 0) {
    await reconcileMutationFailure(request, failure, failure.retryable);
    return "";
  }
  showError(failure.message, failure.requestID);
  return "";
};

async function sendMutation(request) {
	if (mutationLocked() || (state.dirty && acquisitionRoutes.has(request.route))) return;
  let postMutationFocusID = "";
  clearTimeout(state.timer);
  state.pollGeneration += 1;
  clearError();
  state.pendingMutation = true;
  announceMutation(request.route === "author/approve" ? "Approval is being committed." : request.route === "reopen" ? "The settled answer is being reopened." : "Authoring mutation is being submitted.");
  updateControls();
  try {
    const response = await fetch(`api/v4/${request.route}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(request.body),
    });
    const payload = await decodePayload(response);
    if (!response.ok) {
      postMutationFocusID = await handleMutationFailure(request, response.status, payload);
      return;
    }
    clearError();
    payload.__etag = response.headers.get("ETag") || "";
    renderPayload(payload, new Date());
    if (request.route === "author/approve") {
      announceMutation("Approval committed. The authored files are ready for a separate package build.");
      postMutationFocusID = "package-heading";
    } else if (request.route === "reopen") {
      announceMutation("Answer reopened. Submit the complete replacement frontier to continue.");
      postMutationFocusID = answerInputs()[0]?.id || "frontier-heading";
    } else {
      const inputs = answerInputs();
      if (inputs.length > 0) {
        announceMutation("Round submitted. Continue with the next authoring question.");
        postMutationFocusID = inputs[0].id;
      } else if (payload.snapshot?.approval_required) {
        announceMutation("Round submitted. The proposal is ready for review.");
        postMutationFocusID = "review-heading";
      } else {
        announceMutation("Round submitted. Review the updated authoring state.");
        postMutationFocusID = "frontier-heading";
      }
    }
  } catch (error) {
    postMutationFocusID = await handleMutationFailure(request, 0, { error: { code: "network_error", message: error.message, retryable: true } });
  } finally {
    state.pendingMutation = false;
    updateControls();
    if (postMutationFocusID) byID(postMutationFocusID)?.focus();
    schedule(normalInterval);
  }
}

async function sendLifecycleJSON(route, body, message = "Updating local state…") {
	if (state.pendingMutation || !state.renderedPayload || (state.dirty && acquisitionRoutes.has(route))) return;
  clearTimeout(state.timer);
  clearError();
  state.pendingMutation = true;
  announceMutation(message);
  updateControls();
  try {
    const response = await fetch(`api/v4/${route}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const payload = await decodePayload(response);
    if (!response.ok) {
      showError(payload?.error?.message || `Request failed (${response.status}).`, payload?.error?.request_id || "");
      return;
    }
    payload.__etag = response.headers.get("ETag") || "";
    renderPayload(payload, new Date());
    announceMutation("Local state updated.");
  } catch (error) {
    showError(error.message);
  } finally {
    state.pendingMutation = false;
    updateControls();
    schedule(normalInterval);
  }
}

async function sendBrowserTransaction(route, body, message) {
	if (state.pendingMutation || state.dirty || externallyModified() || state.reconciliationRequired || state.unresolvedMutation) return;
	clearTimeout(state.timer);
	clearError();
	state.pendingMutation = true;
	announceMutation(message);
	updateControls();
	let failure = null;
	try {
		const response = await fetch(`api/v4/browser-transactions/${route}`, {
			method: "POST", credentials: "same-origin",
			headers: { Accept: "application/json", "Content-Type": "application/json" },
			body: JSON.stringify(body),
		});
		const payload = await decodePayload(response);
		if (!response.ok) failure = { message: payload?.error?.message || `Browser transaction request failed (${response.status}).`, requestID: payload?.error?.request_id || "" };
		const latest = await fetchSnapshot(true);
		if (latest.payload) renderPayload(latest.payload, latest.refreshedAt);
		if (failure) {
			showError(failure.message, failure.requestID);
			announceMutation("Browser transaction state requires review.");
		} else {
			announceMutation("Browser transaction state updated. Review the exact next checkpoint.");
		}
	} catch (error) {
		showError(error.message);
		announceMutation("Browser transaction update could not be confirmed.");
	} finally {
		state.pendingMutation = false;
		updateControls();
		byID("browser-transaction-heading")?.focus();
		schedule(normalInterval);
	}
}

const renderAcquisition = (payload) => {
  const snapshot = payload.snapshot || {};
  const journey = snapshot.journey || {};
  showText("lifecycle-state", payload.lifecycle || "authoring");
	document.querySelectorAll('input[name="journey"]').forEach((input) => {
		input.checked = input.value === journey.starter;
		input.disabled = acquisitionLocked();
	});
	if (document.activeElement !== byID("journey-goal")) byID("journey-goal").value = journey.goal || "";
	byID("journey-goal").disabled = acquisitionLocked();
	byID("journey-submit").disabled = acquisitionLocked();
	Array.from(byID("upload-form").elements).forEach((control) => { control.disabled = acquisitionLocked(); });

  appendEmptyOrItems("uploaded-sources-list", snapshot.uploaded_sources || [], "No validated uploads are waiting for staging.", (source) => {
    const item = make("li");
    item.append(make("strong", `${source.original_name} · ${source.kind}`));
    item.append(document.createTextNode(` — ${source.operation_count || 0} operations; ${source.bytes || 0} bytes; target `));
    item.append(make("code", source.canonical_target));
    item.append(document.createTextNode(`; sha256:${source.sha256}`));
    const button = make("button", "Stage this source");
    button.type = "button";
		button.disabled = acquisitionLocked();
    button.addEventListener("click", () => sendMutation({ route: "source/stage", body: { revision: state.renderedPayload.revision, id: source.id } }));
    item.append(button);
    return item;
  });
  appendEmptyOrItems("staged-sources-list", snapshot.staged_sources || [], "No UI-owned source has been staged.", (source) => {
    const item = make("li");
    item.append(make("strong", `${source.kind}: ${source.target_path}`));
    item.append(document.createTextNode(` — sha256:${source.sha256}`));
    const button = make("button", "Remove staged source");
    button.type = "button";
		button.disabled = acquisitionLocked();
    button.addEventListener("click", () => sendMutation({ route: "source/remove", body: { revision: state.renderedPayload.revision, id: source.id } }));
    item.append(button);
    return item;
  });

  const doctor = payload.browser_doctor;
  const doctorPanel = byID("doctor-panel");
  clearNode(doctorPanel);
  if (doctor) {
    for (const [label, value] of [["Driver", doctor.driver_ready ? "ready" : "unavailable"], ["Chromium", doctor.browser_ready ? "ready" : "unavailable"], ["Playwright", doctor.playwright_version || "—"]]) {
      const row = make("div"); row.append(make("dt", label), make("dd", value)); doctorPanel.append(row);
    }
  }
	byID("browser-preflight").disabled = acquisitionLocked();
  renderCapture(payload);
	renderRegistrationAuthoring(payload);
};

const renderCapture = (payload) => {
  const capture = payload.capture;
  const panel = byID("capture-panel");
  clearNode(panel);
	const active = capture && !["configuring", "staged", "canceled", "failed"].includes(capture.state);
	Array.from(byID("capture-form").elements).forEach((control) => {
		control.disabled = Boolean(active) || Boolean(capture?.containment_failed) || payload.lifecycle !== "authoring" || state.dirty || state.pendingMutation || state.reconciliationRequired || state.unresolvedMutation || externallyModified(payload);
	});
	byID("capture-cancel").hidden = !active || capture?.state === "canceling";
  if (!capture) return;
  panel.append(make("h4", capture.state.replaceAll("_", " ")));
  if (capture.message) panel.append(make("p", capture.message));
  if (capture.observation) {
    panel.append(make("p", `${capture.observation.origin}${capture.observation.path} · context ${capture.observation.context}`));
    const actions = make("div", null, "form-actions");
    for (const candidate of capture.observation.candidates || []) {
      const credentialRole = ["textbox", "combobox"].includes(candidate.role);
      const button = make("button", credentialRole ? `Use ${candidate.role} in Chromium: ${candidate.label || candidate.id}` : `${candidate.role}: ${candidate.label || candidate.id}`);
      button.type = "button";
      button.addEventListener("click", () => sendCaptureResponse({
        kind: credentialRole ? "focus_human_input" : "click", candidate_id: candidate.id, post_budget: credentialRole ? 0 : 1,
      }));
      actions.append(button);
    }
    const authenticated = make("button", "Authentication is complete");
    authenticated.type = "button";
    authenticated.addEventListener("click", () => sendCaptureResponse({ kind: "authenticated" }));
    actions.append(authenticated);
    panel.append(actions);
  }
  if (capture.approval) {
    panel.append(make("p", `Exact approval: ${capture.approval.kind}; origin ${capture.approval.origin || "current"}; action ${capture.approval.action || "—"}; POST budget ${capture.approval.postBudget ?? capture.approval.post_budget ?? 0}.`));
    for (const choice of ["approve", "deny"]) {
      const button = make("button", choice === "approve" ? "Approve exact action" : "Deny");
      button.type = "button";
      button.addEventListener("click", () => sendCaptureResponse({ kind: choice, approval_id: capture.approval.id }));
      panel.append(button);
    }
  }
  if (capture.checkpoint?.kind === "credential") {
    panel.append(make("p", "Type the credential only in Chromium. Do not paste it into this page."));
    const button = make("button", "Continue after Chromium input"); button.type = "button";
    button.addEventListener("click", () => sendCaptureResponse({ kind: "continue", candidate_id: capture.checkpoint.candidateId || capture.checkpoint.candidate_id })); panel.append(button);
  }
  if (capture.checkpoint?.kind === "mfa") {
    panel.append(make("p", "Complete the selected challenge in Chromium or on the paired device. No OTP value belongs here."));
    for (const kind of capture.checkpoint.challengeKinds || capture.checkpoint.challenge_kinds || []) {
      const button = make("button", `Continue with ${kind}`); button.type = "button";
      button.addEventListener("click", () => sendCaptureResponse({ kind: "continue", candidate_id: capture.checkpoint.candidateId || capture.checkpoint.candidate_id, challenge_kind: kind })); panel.append(button);
    }
  }
  if (capture.checkpoint?.kind === "completion") {
    panel.append(make("p", "The typed completion predicate matched. Optionally select up to 16 value-free output declarations from this final reduced observation."));
    const outputRows = make("fieldset", null, "capture-output-list");
    outputRows.append(make("legend", "Reviewed outputs"));
    for (const [index, candidate] of (capture.observation?.candidates || []).entries()) {
      const row = make("div", null, "capture-output-row");
      const choose = document.createElement("input");
      choose.type = "checkbox";
      choose.dataset.captureOutput = candidate.id;
      const chooseLabel = make("label");
      chooseLabel.append(choose, document.createTextNode(` ${candidate.role}: ${candidate.label || candidate.id}`));
      const key = document.createElement("input");
      key.type = "text";
      key.maxLength = 64;
      key.placeholder = "output_key";
      key.dataset.outputField = "key";
      key.disabled = true;
      const type = document.createElement("select");
      type.dataset.outputField = "type";
      for (const value of ["string", "integer", "number", "boolean", "presence"]) {
        const option = make("option", value); option.value = value; type.append(option);
      }
      type.disabled = true;
      const locator = document.createElement("select");
      locator.dataset.outputField = "locatorMode";
      for (const value of ["exact_name", "unique_role"]) {
        const option = make("option", value); option.value = value; locator.append(option);
      }
      locator.disabled = true;
      choose.addEventListener("change", () => {
        key.disabled = !choose.checked; type.disabled = !choose.checked; locator.disabled = !choose.checked;
        if (choose.checked && !key.value) key.value = `output_${index + 1}`;
      });
      row.append(chooseLabel, key, type, locator);
      outputRows.append(row);
    }
    panel.append(outputRows);
    const button = make("button", "Confirm typed completion"); button.type = "button";
    button.addEventListener("click", () => {
      const outputs = Array.from(outputRows.querySelectorAll("[data-capture-output]:checked")).map((choice) => {
        const row = choice.closest(".capture-output-row");
        return {
          candidateId: choice.dataset.captureOutput,
          key: row.querySelector('[data-output-field="key"]').value.trim(),
          type: row.querySelector('[data-output-field="type"]').value,
          locatorMode: row.querySelector('[data-output-field="locatorMode"]').value,
        };
      });
      if (outputs.length > 16 || outputs.some((output) => !/^[a-z][a-z0-9_]{0,63}$/.test(output.key))) {
        showError("Choose at most 16 outputs and give each a lowercase symbolic key.");
        return;
      }
      sendCaptureResponse({ kind: "confirm", confirmed: true, outputs });
    }); panel.append(button);
  }
  if (capture.result_ready) {
    const button = make("button", "Stage reviewed profiles", "primary"); button.type = "button";
    button.addEventListener("click", () => sendLifecycleJSON("capture/stage", { revision: payload.revision, capture_revision: payload.capture_revision }, "Revalidating capture and staging profiles…")); panel.append(button);
  }
};

const registrationCandidateLabel = (candidate) => `${candidate.role}: ${candidate.label || "unlabeled"}`;

const registrationCandidates = () => state.renderedPayload?.registration_authoring?.observation?.candidates || [];

const populateRegistrationSelect = (select, values, labeler = (value) => value) => {
	const selected = select.value;
	clearNode(select);
	const placeholder = make("option", "Choose…"); placeholder.value = ""; select.append(placeholder);
	for (const value of values) {
		const option = make("option", labeler(value));
		option.value = typeof value === "string" ? value : value.id;
		select.append(option);
	}
	if (Array.from(select.options).some((option) => option.value === selected)) select.value = selected;
};

const registrationSlotSymbols = () => Array.from(byID("registration-slot-list").children)
	.map((row) => row.querySelector('[data-registration-slot="slot"]').value.trim())
	.filter(Boolean);

const refreshRegistrationSlotChoices = () => {
	const symbols = registrationSlotSymbols();
	for (const select of document.querySelectorAll('[data-registration-step="slot"]')) populateRegistrationSelect(select, symbols);
};

const addRegistrationSlot = (slot = "", kind = "identifier", binding = "") => {
	const index = byID("registration-slot-list").children.length + 1;
	const row = make("div", null, "wizard-row registration-slot-row");
	const slotID = `registration-slot-${index}`;
	const slotLabel = make("label", "Slot symbol"); slotLabel.htmlFor = slotID;
	const slotInput = make("input"); slotInput.id = slotID; slotInput.dataset.registrationSlot = "slot"; slotInput.value = slot; slotInput.pattern = "[a-z][a-z0-9_]{0,127}";
	const kindID = `${slotID}-kind`; const kindLabel = make("label", "Credential class"); kindLabel.htmlFor = kindID;
	const kindSelect = make("select"); kindSelect.id = kindID; kindSelect.dataset.registrationSlot = "kind";
	for (const value of ["identifier", "password"]) { const option = make("option", value); option.value = value; kindSelect.append(option); }
	kindSelect.value = kind;
	const bindingID = `${slotID}-binding`; const bindingLabel = make("label", "Environment symbol"); bindingLabel.htmlFor = bindingID;
	const bindingInput = make("input"); bindingInput.id = bindingID; bindingInput.dataset.registrationSlot = "binding"; bindingInput.value = binding; bindingInput.pattern = "[a-z][a-z0-9_]{0,127}";
	const remove = make("button", "Remove slot"); remove.type = "button"; remove.addEventListener("click", () => { row.remove(); refreshRegistrationSlotChoices(); });
	slotInput.addEventListener("input", refreshRegistrationSlotChoices);
	row.append(slotLabel, slotInput, kindLabel, kindSelect, bindingLabel, bindingInput, remove);
	byID("registration-slot-list").append(row);
	refreshRegistrationSlotChoices();
};

const updateRegistrationStepRow = (row) => {
	const type = row.querySelector('[data-registration-step="type"]').value;
	row.querySelector('[data-registration-step-field="navigate"]').hidden = type !== "navigate";
	row.querySelector('[data-registration-step-field="candidate"]').hidden = !["type_credential", "click", "submit", "human_checkpoint", "wait_for"].includes(type);
	row.querySelector('[data-registration-step-field="slot"]').hidden = type !== "type_credential";
	row.querySelector('[data-registration-step-field="checkpoint"]').hidden = type !== "human_checkpoint";
};

const addRegistrationStep = (type = "navigate", slot = "") => {
	const index = byID("registration-step-list").children.length + 1;
	const row = make("fieldset", null, "wizard-row registration-step-row");
	row.append(make("legend", `Step ${index}`));
	const typeID = `registration-step-${index}-type`; const typeLabel = make("label", "Macro type"); typeLabel.htmlFor = typeID;
	const typeSelect = make("select"); typeSelect.id = typeID; typeSelect.dataset.registrationStep = "type";
	for (const value of ["navigate", "type_credential", "click", "submit", "human_checkpoint", "wait_for"]) { const option = make("option", value.replaceAll("_", " ")); option.value = value; typeSelect.append(option); }
	typeSelect.value = type;

	const navigateWrap = make("div"); navigateWrap.dataset.registrationStepField = "navigate";
	const navigateID = `registration-step-${index}-navigate`; const navigateLabel = make("label", "Exact GET navigation URL"); navigateLabel.htmlFor = navigateID;
	const navigate = make("input"); navigate.id = navigateID; navigate.type = "url"; navigate.dataset.registrationStep = "navigate";
	if (type === "navigate") navigate.value = byID("registration-url").value.trim(); navigateWrap.append(navigateLabel, navigate);

	const candidateWrap = make("div"); candidateWrap.dataset.registrationStepField = "candidate";
	const candidateID = `registration-step-${index}-candidate`; const candidateLabel = make("label", "Observed accessibility locator"); candidateLabel.htmlFor = candidateID;
	const candidate = make("select"); candidate.id = candidateID; candidate.dataset.registrationStep = "candidate";
	populateRegistrationSelect(candidate, registrationCandidates(), registrationCandidateLabel); candidateWrap.append(candidateLabel, candidate);

	const slotWrap = make("div"); slotWrap.dataset.registrationStepField = "slot";
	const stepSlotID = `registration-step-${index}-slot`; const stepSlotLabel = make("label", "Credential slot symbol"); stepSlotLabel.htmlFor = stepSlotID;
	const stepSlot = make("select"); stepSlot.id = stepSlotID; stepSlot.dataset.registrationStep = "slot";
	populateRegistrationSelect(stepSlot, registrationSlotSymbols());
	if (slot && Array.from(stepSlot.options).some((option) => option.value === slot)) stepSlot.value = slot;
	slotWrap.append(stepSlotLabel, stepSlot);

	const checkpointWrap = make("div"); checkpointWrap.dataset.registrationStepField = "checkpoint";
	const checkpointID = `registration-step-${index}-checkpoint`; const checkpointLabel = make("label", "Human checkpoint kind"); checkpointLabel.htmlFor = checkpointID;
	const checkpoint = make("select"); checkpoint.id = checkpointID; checkpoint.dataset.registrationStep = "checkpoint";
	for (const value of ["captcha", "email_verification", "mfa", "consent", "other_control"]) { const option = make("option", value.replaceAll("_", " ")); option.value = value; checkpoint.append(option); }
	checkpointWrap.append(checkpointLabel, checkpoint);
	const remove = make("button", "Remove step"); remove.type = "button"; remove.addEventListener("click", () => row.remove());
	typeSelect.addEventListener("change", () => updateRegistrationStepRow(row));
	row.append(typeLabel, typeSelect, navigateWrap, candidateWrap, slotWrap, checkpointWrap, remove);
	byID("registration-step-list").append(row);
	updateRegistrationStepRow(row);
};

const initializeRegistrationRows = () => {
	if (state.registrationRowsInitialized) return;
	state.registrationRowsInitialized = true;
	addRegistrationSlot("identifier", "identifier", "dedicated_test_identifier");
	addRegistrationSlot("password", "password", "dedicated_test_password");
	addRegistrationSlot("contact_name", "identifier", "dedicated_test_contact_name");
	addRegistrationStep("navigate");
	addRegistrationStep("type_credential", "identifier");
	addRegistrationStep("type_credential", "password");
	addRegistrationStep("type_credential", "password");
	addRegistrationStep("type_credential", "contact_name");
	addRegistrationStep("click");
	addRegistrationStep("submit");
};

const refreshRegistrationCandidateChoices = () => {
	for (const select of document.querySelectorAll('[data-registration-step="candidate"]')) populateRegistrationSelect(select, registrationCandidates(), registrationCandidateLabel);
	const origins = byID("registration-origins").value.split(/\r?\n/).map((value) => value.trim()).filter(Boolean);
	populateRegistrationSelect(byID("registration-success-origin"), origins);
};

const renderRegistrationAuthoring = (payload) => {
	initializeRegistrationRows();
	const authoring = payload.registration_authoring;
	showText("registration-authoring-state", authoring?.state?.replaceAll("_", " ") || "Not started");
	showText("registration-authoring-status", authoring?.message || (payload.browser_transaction
		? "This wizard observes accessibility metadata with GET or HEAD only. It cannot type, click, submit, create an account, sign in, or execute the drafted workflow."
		: "Configure package scope, restrictive scratch, and a generation store when launching iCoT before registration authoring."));
	const pendingCandidate = ["review_ready", "transaction_review", "adopted"].includes(authoring?.state);
	const startLocked = state.pendingMutation || !payload.browser_transaction || payload.lifecycle !== "authoring" || externallyModified(payload) || Boolean(authoring?.containment_failed) || registrationActive(authoring) || pendingCandidate;
	Array.from(byID("registration-start-form").elements).forEach((control) => { control.disabled = startLocked; });
	byID("registration-cancel").hidden = !registrationActive(authoring) || ["canceling", "transaction_review"].includes(authoring?.state);

	const panel = byID("registration-observation-panel"); clearNode(panel);
	if (authoring?.bounds) panel.append(make("p", `Fixed bounds: ${authoring.bounds.maxObservations ?? authoring.bounds.max_observations ?? 0} observations; ${authoring.bounds.maxCandidates ?? authoring.bounds.max_candidates ?? 0} candidates.`));
	if (authoring?.state === "observing") {
		const observe = make("button", "Observe current page", "primary"); observe.type = "button";
		observe.addEventListener("click", () => sendRegistrationCommand("observe")); panel.append(observe);
	}
	if (authoring?.observation) {
		panel.append(make("h4", "Reduced accessibility observation"));
		panel.append(make("p", `${authoring.observation.origin}${authoring.observation.path}`));
		panel.append(make("p", "Accessibility names may contain personal or account information. Review and retain only the minimum labels required for portable locators.", "section-help"));
		const list = make("ul", null, "detail-list");
		for (const candidate of authoring.observation.candidates || []) list.append(make("li", `${registrationCandidateLabel(candidate)} · unique matches ${candidate.matches}`));
		panel.append(list);
		const navigation = make("fieldset", null, "navigation-row"); navigation.append(make("legend", "Optional GET or HEAD navigation"));
		const method = make("select"); method.setAttribute("aria-label", "Navigation method"); for (const value of ["GET", "HEAD"]) { const option = make("option", value); option.value = value; method.append(option); }
		const target = make("input"); target.type = "url"; target.placeholder = "https://…"; target.setAttribute("aria-label", "Exact navigation URL");
		const navigate = make("button", "Navigate without submitting"); navigate.type = "button"; navigate.addEventListener("click", () => {
			if (!target.value.trim()) { showError("Enter an exact GET or HEAD URL."); return; }
			sendRegistrationCommand("navigate", { method: method.value, url: target.value.trim() });
		});
		navigation.append(method, target, navigate); panel.append(navigation);
	}
	refreshRegistrationCandidateChoices();
	byID("registration-draft-form").hidden = authoring?.state !== "observation";

	const review = byID("registration-canonical-review");
	review.hidden = !authoring?.draft;
	if (authoring?.draft) {
		showText("registration-accessibility-disclosure", authoring.draft.accessibility_labels);
		appendEmptyOrItems("registration-retained-queries", authoring.draft.retained_queries || [], "No literal query is retained.", (entry) => make("li", `${entry.navigation} — ${(entry.parameters || []).map((parameter) => `${parameter.key}=${parameter.value}`).join("; ")}`));
		appendEmptyOrItems("registration-draft-authority", [
			...(authoring.draft.credential_bindings || []).map((binding) => `${binding.slot} → ${binding.binding}`),
			`success proof → ${authoring.draft.success_proof?.review_kind || "unavailable"}; observed during authoring → ${authoring.draft.success_proof?.observed_during_authoring ? "yes" : "no"}; runtime proof required → ${authoring.draft.success_proof?.runtime_proof_required ? "yes" : "no"}`,
			`cleanup → ${authoring.draft.cleanup_disposition}`,
			`approval → ${authoring.draft.call_controls?.approval}`,
			`duplicate prevention → ${authoring.draft.call_controls?.duplicate_prevention}; on duplicate → ${authoring.draft.call_controls?.on_duplicate}`,
			`ambiguous outcome → ${authoring.draft.call_controls?.ambiguous_outcome}`,
		], "Draft authority is unavailable.", (value) => make("li", value));
		showText("registration-canonical-profile", JSON.stringify(authoring.draft.canonical, null, 2));
	}
	byID("registration-review").hidden = authoring?.state !== "draft_review";
	byID("registration-finish").hidden = authoring?.state !== "reviewed";
	byID("registration-review").disabled = authoring?.state !== "draft_review" || !byID("registration-draft-confirmed").checked || state.pendingMutation;
	byID("registration-finish").disabled = authoring?.state !== "reviewed" || state.pendingMutation;
};

const sendRegistrationCommand = (type, extra = {}) => sendLifecycleJSON("registration-authoring/command", {
	revision: state.renderedPayload.revision,
	registration_revision: state.renderedPayload.registration_authoring_revision,
	type,
	...extra,
}, "Sending a typed no-submit registration-authoring command…");

const sendCaptureResponse = (response) => sendLifecycleJSON("capture/respond", {
  capture_revision: state.renderedPayload.capture_revision,
  response,
}, "Sending a typed browser decision…");

const renderWorkflowHelpers = (snapshot) => {
  const graph = [];
  if (snapshot.boundary?.outcome) graph.push(`Goal → ${snapshot.boundary.outcome}`);
  for (const source of snapshot.selected_sources || []) graph.push(`Source → ${source.kind}:${source.id}`);
  if (snapshot.approval_required) graph.push("Human approval → authored files");
  graph.push("Authored files → separate package build → trusted handoff");
  appendEmptyOrItems("workflow-graph", graph, "The graph will appear as the workflow boundary settles.", (item) => make("li", item));
  const symbols = [];
  for (const source of snapshot.selected_sources || []) {
    for (const [flow, slots] of Object.entries(source.flow_credential_slots || {})) symbols.push(`${source.id}.${flow} → ${slots.join(", ") || "no credential slots"}`);
    if (source.login_state_required) symbols.push(`${source.id} → named browser session required`);
  }
  appendEmptyOrItems("symbol-map", symbols, "No symbolic credential or browser-session binding is selected.", (item) => make("li", item));
};

const renderApprovalArgv = (argv) => {
	const container = byID("approval-command");
	clearNode(container);
	if (!Array.isArray(argv) || argv.length === 0) {
		container.textContent = "Approval-template argv appears only after a passing build.";
		return;
	}
	container.append(make("span", "Exact arguments, in order:"));
	const list = make("ol", null, "argv-list");
	for (const argument of argv) {
		const item = make("li");
		item.append(make("code", String(argument)));
		list.append(item);
	}
	container.append(list);
};

const renderPackage = (payload) => {
  showText("package-status", payload.package?.status || (payload.lifecycle === "authored" ? "Ready to build" : payload.lifecycle));
  byID("package-build").disabled = payload.lifecycle !== "authored" || !byID("package-confirmed").checked || state.pendingMutation;
  byID("package-confirmed").disabled = payload.lifecycle !== "authored" || state.pendingMutation;
  byID("authoring-resume").hidden = payload.lifecycle !== "package_failed";
  appendEmptyOrItems("quality-list", payload.package?.quality?.checks || [], "No package quality result yet.", (check) => make("li", `${check.code}: ${check.status} — ${check.message}`));
  appendEmptyOrItems("remediation-list", payload.package?.remediation || [], "No package remediation is pending.", (item) => make("li", item));
  const digests = byID("handoff-digests"); clearNode(digests);
  const inspection = payload.package?.inspection;
  if (inspection) {
    const values = [
      ["Package digest", inspection.package_sha256], ["Handoff digest", inspection.handoff_sha256],
      ["Side effects", inspection.execution_policy?.side_effectful ? "side-effectful; downstream approval required" : "read-only posture"],
      ["Credential symbols", (inspection.credential_bindings?.declared || []).join(", ") || "none"],
      ["Runtime approvals", (inspection.approval_states || []).map((item) => item.name).join(" → ")],
    ];
    for (const [label, value] of values) { const row = make("div"); row.append(make("dt", label), make("dd", value || "—")); digests.append(row); }
  }
	renderApprovalArgv(payload.package?.approval_template_argv);
  appendEmptyOrItems("package-artifacts", payload.package?.artifacts || [], "No handoff artifact is available.", (artifact) => {
    const item = make("li", `${artifact.name} · ${artifact.path} · ${artifact.sha256}`);
    const button = make("button", "Inspect"); button.type = "button";
    button.addEventListener("click", () => window.open(`api/v4/artifact?name=${encodeURIComponent(artifact.name)}`, "_blank", "noopener")); item.append(button); return item;
  });
};

const validateRound = () => {
  let firstInvalid = null;
  for (const input of answerInputs()) {
    const wrapper = questionWrapper(input);
    const toggle = wrapper?.querySelector(".deferral-toggle");
    const error = byID(`${input.id}-error`);
    clearQuestionError(wrapper);
    if (toggle?.checked) {
      const fields = deferralFields(wrapper);
      const missing = fields.find((field) => field.value.trim() === "");
      const invalidSeparator = fields.find((field) => field.value.includes("|"));
      if (missing || invalidSeparator) {
        error.textContent = missing
          ? "A deferral requires an owner, impact, unblock condition, and suggested next action."
          : "Deferral fields may not contain the | character.";
        error.hidden = false;
        const invalid = missing || invalidSeparator;
        invalid.setAttribute("aria-invalid", "true");
        if (!firstInvalid) firstInvalid = invalid;
      }
    } else if (input.value.trim() === "") {
      error.textContent = "This frontier question requires an answer before the round can be submitted.";
      error.hidden = false;
      input.setAttribute("aria-invalid", "true");
      if (!firstInvalid) firstInvalid = input;
    }
  }
  if (firstInvalid) firstInvalid.focus();
  return firstInvalid === null;
};

byID("round-form").addEventListener("submit", (event) => {
  event.preventDefault();
  if (mutationLocked() || !validateRound()) return;
  const answers = collectAnswers().map(({ question_id, value, deferral }) => (
    deferral
      ? { question_id, deferral: Object.fromEntries(Object.entries(deferral).map(([key, item]) => [key, item.trim()])) }
      : { question_id, value: value.trim() }
  ));
  sendMutation({ route: "round", body: { revision: state.renderedPayload.revision, answers } });
});

byID("journey-form").addEventListener("submit", (event) => {
	event.preventDefault();
	if (acquisitionLocked()) return;
  const starter = document.querySelector('input[name="journey"]:checked')?.value || "";
  const goal = byID("journey-goal").value.trim();
  if (!starter || !goal) { showError("Choose a journey starter and enter a goal."); return; }
  sendMutation({ route: "journey", body: { revision: state.renderedPayload.revision, starter, goal } });
});

byID("upload-form").addEventListener("submit", async (event) => {
	event.preventDefault();
	if (acquisitionLocked()) return;
  const file = byID("source-file").files?.[0];
  if (!file) { showError("Choose one API-family document."); return; }
  const body = new FormData();
  body.append("revision", state.renderedPayload.revision);
  body.append("source", file, file.name);
  state.pendingMutation = true; updateControls(); announceMutation("Validating the private API source upload…");
  try {
    const response = await fetch("api/v4/source/upload", { method: "POST", credentials: "same-origin", headers: { Accept: "application/json" }, body });
    const payload = await decodePayload(response);
    if (!response.ok) { showError(payload?.error?.message || `Upload failed (${response.status}).`, payload?.error?.request_id || ""); return; }
    payload.__etag = response.headers.get("ETag") || ""; byID("source-file").value = ""; renderPayload(payload, new Date());
  } catch (error) { showError(error.message); }
  finally { state.pendingMutation = false; updateControls(); schedule(normalInterval); }
});

byID("browser-preflight").addEventListener("click", () => {
	if (acquisitionLocked()) return;
  sendMutation({ route: "browser/preflight", body: { revision: state.renderedPayload.revision, capture_revision: state.renderedPayload.capture_revision } });
});

byID("capture-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const origins = byID("capture-origins").value.split(/\r?\n/).map((value) => value.trim()).filter(Boolean);
  let dashboard;
  try { dashboard = new URL(byID("capture-dashboard").value); } catch (_) { showError("Enter a valid protected dashboard URL."); return; }
  sendLifecycleJSON("capture/start", {
    revision: state.renderedPayload.revision,
    capture_revision: state.renderedPayload.capture_revision,
    profile_id: byID("capture-profile").value.trim(),
    url: byID("capture-url").value.trim(),
    dashboard_url: dashboard.toString(),
    goal: byID("capture-goal").value.trim(),
    origins,
    goal_origin: dashboard.origin,
    goal_path: dashboard.pathname || "/",
    goal_context: "main",
    goal_role: byID("capture-goal-role").value.trim(),
    goal_label: byID("capture-goal-label").value.trim(),
  }, "Launching the isolated Browsertools worker…");
});

byID("capture-cancel").addEventListener("click", () => sendLifecycleJSON("capture/cancel", {
  capture_revision: state.renderedPayload.capture_revision,
}, "Canceling the browser capture and its descendants…"));

byID("registration-start-form").addEventListener("submit", (event) => {
	event.preventDefault();
	const origins = byID("registration-origins").value.split(/\r?\n/).map((value) => value.trim()).filter(Boolean);
	let target;
	try { target = new URL(byID("registration-url").value.trim()); } catch (_) { showError("Enter a valid absolute registration URL."); return; }
	if (origins.length === 0) { showError("Enter at least one exact approved origin."); return; }
	sendLifecycleJSON("registration-authoring/start", {
		revision: state.renderedPayload.revision,
		registration_revision: state.renderedPayload.registration_authoring_revision,
		profile_id: byID("registration-profile-id").value.trim(),
		url: target.toString(), origins,
	}, "Launching the isolated no-submit registration observer…");
});

byID("registration-add-slot").addEventListener("click", () => addRegistrationSlot());
byID("registration-add-step").addEventListener("click", () => addRegistrationStep("navigate"));

const collectRegistrationDraft = () => {
	const stabilityText = byID("registration-stability").value.trim();
	const slots = Array.from(byID("registration-slot-list").children).map((row) => ({
		slot: row.querySelector('[data-registration-slot="slot"]').value.trim(),
		kind: row.querySelector('[data-registration-slot="kind"]').value,
		binding: row.querySelector('[data-registration-slot="binding"]').value.trim(),
	}));
	const steps = Array.from(byID("registration-step-list").children).map((row) => {
		const type = row.querySelector('[data-registration-step="type"]').value;
		const step = { type };
		if (type === "navigate") step.navigate = row.querySelector('[data-registration-step="navigate"]').value.trim();
		if (["type_credential", "click", "submit", "human_checkpoint", "wait_for"].includes(type)) step.candidate_id = row.querySelector('[data-registration-step="candidate"]').value;
		if (type === "type_credential") step.slot = row.querySelector('[data-registration-step="slot"]').value.trim();
		if (type === "human_checkpoint") step.checkpoint_kind = row.querySelector('[data-registration-step="checkpoint"]').value;
		return step;
	});
	const effects = ["creates_account"];
	if (byID("registration-effect-verification").checked) effects.push("sends_verification");
	if (byID("registration-effect-human").checked) effects.push("requires_human_verification");
	return {
		title: byID("registration-title").value.trim(), provider: byID("registration-provider").value.trim(),
		confidence: byID("registration-confidence").value, expires_after: byID("registration-expiry").value.trim(),
		...(stabilityText === "" ? {} : { ui_stability_score: Number(stabilityText) }),
		credential_slots: slots,
		flow: {
			name: byID("registration-flow-name").value.trim(), description: byID("registration-flow-description").value.trim(), steps, effects,
			confirmation_prompt: byID("registration-confirmation-prompt").value.trim(),
			success: {
				origin: byID("registration-success-origin").value, path: byID("registration-success-path").value.trim(),
				proof: "operator_reviewed_deferred",
				operator_reviewed: byID("registration-success-reviewed").checked,
				locator: { role: byID("registration-success-role").value, name: byID("registration-success-name").value.trim() },
			},
		},
		call_controls: {
			approval: "browser_registration_submit", duplicate_prevention: "operator_attestation", on_duplicate: "fail",
			ambiguous_outcome: "stop_without_retry", cleanup_disposition: byID("registration-cleanup").value,
		},
	};
};

byID("registration-draft-form").addEventListener("submit", (event) => {
	event.preventDefault();
	const draft = collectRegistrationDraft();
	if (!draft.title || !draft.expires_after || draft.credential_slots.some((slot) => !slot.slot || !slot.binding) ||
		draft.flow.steps.some((step) => step.type === "navigate" ? !step.navigate : !step.candidate_id && step.type !== "human_checkpoint") ||
		draft.flow.steps.some((step) => step.type === "type_credential" && !step.slot) || !draft.flow.success.origin ||
		!draft.flow.success.locator.role || !draft.flow.success.locator.name || !byID("registration-success-reviewed").checked) {
		showError("Complete every required metadata, symbolic slot, macro step, confirmation, and success-proof field.");
		return;
	}
	sendRegistrationCommand("draft", { draft });
});

byID("registration-draft-confirmed").addEventListener("change", () => {
	byID("registration-review").disabled = !byID("registration-draft-confirmed").checked || state.pendingMutation;
});
byID("registration-review").addEventListener("click", () => {
	if (!byID("registration-draft-confirmed").checked) return;
	sendRegistrationCommand("review", { confirmed: true });
});
byID("registration-finish").addEventListener("click", () => sendRegistrationCommand("finish", { confirmed: true }));
byID("registration-cancel").addEventListener("click", () => sendLifecycleJSON("registration-authoring/cancel", {
	registration_revision: state.renderedPayload.registration_authoring_revision,
}, "Canceling registration authoring and waiting for process-tree teardown…"));

for (const id of ["transaction-review-confirmed", "transaction-prepare-confirmed", "transaction-promote-confirmed", "transaction-recover-confirmed", "transaction-cancel-confirmed"]) byID(id).addEventListener("change", updateControls);

const transactionAuthority = () => ({
	revision: transactionResource()?.revision || "",
	transaction_sha256: transactionResource()?.transaction_sha256 || "",
	human_approved: true,
});

byID("transaction-review").addEventListener("click", () => sendBrowserTransaction("review", transactionAuthority(), "Accepting the exact candidate review…"));
byID("transaction-prepare").addEventListener("click", () => sendBrowserTransaction("prepare", transactionAuthority(), "Preparing and qualifying without promotion…"));
byID("transaction-promote").addEventListener("click", () => sendBrowserTransaction("promote", {
	...transactionAuthority(),
	preparation_sha256: transactionResource()?.preparation?.preparation_sha256 || "",
	qualification_sha256: transactionResource()?.preparation?.qualification_sha256 || "",
}, "Promoting the exact qualified generation without runtime execution…"));
byID("transaction-cancel").addEventListener("click", () => sendBrowserTransaction("cancel", transactionAuthority(), "Canceling the transaction without runtime execution…"));
byID("transaction-inspect-recovery").addEventListener("click", () => sendBrowserTransaction("recovery/inspect", {
	revision: transactionResource()?.revision || "",
}, "Inspecting promotion recovery state without changing the store…"));
byID("transaction-recover").addEventListener("click", () => sendBrowserTransaction("recovery/reconcile", {
	...transactionAuthority(),
	recovery_sha256: transactionResource()?.recovery?.report?.recovery_sha256 || "",
}, "Reconciling only the exact accepted recovery report…"));
byID("transaction-inspect-selected").addEventListener("click", () => sendBrowserTransaction("selected/inspect", {
	revision: transactionResource()?.revision || "",
	selection_sha256: transactionResource()?.promotion?.selection_sha256 || "",
}, "Inspecting the exact selected package without creating approval or runtime state…"));

byID("package-confirmed").addEventListener("change", updateControls);
byID("package-build").addEventListener("click", () => sendLifecycleJSON("package/build", {
  revision: state.renderedPayload.revision,
  confirmed: true,
}, "Building and assessing the reviewed package…"));
byID("authoring-resume").addEventListener("click", () => sendLifecycleJSON("author/resume", {
  revision: state.renderedPayload.revision,
}, "Returning the package failure to authoring…"));

byID("frontier-fields").addEventListener("input", (event) => {
  if (!event.target.matches(".question-answer, .deferral-field, .deferral-toggle")) return;
  state.dirty = true;
  const wrapper = event.target.closest(".question");
  if (event.target.matches(".deferral-toggle")) updateQuestionState(wrapper);
  clearQuestionError(wrapper);
  clearError();
  updateControls();
});

byID("approval-form").addEventListener("submit", (event) => {
  event.preventDefault();
  if (mutationLocked()) return;
  const mode = event.submitter?.dataset.approval;
  if (mode !== "final" && mode !== "incomplete") return;
  sendMutation({
    route: "author/approve",
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
