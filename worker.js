// Runs the Go engine (automaty.wasm, see wasm.go) off the page's thread. The
// page sends the compiled module first, then one {op, query, mask} at a time.
// A render can't be interrupted from inside: the page terminates the worker.
importScripts('wasm_exec.js');

let ready;
onmessage = async ({ data }) => {
  if (data.module) {
    const go = new Go();
    // run returns once main blocks, with the automaty functions set.
    ready = WebAssembly.instantiate(data.module, go.importObject).then(inst => { go.run(inst); });
    return;
  }
  try {
    await ready;
    const out = automaty[data.op](data.query, data.mask ? new Uint8Array(data.mask) : null);
    postMessage(out, out.body ? [out.body.buffer] : []);
  } catch (err) {
    // A Go panic ends its runtime: the page drops this worker.
    postMessage({ error: String(err), dead: true });
  }
};
