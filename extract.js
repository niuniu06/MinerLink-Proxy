const fs = require('fs');
const readline = require('readline');

async function run() {
  const fileStream = fs.createReadStream('C:/Users/ba876/.gemini/antigravity/brain/5590e357-51a8-4eba-9a01-6dbf77078d8a/.system_generated/logs/transcript.jsonl');
  const rl = readline.createInterface({ input: fileStream, crlfDelay: Infinity });

  for await (const line of rl) {
    if (!line) continue;
    try {
      const step = JSON.parse(line);
      if (step.step_index >= 780) break;
      if (step.tool_calls && step.tool_calls.length > 0) {
        for (const call of step.tool_calls) {
          if (call.name === 'write_to_file' || call.name === 'default_api:write_to_file') {
            const target = call.args.TargetFile;
            if (target && target.includes('frontend') && target.includes('.vue')) {
              console.log('Restoring:', target, 'from step', step.step_index);
              fs.writeFileSync(target, call.args.CodeContent, 'utf8');
            }
            if (target && target.includes('frontend') && target.includes('style.css')) {
              console.log('Restoring:', target, 'from step', step.step_index);
              fs.writeFileSync(target, call.args.CodeContent, 'utf8');
            }
          }
        }
      }
    } catch(e) {}
  }
}
run();
