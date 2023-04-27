import rl from "readline";

export const DEFAULT_PROMPT = ">> ";

export default function repl(): rl.Interface {
  const r = rl.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  r.setPrompt(DEFAULT_PROMPT);
  r.prompt(true);

  return r;
}

