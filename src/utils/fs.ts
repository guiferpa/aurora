import fs from "fs";

export async function read(raw: string): Promise<Buffer> {
  const [filename] = noext(raw);

  return new Promise((resolve, reject) => {
    let buffer = Buffer.from("");

    const reader = fs.createReadStream(`${filename}.ar`);

    reader.on("data", (chunk: Buffer) => {
      buffer = Buffer.concat([buffer, chunk]);
    });

    reader.on("error", (err) => {
      reject(err);
    });

    reader.on("close", () => {
      resolve(buffer);
    });
  });
}

export async function write(filename: string, content: string[]) {
  const writer = fs.createWriteStream(filename, {
    flags: "w",
  });

  for (const inst of content)
    writer.write(Buffer.concat([Buffer.from(inst), Buffer.from("\n")]));

  writer.close();
}

export function noext(raw: string): [string, string] {
  const [ext, ...splitted] = raw.split(".").reverse();
  return splitted.length > 0 ? [splitted.reverse().join("."), ext] : [ext, ""];
}
