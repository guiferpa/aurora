import type {Config} from "@jest/types";

const config: Config.InitialOptions = {
  verbose: true,
  testPathIgnorePatterns: ['dist/'],
  modulePathIgnorePatterns: ['dist/'],
  roots: [
    '<rootDir>/src'
  ],
  transform: {
    '^.+\\.tsx?$': 'ts-jest',
  },
  moduleNameMapper: {
    '@/(.*)': '<rootDir>/src/$1'
  }
};
export default config;
