import { spawn } from "child_process";
import fs from "fs";
import path from "path";

const SCRIPT_TO_TEST = path.resolve("./quickstart.js");
const GOLDEN_FILE_PATH = path.resolve("../../golden.txt");

function runScript() {
  console.log("    quickstart_test.js:--: Running quickstart.js...");
  return new Promise((resolve, reject) => {
    const child = spawn("node", [SCRIPT_TO_TEST]);
    let output = "";
    let errorOutput = "";

    child.stdout.on("data", (data) => {
      output += data.toString();
    });

    child.stderr.on("data", (data) => {
      errorOutput += data.toString();
    });

    child.on("close", (code) => {
      console.log(
        `    quickstart_test.js:--: --- SCRIPT OUTPUT ---\n${output}`
      );
      if (errorOutput) {
        console.log(
          `    quickstart_test.js:--: --- SCRIPT STDERR ---\n${errorOutput}`
        );
      }
      if (code !== 0) {
        const errorMessage = `Script execution failed with error: exit code ${code}\n--- STDERR ---\n${errorOutput}`;
        reject(new Error(errorMessage));
      } else {
        resolve(output);
      }
    });
  });
}

function validateOutput(actualOutput) {
  if (actualOutput.length === 0) {
    throw new Error("Script ran successfully but produced no output.");
  }
  console.log(
    "    quickstart_test.js:--: Primary assertion passed: Script ran successfully and produced output."
  );

  console.log(
    "    quickstart_test.js:--: --- Checking for essential keywords from golden.txt ---"
  );
  try {
    const goldenFile = fs.readFileSync(GOLDEN_FILE_PATH, "utf8");
    const keywords = goldenFile.split("\n").filter((kw) => kw.trim() !== "");

    for (const keyword of keywords) {
      if (actualOutput.includes(keyword)) {
        console.log(
          `    quickstart_test.js:--: Keyword check: Found keyword '${keyword}' in output.`
        );
      } else {
        console.log(
          `    quickstart_test.js:--: Keyword check: Did not find keyword '${keyword}' in output.`
        );
      }
    }
  } catch (err) {
    console.log(
      `    quickstart_test.js:--: Warning: Could not read golden.txt to check for keywords: ${err.message}`
    );
  }
}

async function runTests() {
  const testName = "TestAgentOutputAndKeywords";
  console.log(`=== RUN   ${testName}`);
  const startTime = process.hrtime.bigint();

  try {
    const actualOutput = await runScript();
    validateOutput(actualOutput);

    const endTime = process.hrtime.bigint();
    const duration = (Number(endTime - startTime) / 1e9).toFixed(2);
    console.log(`--- PASS: ${testName} (${duration}s)`);
    console.log("PASS");
    process.exit(0);
  } catch (error) {
    const endTime = process.hrtime.bigint();
    const duration = (Number(endTime - startTime) / 1e9).toFixed(2);
    console.error(`\n--- FAIL: ${testName} (${duration}s)`);
    console.error(`    quickstart_test.js:--: ${error.message}`);
    console.log("FAIL");
    process.exit(1);
  }
}

runTests();
