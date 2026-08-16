import { registerAuroraLanguage } from './language.js';

function nothing(result) {
  return "<nothing>";
}

// printb, printd and printc are three readings of the same tape, and each one formats its
// own line. What arrives here is that line, already finished.
const renderers = {
  output: (result) => toText(result).replace(/\n$/, ''),
}

async function init() {
  const go = new Go();

  if (!WebAssembly.instantiateStreaming) {
    // polyfill
    WebAssembly.instantiateStreaming = async (resp, importObject) => {
      const source = await (await resp).arrayBuffer();
      return await WebAssembly.instantiate(source, importObject);
    };
  }

  try {
    const { instance } = await WebAssembly.instantiateStreaming(
      fetch("main.wasm"),
      go.importObject,
    );
    document.getElementById("runner").disabled = false;
    // go.run never resolves: the Go program blocks so that what it published stays callable.
    // It does run main up to that block before returning, which is when the analyses and the
    // runner appear.
    go.run(instance).catch((err) => console.error(err));
  } catch (err) {
    console.error(err);
  }
}

// whenEditorReady runs fn with the editor and the monaco it belongs to, whichever of the two
// finished loading first: this module and the editor's loader race, and either order is fine.
//
// Both are handed over by the event rather than read off the window, because the loader sets
// the global at a moment of its own — reading it here found it undefined about as often as
// not.
function whenEditorReady(fn) {
  document.addEventListener('editor-ready', (event) => fn(event.detail), { once: true });
  if (window.editor && window.monaco) {
    fn({ monaco: window.monaco, editor: window.editor });
  }
}

function toDecimal(result) {
  return Array.from(result).map(b => b.toString(10)).join(' ');
}

function toText(result) {
  const decoder = new TextDecoder('utf-8');
  return decoder.decode(result);
}

function toHex(result) {
  return Array.from(result).map(b => b.toString(16).padStart(2, '0')).join(' ');
}

function fromResult(result) {
  if (result.length === 0) return nothing();
  const body = toDecimal(result);
  const len = result.length;
  return `= (${len}) ${body}`;
}

function renderOutput(text) {
  const $output = document.getElementById('output');
  const code = document.createElement('code');
  code.innerText = text;
  const li = document.createElement('li');
  li.appendChild(code);
  $output.appendChild(li);
}

function renderError(text) {
  const $output = document.getElementById('output');
  const code = document.createElement('code');
  code.innerText = text;
  const li = document.createElement('li');
  li.classList.add('error');
  li.appendChild(code);
  $output.appendChild(li);
}

window.evalResultHandler = (result, kind) => {
  const render = renderers[kind];
  const text = (!render) ? fromResult(result) : render(result);
  renderOutput(text);
}

window.evalErrorHandler = (error) => {
  const text = `(error) ${toText(error)}`;
  renderError(text);
}

const outputMutationsHandler = (ref) => (muts) => {
  for (const mut of muts) {
    if (mut.type === 'childList') {

      const $clear = document.getElementById('clear');
      if (ref.children.length > 0) {
        $clear.disabled = false;
      }

      for (const node of mut.addedNodes) {
        if (node.nodeType === Node.ELEMENT_NODE && node.tagName === 'LI') {
          ref.scrollTo(0, ref.scrollHeight);
        }
      }
    }
  }
}

document.addEventListener("DOMContentLoaded", () => {
  console.clear();

  const $output = document.getElementById('output');
  const mob = new MutationObserver(outputMutationsHandler($output));
  mob.observe($output, { childList: true });

  const $clear = document.getElementById('clear');
  const clearOutput = () => {
    $output.innerHTML = '';
    $clear.disabled = true;
  };
  $clear.addEventListener('click', clearOutput);

  // The width of a tape decides what a program means — at one byte 255 + 1 is 0 — so what
  // is on screen stops being the output of the program the moment it changes. Rather than
  // leave the two disagreeing, the program runs again at the new width.
  //
  // The button is disabled until the wasm module is ready, and a click on a disabled button
  // does nothing, so this cannot run before there is anything to run it with.
  const $tapeSize = document.getElementById('tape-size');
  $tapeSize.addEventListener('change', () => {
    clearOutput();
    document.getElementById('runner').click();
  });

  const running = init();

  // The editor is told what the language is only once both halves are up: the colours and
  // the marks come from the wasm module, and the editor is what they are attached to.
  let registered = false;
  whenEditorReady(async ({ monaco, editor }) => {
    await running;
    if (registered) return; // the event and the check above can both fire
    registered = true;
    registerAuroraLanguage(monaco, editor);
  });
});
