// DSH SDK 0.2.2-rc.4 tool names use lower-case identifiers. Keep this
// adapter separate from the checked-in workbench bundle so it can be reviewed
// and reused when the upstream workbench sources are available.
export const dshToolActions = Object.freeze({
  read: 'read',
  write: 'write',
  edit: 'edit',
  bash: 'command',
  pwsh: 'command',
  glob: 'glob',
  grep: 'grep',
  web_fetch: 'webFetch',
  web_search: 'webSearch',
  subagent: 'agent',
  skill: 'skill',
});

const structuredActions = Object.freeze({
  job_list: 'jobList',
  job_output: 'jobOutput',
  job_kill: 'jobKill',
  knowledge_search: 'knowledgeSearch',
  canvas_get_document: 'canvasRead',
  canvas_get_selection: 'canvasSelection',
  canvas_create_text: 'canvasCreateText',
  canvas_create_html: 'canvasCreateHTML',
  canvas_commit_html: 'canvasCommitHTML',
  canvas_prepare_pptx: 'canvasPreparePPTX',
  canvas_update_node: 'canvasUpdate',
  canvas_delete_node: 'canvasDelete',
  design_catalog_search: 'designCatalogSearch',
  design_catalog_detail: 'designCatalogDetail',
});

export function dshStructuredAction(name) {
  return structuredActions[name] || (/^connector_[0-9a-f]{32}$/.test(name || '') ? 'connectorCall' : undefined);
}

export function isDSHTool(name) {
  return Object.hasOwn(dshToolActions, name) || !!dshStructuredAction(name);
}

export function displayOtherApproval(name, title, summary, fallback) {
  return [summary, title && title !== name && !(summary || '').includes(title) ? title : ''].filter(Boolean).join(' · ') || fallback;
}

// Structured bridge results are kept intact in diagnostics. Extract only
// human-readable text for the normal view; never render the JSON as prose.
export function displayDSHStructuredOutput(output) {
  let value;
  try { value = JSON.parse(output); } catch { return output; }
  const lines = [];
  function extra(item, key) {
    if (Array.isArray(item)) {
      if (!item.length) lines.push(`${key}: []`);
      item.forEach((entry, i) => extra(entry, `${key}[${i + 1}]`));
      return;
    }
    if (item && typeof item === 'object') {
      if (!Object.keys(item).length) lines.push(`${key}: {}`);
      for (const [name, entry] of Object.entries(item)) extra(entry, `${key}.${name}`);
      return;
    }
    lines.push(`${key}: ${item === null ? 'null' : String(item)}`);
  }
  function visit(item) {
    if (item === undefined) return;
    if (Array.isArray(item)) { item.forEach(visit); return; }
    if (typeof item === 'string') { if (item.trim()) lines.push(item.trim()); return; }
    if (!item || typeof item !== 'object') { lines.push(String(item)); return; }
    for (const key of ['title', 'name', 'heading_path', 'text', 'content', 'message', 'summary', 'description', 'source_url']) {
      if (typeof item[key] === 'string' && item[key].trim()) lines.push(item[key].trim());
      else if (key === 'content' || key === 'heading_path') visit(item[key]);
      else if (item[key] !== undefined && typeof item[key] !== 'string') extra(item[key], key);
    }
    if (Number.isSafeInteger(item.page_no) && item.page_no > 0) lines.push(`P. ${item.page_no}`);
    else if (item.page_no !== undefined) extra(item.page_no, 'page_no');
    if (typeof item.output === 'string') {
      try { visit(JSON.parse(item.output)); } catch { visit(item.output); }
    } else if (item.output !== undefined) visit(item.output);
    for (const key of ['items', 'results', 'nodes', 'citation', 'error']) visit(item[key]);
    const handled = new Set(['title', 'name', 'heading_path', 'text', 'content', 'message', 'summary', 'description', 'source_url', 'page_no', 'output', 'items', 'results', 'nodes', 'citation', 'error']);
    for (const [key, entry] of Object.entries(item)) if (!handled.has(key)) extra(entry, key);
  }
  if (typeof value === 'string') return value;
  visit(value);
  return lines.join('\n\n');
}

// A checkpoint or finished state may advance an existing history item without
// increasing the history sequence. Replay it if it arrived during a snapshot.
export function isSameSequenceCompletion(event) {
  return event.type === 'message' && event.value.checkpoint === true ||
    event.type === 'state' && event.value.finish === true;
}

// DSH read offsets are one-based; the existing workbench tools use zero-based.
export function displayReadOffset(toolName, offset) {
  return toolName === 'read' ? offset : offset + 1;
}

export function dshErrorKey(message) {
  if (message === 'model_vision_unavailable') return 'errors.codes.model_vision_unavailable';
  if (message === 'attachment_image_invalid') return 'errors.codes.attachment_image_invalid';
  if (message === 'attachment_unavailable_or_too_large') return 'errors.codes.attachment_unavailable_or_too_large';
  if (message === 'owned_model_index_unavailable') return 'errors.codes.owned_model_index_unavailable';
  if (message === 'dsh_model_request_too_large') return 'errors.codes.dsh_model_request_too_large';
  if (message === 'dsh_compaction_unverified' || message === 'compaction_result_unverified') return 'errors.codes.dsh_compaction_unverified';
  if (message === 'dsh_compaction_usage_invalid') return 'errors.codes.dsh_compaction_usage_invalid';
  if (/^dsh_write_outcome_unknown(?:$|:)/.test(message)) return 'errors.codes.dsh_write_outcome_unknown';
  if (message === 'dsh_canvas_unavailable') return 'errors.codes.dsh_canvas_unavailable';
  if (/\bdesign_template_selection_conflict$/.test(message)) return 'errors.codes.design_template_selection_conflict';
  if (message === 'tool_approval_unavailable') return 'errors.codes.tool_approval_unavailable';
  if (message === 'session_turn_failed') return 'errors.codes.session_turn_failed';
  return undefined;
}

// Strip only the known DSH read envelope for the readable view. The original
// output remains on the tool call for the raw diagnostic record.
export function displayDSHRead(output) {
  return /^<path>[^\n]*<\/path>\n<type>file<\/type>\n<content>\n([\s\S]*)\n<\/content>$/.exec(output)?.[1] ?? output;
}

// Recognize only the SDK's known successful write result. Keep all other
// output unchanged so unexpected results and errors remain visible.
export function displayDSHWrite(output) {
  return /^<path>[^\n]*<\/path>\n<type>file<\/type>\n<content>\nCreated file\n<\/content>$/.test(output)
    ? '文件已创建'
    : output;
}

// Durable platform history names the SDK argument field rawInput; the
// workbench's existing execution components consume input.
export function normalizeDSHToolCalls(calls) {
  return Object.fromEntries(Object.entries(calls || {}).map(([id, call]) => [id,
    call && call.input === undefined && call.rawInput !== undefined
      ? { ...call, input: call.rawInput }
      : call,
  ]));
}

// A history refresh can race an unfinished SDK stream. Keep its live message
// until settlement; a finished turn is always replaced by persisted history.
export function preserveDSHLiveMessage(previous, incoming, active) {
  const live = active && previous.at(-1);
  if (!live || live.role !== 'assistant' || live.endedAt !== undefined) return incoming;
  const index = incoming.findIndex(message => message.id === live.id);
  if (index >= 0 && incoming[index].endedAt !== undefined) return incoming;
  if (!live.dshStream || live.dshStream.closed) {
    return live.dshStep > 0 && live.parts.length === 0 && index < 0 ? [...incoming, live] : incoming;
  }
  if (index < 0) return [...incoming, live];
  return incoming.map((message, i) => i === index
    ? { ...message, parts: live.parts, dshStream: live.dshStream }
    : message);
}

// Keep streamed text in its own tagged part so a durable settlement can
// replace exactly that attempt without disturbing tools or earlier text.
export function applyDSHStream(message, update) {
  const id = update.streamAttemptId;
  const kind = update.sessionUpdate;
  if (kind === 'agent_message_settled' && !id) {
    const sequence = update._meta?.dshMessageSeq;
    if (!Number.isSafeInteger(sequence) || sequence < 0 || sequence <= (message.dshMessageSeq ?? -1)) return message;
    const raw = update._meta?.dshParts;
    const parts = Array.isArray(raw) && raw.filter(part => part && ['text', 'thought'].includes(part.type) && typeof part.text === 'string');
    if (!parts) return message;
    return { ...message, parts: [...message.parts, ...parts], dshMessageSeq: sequence };
  }
  if (typeof id !== 'string' || !id) return message;
  const previous = message.dshStream;
  const revision = update.streamRevision;
  if (kind === 'agent_message_start') {
    if (!Number.isSafeInteger(revision) || (previous && revision <= previous.revision)) return message;
    const parts = previous && !previous.closed
      ? message.parts.filter(part => part.dshAttemptId !== previous.id)
      : message.parts;
    return { ...message, parts, dshStream: { id, revision, index: -1, closed: false } };
  }
  if (kind === 'agent_message_reset') {
    const parts = message.parts.filter(part => part.dshAttemptId !== id);
    return { ...message, parts, dshStream: previous && previous.id === id ? { ...previous, closed: true } : previous };
  }
  if (!previous || previous.id !== id || previous.closed) return message;
  const at = message.parts.findIndex(part => part.dshAttemptId === id);
  if (kind === 'agent_message_chunk' || kind === 'agent_thought_chunk') {
    if (!Number.isSafeInteger(revision) || !Number.isSafeInteger(update.streamIndex) || revision <= previous.revision || update.streamIndex <= previous.index || update.content?.type !== 'text') return message;
    const parts = [...message.parts];
    const type = kind === 'agent_thought_chunk' ? 'thought' : 'text';
    const last = parts.length - 1;
    if (last < 0 || parts[last].dshAttemptId !== id || parts[last].type !== type) parts.push({ type, text: update.content.text, dshAttemptId: id });
    else parts[last] = { ...parts[last], text: parts[last].text + update.content.text };
    return { ...message, parts, dshStream: { ...previous, revision, index: update.streamIndex } };
  }
  if (kind === 'agent_message_settled' && update.content?.type === 'text') {
    if (update._meta?.dshProvisional === true) return message;
    const parts = [...message.parts];
    const settled = Array.isArray(update._meta?.dshParts)
      ? update._meta.dshParts.filter(part => part && ['text', 'thought'].includes(part.type) && typeof part.text === 'string')
      : [{ type: 'text', text: update.content.text }];
    const replacement = settled.map(part => ({ type: part.type, text: part.text, dshAttemptId: id }));
    const end = parts.filter(part => part.dshAttemptId !== id);
    end.splice(at < 0 ? end.length : at, 0, ...replacement);
    return { ...message, parts: end, dshStream: { ...previous, closed: true } };
  }
  return message;
}
