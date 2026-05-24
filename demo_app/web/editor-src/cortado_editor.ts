import {Compartment, EditorState, type Extension} from '@codemirror/state';
import {
  EditorView,
  crosshairCursor,
  drawSelection,
  dropCursor,
  highlightActiveLine,
  highlightActiveLineGutter,
  highlightSpecialChars,
  keymap,
  lineNumbers,
  rectangularSelection,
} from '@codemirror/view';
import {hoverTooltip, type Tooltip} from '@codemirror/tooltip';
import {
  autocompletion,
  closeBrackets,
  closeBracketsKeymap,
  completionKeymap,
  type Completion,
  type CompletionResult,
} from '@codemirror/autocomplete';
import {defaultKeymap, history, historyKeymap} from '@codemirror/commands';
import {
  bracketMatching,
  defaultHighlightStyle,
  foldGutter,
  foldKeymap,
  indentOnInput,
  syntaxHighlighting,
} from '@codemirror/language';
import {
  lintKeymap,
  lintGutter,
  setDiagnostics,
  type Diagnostic,
} from '@codemirror/lint';
import {highlightSelectionMatches, searchKeymap} from '@codemirror/search';
import {javascript} from '@codemirror/lang-javascript';
import {json} from '@codemirror/lang-json';
import {python} from '@codemirror/lang-python';
import {go} from '@codemirror/lang-go';
import {yaml} from '@codemirror/lang-yaml';
import {shouldDropCompletionResult} from './completion_bridge.js';
import DOMPurify from 'dompurify';
import {marked} from 'marked';

// Minimal poly types for global exposure
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const g: any = globalThis as any;

const editorBaseExtensions: Extension[] = [
  lineNumbers(),
  highlightActiveLineGutter(),
  highlightSpecialChars(),
  history(),
  foldGutter(),
  // Show lint markers in the gutter and enable underline/highlight styles.
  lintGutter(),
  drawSelection(),
  dropCursor(),
  EditorState.allowMultipleSelections.of(true),
  indentOnInput(),
  syntaxHighlighting(defaultHighlightStyle, {fallback: true}),
  bracketMatching(),
  closeBrackets(),
  rectangularSelection(),
  crosshairCursor(),
  highlightActiveLine(),
  highlightSelectionMatches(),
  keymap.of([
    ...closeBracketsKeymap,
    ...defaultKeymap,
    ...searchKeymap,
    ...historyKeymap,
    ...foldKeymap,
    ...completionKeymap,
    ...lintKeymap,
  ]),
];

type HashChangeCb = (hash: string) => void;
type SaveCb = () => void;
type LSPRequestFn = (params: {
  editorId: string;
  requestId: number;
  position: { line: number; character: number };
  // Optional hint for the host: current line text and prefix before the cursor
  lineText?: string;
  prefix?: string;
}) => void;

interface EditorEntry {
  view: EditorView;
  onChange?: HashChangeCb;
  onSave?: SaveCb;
  suppressChange: boolean;
  // Bumps whenever the cursor moves. Used to drop stale completion results.
  cursorVersion: number;
  // Compartment to (re)configure completion source.
  completionCompartment: Compartment;
  // Compartment to (re)configure read-only state.
  readOnlyCompartment: Compartment;
}

// Shape accepted from the host (Dart) for diagnostics. Supports either
// absolute character offsets (from/to) or LSP-style ranges.
type HostDiagnostic = {
  from?: number;
  to?: number;
  range?: {
    start: { line: number; character: number };
    end: { line: number; character: number };
  };
  severity?: 'error' | 'warning' | 'info' | 'hint' | 1 | 2 | 3 | 4;
  message: string;
  source?: string;
};

function langFrom(nameOrFile?: string): Extension | null {
  if (!nameOrFile) return null;
  const s = nameOrFile.toLowerCase();
  const ext = s.includes('.') ? s.substring(s.lastIndexOf('.') + 1) : s;
  switch (ext) {
    case 'js':
    case 'jsx':
    case 'ts':
    case 'tsx':
    case 'javascript':
    case 'typescript':
      return javascript();
    case 'json':
      return json();
    case 'py':
    case 'python':
      return python();
    case 'go':
      return go();
    case 'yaml':
    case 'yml':
      return yaml();
    case 'dart':
      // No official CM6 Dart package at time of writing.
      // Fallback: no language extension (plain text)
      return null;
    default:
      return null;
  }
}

const CortadoEditor = {
  _editors: new Map<string, EditorEntry>(),
  _pendingCompletions: new Map<
    number,
    {
      resolve: (r: CompletionResult | null) => void;
      editorId: string;
      pos: number;
      from: number;
      cursorVersion: number;
    }
  >(),
  _pendingHovers: new Map<
    number,
    {
      resolve: (t: Tooltip | null) => void;
      editorId: string;
      pos: number;
      cursorVersion: number;
    }
  >(),
  _pendingDefinitions: new Map<
    number,
    {
      editorId: string;
      pos: number;
      cursorVersion: number;
    }
  >(),
  _reqCounter: 1,

  _hashText(text: string): string {
    let hash = 0x811c9dc5;
    for (let index = 0; index < text.length; index++) {
      hash ^= text.charCodeAt(index);
      hash = Math.imul(hash, 0x01000193) >>> 0;
    }
    return hash.toString(16).padStart(8, '0');
  },

  init(
    container: HTMLElement,
    id: string,
    languageOrOptions?: string | { language?: string; value?: string },
    onChange?: HashChangeCb,
    onSave?: SaveCb,
  ) {
    const options =
      typeof languageOrOptions === 'string'
        ? { language: languageOrOptions }
        : languageOrOptions ?? {};
    const value = options.value ?? '';
    const lang = langFrom(options.language);
    const langCompartment = new Compartment();
    const completionCompartment = new Compartment();
    const readOnlyCompartment = new Compartment();

    const saveKeymap = [
      {
        key: 'Mod-s',
        preventDefault: true,
        run: () => {
          const entry = this._editors.get(id);
          if (entry && entry.onSave) {
            entry.onSave();
          }
          return true;
        },
      },
    ];

    const view = new EditorView({
      doc: value,
      extensions: [
        editorBaseExtensions,
        keymap.of(saveKeymap),
        langCompartment.of(lang ? [lang] : []),
        readOnlyCompartment.of([
          EditorState.readOnly.of(false),
          EditorView.editable.of(true),
        ]),
        // Hover support (textDocument/hover), debounced by 500ms
        hoverTooltip(this._hoverSourceFactory(id).bind(this), {
          hoverTime: 500,
          hideOnChange: true,
        }),
        // Track changes and cursor movement.
        EditorView.updateListener.of((update) => {
          const entry = this._editors.get(id);
          if (update.selectionSet && entry) {
            entry.cursorVersion++;
          }
          if (!update.docChanged) {
            return;
          }
          if (!entry || entry.suppressChange) {
            return;
          }
          // Doc changes imply cursor motion in most cases; bump version too.
          entry.cursorVersion++;
          if (entry.onChange) {
            entry.onChange(this._hashText(update.state.doc.toString()));
          }
        }),
        // Install the completion source; it is a no-op until the host
        // provides window._cortadoLSPRequest (Dart side).
        completionCompartment.of(
          autocompletion({
            override: [this._completionSourceFactory(id).bind(this)],
          }),
        ),
        // Ctrl+Click go-to-definition (textDocument/definition)
        EditorView.domEventHandlers({
          mousedown: (event, view) => {
            const me = event as MouseEvent;
            if (me.button !== 0) return false;
            if (!me.ctrlKey) return false;
            const pos = view.posAtCoords({ x: me.clientX, y: me.clientY }, false);
            if (pos == null) return false;
            const line = view.state.doc.lineAt(pos);
            const requestId = this._reqCounter++;
            const entry = this._editors.get(id);
            const cursorVersion = entry ? entry.cursorVersion : 0;
            this._pendingDefinitions.set(requestId, {
              editorId: id,
              pos,
              cursorVersion,
            });
            const defReq: LSPRequestFn | undefined = (g as any)._cortadoLSPDefinitionRequest;
            if (defReq) {
              defReq({
                editorId: id,
                requestId,
                position: { line: line.number - 1, character: pos - line.from },
              });
            }
            // Prevent default text selection when ctrl-clicking.
            me.preventDefault();
            return true;
          },
        }),
      ],
      parent: container,
    });

    // Stash the Compartment on the view for later reconfiguration.
    (view as any)._cortadoLangCompartment = langCompartment;
    this._editors.set(id, {
      view,
      onChange,
      onSave,
      suppressChange: false,
      cursorVersion: 0,
      completionCompartment,
      readOnlyCompartment,
    });

    // Ensure a global result sink exists once per page.
    this._ensureResultSinkInstalled();
  },

  setContent(id: string, text: string, preserveSelection = false): string {
    const entry = this._editors.get(id);
    if (!entry) return '';
    const view = entry.view;
    const savedSelection = view.state.selection.main;
    entry.suppressChange = true;
    try {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: text },
      });
      if (preserveSelection) {
        const nextAnchor = Math.min(savedSelection.anchor, view.state.doc.length);
        const nextHead = Math.min(savedSelection.head, view.state.doc.length);
        view.dispatch({
          selection: { anchor: nextAnchor, head: nextHead },
        });
      }
    } finally {
      entry.suppressChange = false;
    }
    return this._hashText(text);
  },

  getContent(id: string): string {
    const entry = this._editors.get(id);
    if (!entry) return '';
    return entry.view.state.doc.toString();
  },

  setLanguage(id: string, language: string) {
    const entry = this._editors.get(id);
    if (!entry) return;
    const lang = langFrom(language);
    const view = entry.view;
    const comp: Compartment | undefined = (view as any)._cortadoLangCompartment;
    if (comp) {
      view.dispatch({ effects: comp.reconfigure(lang ? [lang] : []) });
    }
  },

  setReadOnly(id: string, readOnly: boolean) {
    const entry = this._editors.get(id);
    if (!entry) return;
    entry.view.dispatch({
      effects: entry.readOnlyCompartment.reconfigure([
        EditorState.readOnly.of(readOnly),
        EditorView.editable.of(!readOnly),
      ]),
    });
  },

  dispose(id: string) {
    const entry = this._editors.get(id);
    if (!entry) return;
    entry.view.destroy();
    this._editors.delete(id);
  },

  // Replace diagnostics for an editor. Accepts either absolute offsets or
  // LSP-style line/character ranges. Passing an empty array clears diagnostics.
  setDiagnostics(id: string, diagnostics: HostDiagnostic[] | null | undefined) {
    const entry = this._editors.get(id);
    if (!entry) return;
    const view = entry.view;
    const docLen = view.state.doc.length;

    const offsetAt = (lineZeroBased: number, ch: number): number => {
      // Clamp inputs conservatively; CodeMirror lines are 1-based.
      const line = Math.max(0, lineZeroBased | 0);
      const lineObj = view.state.doc.line(Math.min(line + 1, view.state.doc.lines));
      const col = Math.max(0, ch | 0);
      return Math.min(lineObj.from + col, docLen);
    };

    const mapSeverity = (
      sev: HostDiagnostic['severity'],
    ): Diagnostic['severity'] | undefined => {
      if (sev === 1 || sev === 'error') return 'error';
      if (sev === 2 || sev === 'warning') return 'warning';
      if (sev === 3 || sev === 'info') return 'info';
      if (sev === 4 || sev === 'hint') return 'hint';
      return undefined;
    };

    const toCM = (d: HostDiagnostic): Diagnostic | null => {
      let from = typeof d.from === 'number' ? d.from : undefined;
      let to = typeof d.to === 'number' ? d.to : undefined;
      if ((from == null || to == null) && d.range) {
        from = offsetAt(d.range.start.line, d.range.start.character);
        to = offsetAt(d.range.end.line, d.range.end.character);
      }
      if (from == null || to == null) return null;
      // Normalize/clamp.
      from = Math.max(0, Math.min(from | 0, docLen));
      to = Math.max(from, Math.min(to | 0, docLen));
      if (!d.message) return null;
      return {
        from,
        to,
        message: d.message,
        severity: mapSeverity(d.severity) ?? 'error',
        source: d.source,
      };
    };

    const cmDiags: Diagnostic[] = (diagnostics ?? [])
      .map(toCM)
      .filter((x): x is Diagnostic => !!x);

    setDiagnostics(view, cmDiags);
  },

  // Create a completion source bound to an editor id.
  // - Debounces 150ms before sending the request.
  // - Drops results if the cursor moved before the result arrives.
  // - Requires the host to define window._cortadoLSPRequest(params) and to
  //   call window._cortadoLSPResult(requestId, items) when ready.
  _completionSourceFactory(id: string) {
    return function completionSource(this: typeof CortadoEditor, ctx): Promise<CompletionResult | null> {
      const entry = this._editors.get(id);
      if (!entry) return Promise.resolve(null);
      const reqFn: LSPRequestFn | undefined = (g as any)._cortadoLSPRequest;
      if (!reqFn) return Promise.resolve(null);

      const pos = ctx.pos;
      const line = ctx.state.doc.lineAt(pos);
      const prefixMatch = ctx.matchBefore(/[A-Za-z0-9_]+/);
      const from = prefixMatch ? prefixMatch.from : pos;

      // Snapshot cursor version to detect staleness.
      const versionAtRequest = entry.cursorVersion;
      const requestId = this._reqCounter++;

      return new Promise<CompletionResult | null>((resolve) => {
        // Debounce 150ms before actually dispatching to Dart.
        const timer = setTimeout(() => {
          const freshEntry = this._editors.get(id);
          if (!freshEntry) {
            resolve(null);
            return;
          }
          // If the cursor moved during debounce, abort.
          if (freshEntry.cursorVersion !== versionAtRequest) {
            resolve(null);
            return;
          }

          // Register resolver; Dart will call back into _cortadoLSPResult.
          this._pendingCompletions.set(requestId, {
            resolve,
            editorId: id,
            pos,
            from,
            cursorVersion: versionAtRequest,
          });

          // Dispatch request to host (Dart).
          reqFn({
            editorId: id,
            requestId,
            position: { line: line.number - 1, character: pos - line.from },
            lineText: line.text,
            prefix: ctx.state.sliceDoc(from, pos),
          });
        }, 150);

        // Note: If the editor is disposed while waiting, the pending entry
        // will be ignored on result due to missing editor; no special hook here.
        void timer; // appease TS about unused var when compiled
      });
    };
  },

  _ensureResultSinkInstalled() {
    if ((g as any)._cortadoLSPResult) return;
    // Result handler called by Dart with already-mapped CodeMirror completions.
    (g as any)._cortadoLSPResult = (requestId: number, items: Completion[]) => {
      const pending = this._pendingCompletions.get(requestId);
      if (!pending) return;
      this._pendingCompletions.delete(requestId);

      const entry = this._editors.get(pending.editorId);
      if (shouldDropCompletionResult(entry, pending)) {
        pending.resolve(null);
        return;
      }

      pending.resolve({ from: pending.from, options: items });
    };

    // Hover result handler called by the host.
    (g as any)._cortadoLSPHoverResult = (
      requestId: number,
      result:
        | null
        | {
            markdown?: string | null;
            plaintext?: string | null;
          },
    ) => {
      const pending = this._pendingHovers.get(requestId);
      if (!pending) return;
      this._pendingHovers.delete(requestId);
      const entry = this._editors.get(pending.editorId);
      if (!entry || entry.cursorVersion !== pending.cursorVersion) {
        pending.resolve(null);
        return;
      }

      const html = result?.markdown
        ? marked.parse(result.markdown)
        : result?.plaintext
        ? // Escape plaintext by assigning to textContent below.
          null
        : null;

      const tooltip: Tooltip = {
        pos: pending.pos,
        create() {
          const dom = document.createElement('div');
          dom.className = 'cm-tooltip-cortado-hover';
          if (html != null) {
            dom.innerHTML = DOMPurify.sanitize(html);
          } else if (result?.plaintext) {
            dom.textContent = result.plaintext;
          } else {
            dom.textContent = '';
          }
          return { dom };
        },
      };
      pending.resolve(tooltip);
    };

    // Definition result handler. If the returned location references the same
    // editorId, move the cursor there. Otherwise, the host is expected to
    // handle cross-file navigation in its own UI.
    (g as any)._cortadoLSPDefinitionResult = (
      requestId: number,
      result:
        | null
        | {
            editorId?: string | null;
            range?: { start: { line: number; character: number } } | null;
          }
        | Array<{
            editorId?: string | null;
            range?: { start: { line: number; character: number } } | null;
          }>,
    ) => {
      const pending = this._pendingDefinitions.get(requestId);
      if (!pending) return;
      this._pendingDefinitions.delete(requestId);
      const first = Array.isArray(result) ? result[0] : result;
      if (!first) return;
      const targetEditorId = first.editorId ?? pending.editorId;
      const entry = this._editors.get(targetEditorId);
      if (!entry) return;
      if (entry.cursorVersion !== pending.cursorVersion) return;
      const view = entry.view;
      const start = first.range?.start;
      if (!start) return;
      const line = Math.max(0, start.line | 0);
      const ch = Math.max(0, start.character | 0);
      const lineObj = view.state.doc.line(Math.min(line + 1, view.state.doc.lines));
      const pos = Math.min(lineObj.from + ch, view.state.doc.length);
      view.dispatch({
        selection: { anchor: pos },
        effects: EditorView.scrollIntoView(pos, { y: 'center' as any }),
      });
    };
  },

  // Build a hover source bound to an editor id. Dispatches a hover request
  // to the host and resolves when the host replies via _cortadoLSPHoverResult.
  _hoverSourceFactory(id: string) {
    return function hoverSource(this: typeof CortadoEditor, view: EditorView, pos: number) {
      const reqFn: LSPRequestFn | undefined = (g as any)._cortadoLSPHoverRequest;
      if (!reqFn) return null;
      const entry = this._editors.get(id);
      if (!entry) return null;
      const line = view.state.doc.lineAt(pos);
      const requestId = this._reqCounter++;
      const cursorVersion = entry.cursorVersion;
      return new Promise<Tooltip | null>((resolve) => {
        this._pendingHovers.set(requestId, {
          resolve,
          editorId: id,
          pos,
          cursorVersion,
        });
        reqFn({
          editorId: id,
          requestId,
          position: { line: line.number - 1, character: pos - line.from },
        });
      });
    };
  },
};

// Expose as a stable global for Flutter HtmlElementView interop.
if (!g.CortadoEditor) {
  g.CortadoEditor = CortadoEditor;
}

export {};
