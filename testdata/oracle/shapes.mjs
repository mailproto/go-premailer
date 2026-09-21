// Document shapes for cross-implementation benchmarking. Mirrored in shapes.go.
function rep(n, f) { let s = ""; for (let i = 0; i < n; i++) s += f(i); return s; }

const shapes = {
  // Smallest realistic unit: one rule, one element.
  tiny: () => `<style>div{color:red}</style><div>x</div>`,

  // Short transactional email.
  small: () => `<style>${rep(12, i => `.c${i}{color:#${(i*111111).toString(16).padStart(6,'0')};padding:${i}px}\n`)}` +
    `a{color:#06c}td{vertical-align:top}</style><table><tbody>` +
    rep(12, i => `<tr><td class="c${i}"><a href="#">l${i}</a>text ${i}</td></tr>`) + `</tbody></table>`,

  // Many rules, few nodes: index construction cannot amortize.
  manyRulesFewNodes: () => `<style>${rep(800, i => `.c${i} span{color:#${(i*7).toString(16).padStart(6,'0')}}\n`)}</style>` +
    `<div class="c1"><span>x</span></div>`,

  // Few rules, many nodes: the walk dominates.
  fewRulesManyNodes: () => `<style>td{padding:2px}.a{color:red}</style><table><tbody>` +
    rep(3000, i => `<tr><td class="a">${i}</td></tr>`) + `</tbody></table>`,

  // Every selector is universal or attribute-keyed, so nothing buckets and
  // the index degenerates to brute force plus merge overhead.
  unbucketable: () => `<style>${rep(120, i => `*[data-k="${i}"]{color:#${(i*9).toString(16).padStart(6,'0')}}\n`)}</style>` +
    `<div>` + rep(400, i => `<p data-k="${i%120}">x</p>`) + `</div>`,

  // Deep descendant chains: many partial matches per node.
  deepDescendant: () => `<style>${rep(60, i => `div div div .d${i} span{color:#${(i*13).toString(16).padStart(6,'0')}}\n`)}</style>` +
    `<div><div><div>` + rep(300, i => `<p class="d${i%60}"><span>x</span></p>`) + `</div></div></div>`,

  // Large newsletter, where juice's O(rules x nodes) should hurt most.
  large: () => `<style>${rep(200, i => `.c${i}{color:#${(i*7919%0xffffff).toString(16).padStart(6,'0')};font-size:${10+i%12}px}\n`)}` +
    `table{border-collapse:collapse}td{padding:4px}</style><table><tbody>` +
    rep(4000, i => `<tr id="r${i}"><td class="c${i%200}"><p>Row ${i} body copy here</p></td></tr>`) + `</tbody></table>`,
};
export { shapes };
