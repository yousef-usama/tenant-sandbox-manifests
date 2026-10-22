import { spawn } from 'child_process';
import fs from 'fs';
import path from 'path';

const CHROME_PATH = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const DEBUG_PORT = 9222;
const USER_DATA_DIR = '/tmp/chrome-ss-profile-' + Date.now();

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// Simple CDP client over native WebSocket
class CDPClient {
  constructor(wsUrl) {
    this.wsUrl = wsUrl;
    this.id = 1;
    this.callbacks = new Map();
  }

  async connect() {
    this.ws = new WebSocket(this.wsUrl);
    await new Promise((resolve, reject) => {
      this.ws.onopen = resolve;
      this.ws.onerror = reject;
    });
    this.ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      if (msg.id && this.callbacks.has(msg.id)) {
        const { resolve, reject } = this.callbacks.get(msg.id);
        this.callbacks.delete(msg.id);
        if (msg.error) reject(msg.error);
        else resolve(msg.result);
      }
    };
  }

  send(method, params = {}) {
    return new Promise((resolve, reject) => {
      const msgId = this.id++;
      this.callbacks.set(msgId, { resolve, reject });
      this.ws.send(JSON.stringify({ id: msgId, method, params }));
    });
  }

  close() {
    if (this.ws) this.ws.close();
  }
}

async function getDebuggerUrl() {
  for (let i = 0; i < 30; i++) {
    try {
      const res = await fetch(`http://127.0.0.1:${DEBUG_PORT}/json/version`);
      if (res.ok) {
        const data = await res.json();
        return data.webSocketDebuggerUrl;
      }
    } catch (_) {}
    await sleep(300);
  }
  throw new Error('Chrome remote debugging did not become ready');
}

async function run() {
  console.log('Starting headless Chrome...');
  const chromeProcess = spawn(CHROME_PATH, [
    '--headless',
    '--disable-gpu',
    `--remote-debugging-port=${DEBUG_PORT}`,
    '--ignore-certificate-errors',
    '--window-size=1440,900',
    '--no-first-run',
    `--user-data-dir=${USER_DATA_DIR}`
  ], { stdio: 'ignore' });

  try {
    const wsUrl = await getDebuggerUrl();
    console.log('Connected to CDP at:', wsUrl);

    // Get argo token
    let argoToken = '';
    try {
      const res = await fetch('https://localhost:8443/api/v1/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: 'admin', password: '28nEyTuo1pEM4o2a' })
      });
      const data = await res.json();
      argoToken = data.token;
      console.log('Got ArgoCD token successfully');
    } catch (e) {
      console.error('Failed to get ArgoCD token:', e.message);
    }

    // Helper to create page target, set cookies, navigate, screenshot
    async function capturePage({ url, cookies = [], waitMs = 3000, outputPath, basicAuth = null }) {
      console.log(`Capturing ${url} -> ${outputPath}...`);
      // Create target
      const newTargetRes = await fetch(`http://127.0.0.1:${DEBUG_PORT}/json/new?${encodeURIComponent(url)}`, { method: 'PUT' });
      const target = await newTargetRes.json();
      const client = new CDPClient(target.webSocketDebuggerUrl);
      await client.connect();

      await client.send('Page.enable');
      await client.send('Network.enable');

      if (cookies.length > 0) {
        await client.send('Network.setCookies', { cookies });
      }

      if (basicAuth) {
        await client.send('Network.setExtraHTTPHeaders', {
          headers: {
            'Authorization': 'Basic ' + Buffer.from(`${basicAuth.user}:${basicAuth.pass}`).toString('base64')
          }
        });
      }

      await client.send('Page.navigate', { url });
      await sleep(waitMs);

      const ss = await client.send('Page.captureScreenshot', { format: 'png' });
      fs.writeFileSync(outputPath, Buffer.from(ss.data, 'base64'));
      console.log(`Saved screenshot: ${outputPath} (${ss.data.length} bytes)`);

      client.close();
      await fetch(`http://127.0.0.1:${DEBUG_PORT}/json/close/${target.id}`);
    }

    const outDir = '/Users/Yousef/Desktop/tenant-sandbox-manifests/docs/screenshots';
    fs.mkdirSync(outDir, { recursive: true });

    // 1. ArgoCD UI
    if (argoToken) {
      await capturePage({
        url: 'https://localhost:8443/applications',
        cookies: [{
          name: 'argocd.token',
          value: argoToken,
          domain: 'localhost',
          path: '/',
          secure: false,
          httpOnly: false
        }],
        waitMs: 6000,
        outputPath: path.join(outDir, '03-argocd-applications.png')
      });
    }

    console.log('ArgoCD capture completed successfully!');
  } finally {
    chromeProcess.kill();
    await sleep(500);
    try {
      fs.rmSync(USER_DATA_DIR, { recursive: true, force: true, maxRetries: 3 });
    } catch (_) {}
  }
}

run().catch(err => {
  console.error('Capture failed:', err);
  process.exit(1);
});
