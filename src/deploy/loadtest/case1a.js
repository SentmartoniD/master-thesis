import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';

// load dataset
const codes = new SharedArray('codes', () =>
  JSON.parse(open('./codes.json').replace(/^\uFEFF/, ''))
);

const REDIRECT     = __ENV.BASE_REDIRECT     || 'http://redirect-service:8000';
const URLSHORTENER = __ENV.BASE_URLSHORTENER || 'http://urlshortener-service:8000';
const ANALYTICS    = __ENV.BASE_ANALYTICS    || 'http://analytics-service:8000';

function scenario(exec) {
  return {
    executor: 'ramping-arrival-rate',
    startRate: 100,
    timeUnit: '1s',
    preAllocatedVUs: 50,
    maxVUs: 300,
    exec,
    stages: [
      { duration: '60s', target: 100 },
      { duration: '90s', target: 100 },
      { duration: '5s',  target: 200 },
      { duration: '85s', target: 200 },
      { duration: '5s',  target: 300 },
      { duration: '85s', target: 300 },
    ],
  };
}

export const options = {
  scenarios: {
    redirect:     scenario('redirect'),
    urlshortener: scenario('urlshortener'),
    analytics:    scenario('analytics'),
  },
  thresholds: {
    'http_req_failed': ['rate<0.01'],
  },
};

export function redirect() {
  const code = codes[Math.floor(Math.random() * codes.length)];
  const res = http.get(`${REDIRECT}/${code}`, {
    redirects: 0,
    // Without an explicit name, k6 tags each request with its full URL.
    // 500,000 distinct codes would create 500,000 time series.
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
