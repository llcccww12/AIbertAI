const operationPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const keyFor = (sessionId) => `aibertai.compact.${sessionId}`;

// Keep the identity until the server confirms a terminal response. Reconnects
// and lost replies must retry the same operation rather than start another.
export function beginCompactOperation(storage, sessionId, createId) {
  const key = keyFor(sessionId);
  let operationId = storage.getItem(key);
  if (operationId == null) {
    operationId = createId();
    if (!operationPattern.test(operationId)) throw new Error("invalid_compaction_operation");
    storage.setItem(key, operationId);
  }
  if (!operationPattern.test(operationId) || storage.getItem(key) !== operationId)
    throw new Error("compaction_operation_unavailable");
  return operationId;
}

export function finishCompactOperation(storage, sessionId, operationId) {
  const key = keyFor(sessionId);
  if (storage.getItem(key) === operationId) storage.removeItem(key);
}

export function applyDSHCompaction(message, id, part) {
  const index = message.parts.findIndex(item => item.type === 'compaction' && item.compactionId === id);
  const next = { ...part, compactionId: id };
  if (index < 0) return { ...message, dshExecution: true, parts: [...message.parts, next] };
  if (message.parts[index].status !== 'in_progress' || part.status === 'in_progress') return message;
  return { ...message, parts: message.parts.map((item, i) => i === index ? next : item) };
}
