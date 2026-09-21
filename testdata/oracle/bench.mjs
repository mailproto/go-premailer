// Node-side counterpart to BenchmarkInline. TestGeneratorsAgree asserts this
// runs over byte-identical input to the Go benchmark.
import juice from 'juice';
import { genEmail } from './gen.mjs';

const doc = genEmail(60, 213);
const N = 50;
for (let i = 0; i < 10; i++) juice(doc, {});  // warm up
const t0 = process.hrtime.bigint();
for (let i = 0; i < N; i++) juice(doc, {});
const t1 = process.hrtime.bigint();
const ms = Number(t1 - t0) / 1e6 / N;
console.log(`juice: ${ms.toFixed(2)} ms/op  ${(doc.length / 1e6 / (ms / 1e3)).toFixed(2)} MB/s  (${doc.length} bytes)`);
