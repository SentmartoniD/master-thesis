import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';

// load dataset
const codes = new SharedArray('codes', () =>
  JSON.parse(open('./codes.json').replace(/^\uFEFF/, ''))
);

// In-cluster service DNS. Overridable for local testing.
const REDIRECT     = __ENV.BASE_REDIRECT     || 'http://redirect-service:8000';
const URLSHORTENER = __ENV.BASE_URLSHORTENER || 'http://urlshortener-service:8000';
const ANALYTICS    = __ENV.BASE_ANALYTICS    || 'http://analytics-service:8000';

const MIX = { redirect: 0.990, urlshortener: 0.009, analytics: 0.001 };

const TOTALS = (__ENV.TOTALS || '1000,2000,3000').split(',').map(Number);

function scenario(exec, share) {
  // Math.max(1, ...) keeps analytics at a whole request per second on the
  // lowest step. Below 1 rps k6 would round the arrival rate down to zero.
  const r = TOTALS.map((t) => Math.max(1, Math.round(t * share)));

  return {
    executor: 'ramping-arrival-rate',
    startRate: r[0],
    timeUnit: '1s',
    preAllocatedVUs: 50,
    maxVUs: 2000,
    exec,
    stages: [
      { duration: '60s', target: r[0] },
      { duration: '90s', target: r[0] },
      { duration: '5s',  target: r[1] },
      { duration: '85s', target: r[1] },
      { duration: '5s',  target: r[2] },
      { duration: '85s', target: r[2] }, 
    ],
  };
}

export const options = {
  scenarios: {
    redirect:     scenario('redirect',     MIX.redirect),
    urlshortener: scenario('urlshortener', MIX.urlshortener),
    analytics:    scenario('analytics',    MIX.analytics),
  },
  thresholds: {
    'http_req_failed': ['rate<0.01'],
  },
};

export function redirect() {
  const code = codes[Math.floor(Math.random() * codes.length)];
  const res = http.get(`${REDIRECT}/${code}`, {
    redirects: 0,
    tags: { name: 'redirect' },
  });
  check(res, { 'redirect is 302': (r) => r.status === 302 });
}

export function urlshortener() {
  const res = http.post(
    `${URLSHORTENER}/api/v1/links`,
    JSON.stringify({ url: 'https://example.com/page' }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'create' },
    }
  );
  check(res, { 'create is 201': (r) => r.status === 201 });
}

export function analytics() {
  const code = codes[Math.floor(Math.random() * codes.length)];
  const res = http.get(`${ANALYTICS}/api/v1/analytics/${code}`, {
    tags: { name: 'analytics' },
  });
  check(res, { 'analytics is 200': (r) => r.status === 200 });
}
