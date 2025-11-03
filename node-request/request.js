#!/usr/bin/env node

import { parseArgs } from 'node:util';
import { hrtime } from 'node:process';

/**
 * Parses and validates command-line arguments.
 */
function getArgs() {
    const options = {
        url: { type: 'string', short: 'u', help: 'The URL to request' },
        concurrency: { type: 'string', short: 'c', default: '10', help: 'Number of parallel requests' },
        timeout: { type: 'string', short: 't', default: '5000', help: 'Request timeout in milliseconds' },
        keepalive: { type: 'boolean', short: 'k', default: false, help: 'Use HTTP Keep-Alive' },
    };

    try {
        const { values } = parseArgs({ options, allowPositionals: false });

        if (values.help) {
            console.log('Parallel HTTP Request Client');
            console.log('Usage: node parallel-client.js --url <URL> [OPTIONS]\n');
            console.log('Options:');
            for (const [key, { help, short, default: def }] of Object.entries(options)) {
                let line = `  --${key}`;
                if (short) line += `, -${short}`;
                line += `\t${help}`;
                if (def) line += ` (default: ${def})`;
                console.log(line);
            }
            process.exit(0);
        }

        if (!values.url) {
            throw new Error('--url is a required argument.');
        }

        // Ensure URL is valid
        try {
            new URL(values.url);
        } catch (e) {
            throw new Error(`Invalid URL provided: ${values.url}`);
        }

        const config = {
            url: values.url,
            concurrency: parseInt(values.concurrency, 10),
            timeout: parseInt(values.timeout, 10),
            keepAlive: values.keepalive,
        };

        if (isNaN(config.concurrency) || config.concurrency < 1) {
            throw new Error('--concurrency must be a number greater than 0.');
        }
        if (isNaN(config.timeout) || config.timeout < 1) {
            throw new Error('--timeout must be a number greater than 0.');
        }

        return config;
    } catch (err) {
        console.error(`Error: ${err.message}`);
        console.log('Run with --help for usage information.');
        process.exit(1);
    }
}

/**
 * Performs a single fetch request and returns a structured result.
 * @param {number} id - The unique ID for this request.
 * @param {string} url - The URL to fetch.
 * @param {object} options - Configuration options.
 * @param {number} options.timeout - Timeout in ms.
 * @param {boolean} options.keepAlive - Keep-Alive flag.
 * @returns {Promise<object>} - A result object.
 */
async function singleRequest(id, url, { timeout, keepAlive }) {
    const startTime = hrtime.bigint();

    // Each request needs its own AbortController and signal
    const fetchOptions = {
        signal: AbortController.timeout(timeout),
        keepalive: keepAlive,
    };

    try {
        const response = await fetch(url, fetchOptions);

        // We must consume the body to fully complete the request
        // and allow the connection to be reused (if keepAlive=true)
        await response.text();

        const endTime = hrtime.bigint();
        const duration = (endTime - startTime) / 1_000_000n; // Nanoseconds to milliseconds

        if (!response.ok) {
            return { status: 'failed', id, reason: `HTTP ${response.status}`, duration };
        }
        return { status: 'success', id, httpStatus: response.status, duration };

    } catch (error) {
        const endTime = hrtime.bigint();
        const duration = (endTime - startTime) / 1_000_000n;

        let reason = error.message;
        if (error.name === 'AbortError') {
            reason = 'Timeout';
        }
        return { status: 'failed', id, reason, duration };
    }
}

/**
 * Main function to run the client.
 */
async function main() {
    const config = getArgs();

    while (true) {
        // Create an array of promises, each representing one request
        const requests = [];
        for (let i = 0; i < config.concurrency; i++) {
            requests.push(singleRequest(i + 1, config.url, config));
        }

        // Wait for all requests to either succeed or fail
        const totalStartTime = hrtime.bigint();
        await Promise.allSettled(requests);
        const totalEndTime = hrtime.bigint();

        const totalDuration = (totalEndTime - totalStartTime) / 1_000_000n;
        console.log(`---------------- Total Time: ${totalDuration}ms ----------------`);
    }
}

// Run the main function
main();
