import http from 'k6/http';
import { check, group } from 'k6'
import { Trend } from 'k6/metrics';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

export const options = {
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
    if (__ENV.REQUEST_TYPE === "get") {
        group('get-request', function () {
            get();
        })
    } else {
        group('post-request', function () {
            post();
        })
    }
}

export function get() {
    let resp = http.get(`https://hello.${DOMAIN}/headers`);
    getRequestTrend.add(resp.timings.duration);
    if (!check(resp, {
        'is status 200': (r) => r.status === 200,
    })) {
        console.log(`status: ${resp.status}`);
        console.log(JSON.stringify(resp));
    }
}

export function post() {
    const payload = JSON.stringify({
        'x': 'y'
    });

    let resp = http.post(`https://hello.${DOMAIN}/post`, payload);
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