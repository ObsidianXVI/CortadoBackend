export function shouldDropCompletionResult(entry, pending) {
  if (!entry) {
    return true;
  }

  const currentPos = entry.view.state.selection.main.head;
  return entry.cursorVersion !== pending.cursorVersion || currentPos !== pending.pos;
}

const ignoredInlineCompletionKeys = new Set([
  'Alt',
  'CapsLock',
  'ContextMenu',
  'Control',
  'Fn',
  'FnLock',
  'Hyper',
  'Meta',
  'NumLock',
  'ScrollLock',
  'Shift',
  'Super',
  'Symbol',
  'SymbolLock',
]);

export function shouldCancelInlineCompletionForKey(event) {
  if (!event || event.defaultPrevented || event.isComposing) {
    return false;
  }

  if (event.key === 'Tab') {
    return false;
  }

  if (ignoredInlineCompletionKeys.has(event.key)) {
    return false;
  }

  return true;
}

export function applyInlineCompletionText(doc, pos, completion) {
  if (!completion) {
    return doc;
  }

  const clampedPos = Math.max(0, Math.min(pos | 0, doc.length));
  return `${doc.slice(0, clampedPos)}${completion}${doc.slice(clampedPos)}`;
}
