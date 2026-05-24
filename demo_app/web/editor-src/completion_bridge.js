export function shouldDropCompletionResult(entry, pending) {
  if (!entry) {
    return true;
  }

  const currentPos = entry.view.state.selection.main.head;
  return entry.cursorVersion !== pending.cursorVersion || currentPos !== pending.pos;
}
