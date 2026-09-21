import juice from 'juice';
import { shapes } from './shapes.mjs';
const out = {};
for (const [name, gen] of Object.entries(shapes)) {
  const doc = gen();
  let n = doc.length > 200000 ? 10 : doc.length > 20000 ? 40 : 300;
  for (let i = 0; i < Math.min(n, 20); i++) juice(doc, {});
  const t0 = process.hrtime.bigint();
  for (let i = 0; i < n; i++) juice(doc, {});
  const t1 = process.hrtime.bigint();
  out[name] = { ms: Number(t1 - t0) / 1e6 / n, bytes: doc.length };
}
console.log(JSON.stringify(out, null, 0));
