// What the editor knows about Aurora comes from the compiler, not from a grammar written
// here a second time: the colours are the lexer's semantic tokens and the marks are the
// parser's diagnostics, the same two the language server answers editors with.
//
// The wasm module publishes them on the window. Until it is up they are simply absent, and
// every call below answers with nothing rather than waiting — the document is coloured on
// the next keystroke instead.

const LANGUAGE = 'aurora';
const OWNER = 'aurora';

// How long to sit on a keystroke before analysing. Long enough that typing a word is one
// analysis and not five, short enough to feel like the editor is keeping up.
const SETTLE_MS = 150;

const MARKER_ERROR = 8; // monaco.MarkerSeverity.Error

// call runs one of the analyses the wasm module published, and answers with null while it is
// not there yet or when it could not answer.
function call(name, ...args) {
  const fn = window[name];
  if (typeof fn !== 'function') return null;
  const answer = fn(...args);
  if (typeof answer !== 'string') return null;
  try {
    return JSON.parse(answer);
  } catch (err) {
    console.error(`aurora: ${name} answered with something that is not JSON`, err);
    return null;
  }
}

// The protocol counts lines and characters from zero; Monaco counts lines and columns from
// one. Everything crossing between the two goes through here.
function toProtocol(position) {
  return [position.lineNumber - 1, position.column - 1];
}

// Diagnostics count lines and characters from zero, and Monaco counts both from one.
function toMarker(diagnostic) {
  const { start, end } = diagnostic.range;
  return {
    startLineNumber: start.line + 1,
    startColumn: start.character + 1,
    endLineNumber: end.line + 1,
    endColumn: end.character + 1,
    message: diagnostic.message,
    severity: MARKER_ERROR,
    source: diagnostic.source,
  };
}

function markDocument(monaco, model) {
  // The width is not passed: the analyses read the control themselves, the same one Run
  // reads, so a mark and a result cannot come from two different widths.
  const diagnostics = call('auroraDiagnostics', model.getValue());
  if (diagnostics === null) return;
  monaco.editor.setModelMarkers(model, OWNER, diagnostics.map(toMarker));
}

// What the compiler calls a completion and what Monaco calls one are the same list in a
// different order, so the kinds are matched by name rather than by number.
//
// A kind with no entry is left to Monaco's default, which is a plain text suggestion: an
// item is worth offering even when its icon is not the right one.
const KINDS = {
  5: 'Field',
  6: 'Variable',
  14: 'Keyword',
  22: 'Shape',
};

const SNIPPET_FORMAT = 2;

function toSuggestion(monaco, item, range) {
  const kind = monaco.languages.CompletionItemKind[KINDS[item.kind]];
  const suggestion = {
    label: item.label,
    detail: item.detail,
    documentation: item.documentation,
    kind: kind === undefined ? monaco.languages.CompletionItemKind.Text : kind,
    insertText: item.insertText || item.label,
    range,
  };
  // Monaco names this enum in the singular and its own documentation names it in the plural,
  // so take whichever this build has. Reading the wrong one is not a missing snippet: it
  // throws inside the provider and the suggestions never appear at all.
  const rules = monaco.languages.CompletionItemInsertTextRule || monaco.languages.CompletionItemInsertTextRules;
  if (item.insertTextFormat === SNIPPET_FORMAT && rules) {
    suggestion.insertTextRules = rules.InsertAsSnippet;
  }
  return suggestion;
}

// The legend names what the numbers in the token data mean, and it is asked for rather than
// kept here: its order is the wire format, and a copy would be one more thing to keep in
// step with the lexer.
function semanticProvider(monaco, legend) {
  return {
    getLegend: () => legend,
    provideDocumentSemanticTokens: (model) => {
      const data = call('auroraSemanticTokens', model.getValue());
      if (data === null) return null;
      return { data: new Uint32Array(data) };
    },
    releaseDocumentSemanticTokens: () => {},
  };
}

// Hover answers with lines, and Monaco renders markdown, where a single newline is not a
// break. Each line becomes a paragraph of its own so it reads as it was written.
function hoverProvider() {
  return {
    provideHover: (model, position) => {
      const info = call('auroraHover', model.getValue(), ...toProtocol(position));
      if (!info) return null;
      return { contents: info.split('\n').map((value) => ({ value })) };
    },
  };
}

// What is offered depends on what sits in front of the cursor — right after a dot it is the
// fields of that shape and nothing else — which is why the position goes across too.
function completionProvider(monaco) {
  return {
    triggerCharacters: ['.'],
    provideCompletionItems: (model, position) => {
      const items = call('auroraCompletions', model.getValue(), ...toProtocol(position));
      if (items === null) return { suggestions: [] };

      // The word being typed is what the suggestion replaces. Without it Monaco inserts at
      // the cursor and the half already typed stays where it is.
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };

      return { suggestions: items.map((item) => toSuggestion(monaco, item, range)) };
    },
  };
}

// What the editor needs to be told about the shape of the language: a comment runs to the
// end of the line, and these brackets come in pairs. It is configuration rather than
// knowledge — Monaco closes a brace for you, and the compiler has nothing to say about that.
function configureLanguage(monaco) {
  monaco.languages.setLanguageConfiguration(LANGUAGE, {
    comments: { lineComment: '#-' },
    brackets: [['{', '}'], ['(', ')'], ['[', ']']],
    autoClosingPairs: [
      { open: '{', close: '}' },
      { open: '(', close: ')' },
      { open: '[', close: ']' },
      { open: '"', close: '"' },
    ],
    surroundingPairs: [
      { open: '{', close: '}' },
      { open: '(', close: ')' },
      { open: '[', close: ']' },
      { open: '"', close: '"' },
    ],
  });
}

export function registerAuroraLanguage(monaco, editor) {
  configureLanguage(monaco);

  editor.updateOptions({
    // Semantic colouring is off until it is asked for. A theme can ask for it and this one
    // does not — the editor is on the stock dark theme — so the editor is told directly. The
    // token types are named after what a theme already colours (keyword, number, string),
    // so vs-dark knows what to do with them without a rule of our own.
    'semanticHighlighting.enabled': true,
    // Otherwise Monaco offers every word already in the document, next to what the compiler
    // answered: after a dot, where the fields of a shape are the only thing that can
    // follow, it would still offer "shape" and "ident" because they appear further up.
    wordBasedSuggestions: 'off',
  });

  const legend = call('auroraSemanticLegend') || { tokenTypes: [], tokenModifiers: [] };
  monaco.languages.registerDocumentSemanticTokensProvider(LANGUAGE, semanticProvider(monaco, legend));
  monaco.languages.registerHoverProvider(LANGUAGE, hoverProvider());
  monaco.languages.registerCompletionItemProvider(LANGUAGE, completionProvider(monaco));

  const model = editor.getModel();
  let settling = null;
  const mark = () => {
    clearTimeout(settling);
    settling = setTimeout(() => markDocument(monaco, model), SETTLE_MS);
  };

  editor.onDidChangeModelContent(mark);

  // The width decides what is an error — text of nine bytes fits a sixteen-byte tape and not
  // an eight-byte one — so the marks are stale the moment it changes.
  const control = document.getElementById('tape-size');
  if (control) control.addEventListener('change', mark);

  mark();
}
