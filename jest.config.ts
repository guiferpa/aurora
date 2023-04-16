import type {Config} from "@jest/types";

const config: Config.InitialOptions = {
	verbose: true,
  testPathIgnorePatterns: [ 'dist/' ],
	transform: {
		'^.+\\.tsx?$': 'ts-jest',
	},
};
export default config;
