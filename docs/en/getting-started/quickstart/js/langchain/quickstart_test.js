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

import fs from "fs";
import path from "path";
import assert from "assert/strict";
import { main as runAgent } from "./quickstart.js";

const GOLDEN_FILE_PATH = path.resolve("../../golden.txt");

async function runTests() {
  const capturedOutput = [];
  const originalLog = console.log;
  console.log = (msg) => {
    capturedOutput.push(msg);
  };

  try {
    await runAgent();
    const actualOutput = capturedOutput.join("\n");
    console.log = originalLog;

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

    process.exit(0);
  } catch (error) {
    console.log = originalLog;
    console.error(error.message);
    process.exit(1);
  }
}

runTests();
