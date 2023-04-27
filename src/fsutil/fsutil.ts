import fs from "fs";

export async function read(arg: string): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    let buffer = Buffer.from("");

    const reader = fs.createReadStream(arg);

    reader.on('data', (chunk: Buffer) => {
      buffer = Buffer.concat([buffer, chunk]);
    });

    reader.on('error', (err) => {
      reject(err);
    });

    reader.on('close', () => {
      resolve(buffer);
    });
  });
}
