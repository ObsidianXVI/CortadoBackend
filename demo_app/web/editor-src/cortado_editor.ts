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
import {lintKeymap} from '@codemirror/lint';
import {highlightSelectionMatches, searchKeymap} from '@codemirror/search';
import {javascript} from '@codemirror/lang-javascript';
import {json} from '@codemirror/lang-json';
import {python} from '@codemirror/lang-python';
import {go} from '@codemirror/lang-go';
import {yaml} from '@codemirror/lang-yaml';
import {shouldDropCompletionResult} from './completion_bridge.js';

// Minimal poly types for global exposure
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const g: any = globalThis as any;

const editorBaseExtensions: Extension[] = [
  lineNumbers(),
  highlightActiveLineGutter(),
  highlightSpecialChars(),
  history(),
  foldGutter(),
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
}

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

  dispose(id: string) {
    const entry = this._editors.get(id);
    if (!entry) return;
    entry.view.destroy();
    this._editors.delete(id);
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
  },
};

// Expose as a stable global for Flutter HtmlElementView interop.
if (!g.CortadoEditor) {
  g.CortadoEditor = CortadoEditor;
}

export {};
