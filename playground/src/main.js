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
    console.clear();
    await go.run(instance);
  } catch (err) {
    console.error(err);
  }
}

document.addEventListener("DOMContentLoaded", () => {
  init();
});
