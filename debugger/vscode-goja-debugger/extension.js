const vscode = require('vscode');
const { spawn } = require('child_process');
const net = require('net');

/** @type {import('child_process').ChildProcess | null} */
let harnessProcess = null;

function activate(context) {
	const factory = {
		async createDebugAdapterDescriptor(session) {
			const config = session.configuration;
			const port = config.port || 4711;
			const host = config.host || '127.0.0.1';

			if (config.request === 'launch') {
				await launchProgram(config, port);
				// No waitForPort — launchProgram already waits for "listening"
				// on stderr, which guarantees the port is bound.
			} else if (config.request === 'attach') {
				// For attach (including compound configs where another
				// debugger launches the Go program), wait for the DAP
				// server to start listening. The probe connection is safe
				// because ListenTCP discards probes that don't send data.
				await waitForPort(host, port, 30000);
			}

			return new vscode.DebugAdapterServer(port, host);
		}
	};

	context.subscriptions.push(
		vscode.debug.registerDebugAdapterDescriptorFactory('goja', factory)
	);

	context.subscriptions.push(
		vscode.debug.onDidTerminateDebugSession(session => {
			if (session.configuration.type === 'goja' && harnessProcess) {
				harnessProcess.kill();
				harnessProcess = null;
			}
		})
	);
}

/**
 * Launch the user's Go program that embeds goja and starts a DAP server.
 *
 * The launch config should look like:
 *   {
 *     "type": "goja",
 *     "request": "launch",
 *     "program": "/path/to/go/package",  // directory with main.go
 *     "args": ["--script", "foo.ts"],     // any args for the program
 *     "port": 4711,                       // DAP port (passed via --port)
 *     "cwd": "/optional/working/dir",     // defaults to program dir
 *     "env": { "KEY": "VAL" },            // extra env vars
 *     "buildArgs": ["-tags", "debug"]     // extra args for `go run`
 *   }
 *
 * The extension runs: go run [buildArgs...] . [--port PORT] [args...]
 * It waits for the program to print "listening" on stderr before connecting.
 */
async function launchProgram(config, port) {
	if (harnessProcess) {
		harnessProcess.kill();
		harnessProcess = null;
	}

	const program = config.program;
	if (!program) {
		throw new Error('goja launch config requires a "program" field pointing to a Go package directory');
	}

	// Build the command: go run [buildArgs...] . [--port PORT] [args...]
	const goRunArgs = ['run'];
	if (config.buildArgs) {
		goRunArgs.push(...config.buildArgs);
	}
	goRunArgs.push('.');
	goRunArgs.push('--port', String(port));
	if (config.args) {
		goRunArgs.push(...config.args);
	}

	const cwd = config.cwd || program;

	const output = vscode.window.createOutputChannel('Goja Debugger');
	output.show(true);
	output.appendLine(`> cd ${cwd} && go ${goRunArgs.join(' ')}`);

	return new Promise((resolve, reject) => {
		const proc = spawn('go', goRunArgs, {
			cwd,
			env: { ...process.env, ...config.env },
		});
		harnessProcess = proc;

		let resolved = false;

		proc.stderr.on('data', (data) => {
			const text = data.toString();
			output.append(text);
			if (!resolved && text.includes('listening')) {
				resolved = true;
				resolve();
			}
		});

		proc.stdout.on('data', (data) => {
			output.append(data.toString());
		});

		proc.on('error', (err) => {
			output.appendLine(`Process error: ${err.message}`);
			if (!resolved) {
				resolved = true;
				reject(err);
			}
		});

		proc.on('exit', (code) => {
			output.appendLine(`Process exited with code ${code}`);
			harnessProcess = null;
			if (!resolved) {
				resolved = true;
				if (code !== 0) {
					reject(new Error(`Process exited with code ${code}`));
				} else {
					resolve();
				}
			}
		});
	});
}

/**
 * Wait for a TCP port to accept connections.
 * The server's ListenTCP gracefully handles these probe connections
 * (they connect but send no data, so the server re-accepts).
 */
function waitForPort(host, port, timeoutMs) {
	return new Promise((resolve, reject) => {
		const deadline = Date.now() + timeoutMs;

		function tryConnect() {
			if (Date.now() > deadline) {
				reject(new Error(`Timeout waiting for ${host}:${port}`));
				return;
			}
			const sock = new net.Socket();
			sock.once('connect', () => {
				sock.destroy();
				resolve();
			});
			sock.once('error', () => {
				sock.destroy();
				setTimeout(tryConnect, 200);
			});
			sock.connect(port, host);
		}

		tryConnect();
	});
}

function deactivate() {
	if (harnessProcess) {
		harnessProcess.kill();
		harnessProcess = null;
	}
}

module.exports = { activate, deactivate };
