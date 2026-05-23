import {Compartment, type Extension} from "@codemirror/state";
import {EditorView, keymap} from "@codemirror/view";
import {javascript} from "@codemirror/lang-javascript";
import {json} from "@codemirror/lang-json";
import {python} from "@codemirror/lang-python";
import {go} from "@codemirror/lang-go";
import {yaml} from "@codemirror/lang-yaml";
import {basicSetup} from "codemirror";

// Minimal poly types for global exposure
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const g: any = globalThis as any;

type HashChangeCb = (hash: string) => void;
type SaveCb = () => void;

interface EditorEntry {
  view: EditorView;
  onChange?: HashChangeCb;
  onSave?: SaveCb;
  suppressChange: boolean;
}

function langFrom(nameOrFile?: string): Extension | null {
  if (!nameOrFile) return null;
  const s = nameOrFile.toLowerCase();
  const ext = s.includes(".") ? s.substring(s.lastIndexOf(".") + 1) : s;
  switch (ext) {
    case "js":
    case "jsx":
    case "ts":
    case "tsx":
    case "javascript":
    case "typescript":
      return javascript();
    case "json":
      return json();
    case "py":
    case "python":
      return python();
    case "go":
      return go();
    case "yaml":
    case "yml":
      return yaml();
    case "dart":
      // No official CM6 Dart package at time of writing.
      // Fallback: no language extension (plain text)
      return null;
    default:
      return null;
  }
}

const CortadoEditor = {
  _editors: new Map<string, EditorEntry>(),

  _hashText(text: string): string {
    let hash = 0x811c9dc5;
    for (let index = 0; index < text.length; index++) {
      hash ^= text.charCodeAt(index);
      hash = Math.imul(hash, 0x01000193) >>> 0;
    }
    return hash.toString(16).padStart(8, "0");
  },

  init(
    container: HTMLElement,
    id: string,
    languageOrOptions?: string | { language?: string; value?: string },
    onChange?: HashChangeCb,
    onSave?: SaveCb,
  ) {
    const options =
      typeof languageOrOptions === "string"
        ? { language: languageOrOptions }
        : languageOrOptions ?? {};
    const value = options.value ?? "";
    const lang = langFrom(options.language);
    const langCompartment = new Compartment();

    const saveKeymap = [
      {
        key: "Mod-s",
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
        basicSetup,
        keymap.of(saveKeymap),
        langCompartment.of(lang ? [lang] : []),
        EditorView.updateListener.of((update) => {
          if (!update.docChanged) {
            return;
          }
          const entry = this._editors.get(id);
          if (!entry || entry.suppressChange) {
            return;
          }
          if (entry.onChange) {
            entry.onChange(this._hashText(update.state.doc.toString()));
          }
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
    });
  },

  setContent(id: string, text: string, preserveSelection = false): string {
    const entry = this._editors.get(id);
    if (!entry) return "";
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
    if (!entry) return "";
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
};

// Expose as a stable global for Flutter HtmlElementView interop.
if (!g.CortadoEditor) {
  g.CortadoEditor = CortadoEditor;
}

export {};
