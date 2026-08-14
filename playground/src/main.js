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
    await go.run(instance);
  } catch (err) {
    console.error(err);
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
  $clear.addEventListener('click', () => {
    $output.innerHTML = '';
    $clear.disabled = true;
  });

  init();
});
