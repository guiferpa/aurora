import fs from "fs";

export async function read(args: string[]): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    let buffer = Buffer.from("");

    const reader = fs.createReadStream(args[0]);

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
