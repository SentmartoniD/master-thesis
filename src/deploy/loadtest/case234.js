import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';

// Cases 2 and 3. One script for both so the horizontal and vertical curves are
// measured on identical rates and can be overlaid directly.
//   Case 2 (horizontal): node fixed at D2s_v5, replicas vary  - H4 = 2, H8 = 4, H16 = 8
//   Case 3 (vertical):   replicas fixed at 2, node varies     - V4 = D2s_v5, V8 = D4s_v5, V16 = D8s_v5
// With DB_MAX_CONNS=2 the connection budget is replicas x 2.
const RATES = [4000, 8000, 14000, 22000];

// .replace strips the UTF-8 BOM that PowerShell writes at the start of files.
const codes = new SharedArray('codes', () =>
  JSON.parse(open('./codes.json').replace(/^\uFEFF/, ''))
);

const REDIRECT = __ENV.BASE_REDIRECT || 'http://redirect-service:8000';

// 60s warm-up, then per rate a 10s ramp and a 90s hold. The hold is 90s so a
// 75-second measurement window fits inside it with 15s of clearance after the
// ramp. Step i ends at t0 + 60 + i*100, so 160 / 260 / 360 / 460.
function steps(rates) {
  const out = [{ duration: '60s', target: rates[0] }];   // warm-up, discarded
  for (const r of rates) {
    out.push({ duration: '10s', target: r });
    out.push({ duration: '90s', target: r });
  }
  return out;
}

export const options = {
  scenarios: {
    redirect: {
      executor: 'ramping-arrival-rate',
      startRate: RATES[0],
      timeUnit: '1s',
      // Little's Law: VUs = rate x latency. 1600 covers 16000 rps at 100 ms,
      // well past the 200 ms SLO for any step that could still pass. maxVUs
      // carries the collapsed step, where dropped iterations are expected.
      preAllocatedVUs: 1600,
      maxVUs: 8000,
      stages: steps(RATES),
    },
  },
  thresholds: {
    // Runaway guard, not the SLO. Raised to 5s so a collapsing step still
    // completes its 90-second hold and its measurement window, instead of
    // aborting part-way and leaving the window empty.
    'http_req_duration': [
      { threshold: 'p(95)<5000', abortOnFail: true, delayAbortEval: '30s' },
    ],
  },
};

export default function () {
  const code = codes[Math.floor(Math.random() * codes.length)];
  const res = http.get(`${REDIRECT}/${code}`, {
    redirects: 0,
    // Without an explicit name, k6 tags each request with its full URL.
    // 500,000 distinct codes would create 500,000 time series.
    tags: { name: 'redirect' },
  });
  check(res, { 'is 302': (r) => r.status === 302 });
}
