import http from 'k6/http';
import { check, group } from 'k6'
import { Trend } from 'k6/metrics';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

export const options = {
    thresholds: {
        http_req_failed: ['rate<0.01'], // http errors should be less than 1%
        http_req_duration: ['p(95)<250'], // 95% of requests should be below 250ms
    },
    scenarios: {
        get_scenario: {
            executor: 'constant-vus',

            gracefulStop: '30s',
            env: { REQUEST_TYPE: 'get' },

            vus: 100,
            duration: '60s',
        },
        post_scenario: {
            executor: 'constant-vus',

            gracefulStop: '30s',
            env: { REQUEST_TYPE: 'post' },

            vus: 100,
            duration: '60s',
        },
    },
};

let getRequestTrend = new Trend('get_request_duration', true);
let postRequestTrend = new Trend('post_request_duration', true);

let DOMAIN = __ENV.DOMAIN

export default function run() {
    const params = {
        headers: {
            "x-api-usage":"PRO",
        }
    };

    if (__ENV.REQUEST_TYPE === "get") {
        group('get-request', function () {
            get(params);
        })
    } else {
        group('post-request', function () {
            post(params);
        })
    }
}

export function get(params) {


    let resp = http.get(`https://hello.${DOMAIN}/headers`, params);
    getRequestTrend.add(resp.timings.duration);
    if (!check(resp, {
        'is status 200': (r) => r.status === 200,
    })) {
        console.log(`status: ${resp.status}`);
        console.log(JSON.stringify(resp));
    }
}

export function post(params) {
    const payload = JSON.stringify({
        'x': 'y'
    });

    let resp = http.post(`https://hello.${DOMAIN}/post`, payload, params);
    postRequestTrend.add(resp.timings.duration);
    if (!check(resp, {
        'is status 200': (r) => r.status === 200,
    })) {
        console.log(`status: ${resp.status}`);
        console.log(JSON.stringify(resp));
    }
}

export function handleSummary(data) {
    return {
        "summary.html": htmlReport(data),
    };
}
