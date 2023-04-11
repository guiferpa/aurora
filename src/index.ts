import fs from "fs";

import { Lexer } from "./lexer";
import { Parser } from "./v2/parser";

async function read(path: string): Promise<Buffer> {
	return new Promise((resolve, reject) => {
		let buffer = Buffer.from("");

		const reader = fs.createReadStream(path);

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


;(async () => {
	const buffer = await read('./test.ar');

	const lexer = new Lexer(buffer); // Tokenizer
	const p = new Parser(lexer);
	const astv2 = p.parse();
	console.log("AST v2:", JSON.stringify(astv2, null, 2));

})();
