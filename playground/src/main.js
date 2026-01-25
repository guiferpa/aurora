function print(result) {
  return `(print) ${toHex(result)}`;
}

function echo(result) {
  return `(echo) ${toText(result)}`;
}

const builtins = {
  print,
  echo,
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

function toText(result) {
  const decoder = new TextDecoder('utf-8');
  return decoder.decode(result);
}

function toHex(result) {
  return Array.from(result).map(b => b.toString(16).padStart(2, '0')).join(' ');
}

function resultToText(result) {
  const body = toHex(result);
  const len = result.length;
  return `= (${len / 8}) ${body}`;
}

function renderOutput(text) {
  const $output = document.getElementById('output');
  const code = document.createElement('code');
  code.innerText = text;
  const li = document.createElement('li');
  li.appendChild(code);
  $output.appendChild(li);
}

window.evalResultHandler = (result, builtin) => {
  console.log(result, builtin);
  const fn = builtins[builtin];
  const text = (!fn) ? resultToText(result) : fn(result);
  renderOutput(text);
}

const outputMutationsHandler = (ref) => (muts) => {
  for (const mut of muts) {
    if (mut.type === 'childList') {
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

  init();
});
