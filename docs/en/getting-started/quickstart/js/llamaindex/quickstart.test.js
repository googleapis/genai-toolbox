// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { describe, test, before, after } from "node:test";
import assert from "node:assert/strict";
import fs from "fs";
import path from "path";
import { main as runAgent } from "./quickstart.js";

const GOLDEN_FILE_PATH = path.resolve("../../golden.txt");

describe("LangChain Quickstart Agent", () => {
  let capturedOutput = [];
  let originalLog;

  before(() => {
    originalLog = console.log;
    console.log = (msg) => {
      capturedOutput.push(msg);
    };
  });

  after(() => {
    console.log = originalLog;
  });

  test("Assert", async () => {
    await runAgent();
    const actualOutput = capturedOutput.join("\n");

    assert.ok(
      actualOutput.length > 0,
      "Assertion Failed: Script ran successfully but produced no output."
    );

    const goldenFile = fs.readFileSync(GOLDEN_FILE_PATH, "utf8");
    const keywords = goldenFile.split("\n").filter((kw) => kw.trim() !== "");
    const missingKeywords = [];

    for (const keyword of keywords) {
      if (!actualOutput.toLowerCase().includes(keyword.toLowerCase())) {
        missingKeywords.push(keyword);
      }
    }

    assert.ok(
      missingKeywords.length === 0,
      `Assertion Failed: The following keywords were missing from the output: [${missingKeywords.join(", ")}]`
    );
  });
});