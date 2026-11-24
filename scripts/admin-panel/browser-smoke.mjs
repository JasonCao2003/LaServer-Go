import { spawn } from 'node:child_process';
import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';

const { origin, password, user, profile } = JSON.parse(readFileSync(0, 'utf8'));
const chrome = spawn(process.env.CHROME_BINARY || 'google-chrome',
  ['--headless', '--remote-debugging-pipe', '--disable-background-networking',
   '--no-first-run', '--no-default-browser-check', `--user-data-dir=${profile}`],
  {stdio: ['ignore', 'ignore', 'ignore', 'pipe', 'pipe']});

let nextID = 0;
let pending = '';
const callbacks = new Map();
chrome.stdio[4].setEncoding('utf8');
chrome.stdio[4].on('data', chunk => {
  pending += chunk;
  for (;;) {
    const end = pending.indexOf('\0');
    if (end < 0) break;
    const reply = JSON.parse(pending.slice(0, end));
    pending = pending.slice(end + 1);
    callbacks.get(reply.id)?.(reply);
  }
});
const send = (method, params = {}, sessionId) => new Promise((resolve, reject) => {
  const id = ++nextID;
  const timer = setTimeout(() => { callbacks.delete(id); reject(Error(method + ' timed out')); }, 5000);
  callbacks.set(id, reply => {
    clearTimeout(timer);
    callbacks.delete(id);
    if (reply.error) reject(Error(method + ': ' + reply.error.message));
    else resolve(reply.result);
  });
  chrome.stdio[3].write(JSON.stringify({id, method, params, sessionId}) + '\0');
});

try {
  const { targetId } = await send('Target.createTarget', {url: 'about:blank'});
  const { sessionId } = await send('Target.attachToTarget', {targetId, flatten: true});
  const page = (method, params) => send(method, params, sessionId);
  const evaluate = async expression => {
    const reply = await page('Runtime.evaluate', {expression, returnByValue: true});
    assert(!reply.exceptionDetails, 'browser evaluation failed');
    return reply.result.value;
  };
  const waitFor = async expression => {
    for (let i = 0; i < 80; i++) {
      if (await evaluate(expression)) return;
      await new Promise(resolve => setTimeout(resolve, 40));
    }
    throw Error('page condition not reached: ' + expression);
  };
  const key = async (key, windowsVirtualKeyCode) => {
    for (const type of ['keyDown', 'keyUp']) {
      await page('Input.dispatchKeyEvent', {type, key, code:key, windowsVirtualKeyCode, ...(type === 'keyDown' && key === 'Enter' ? {text:'\r'} : {})});
    }
  };
  await page('Emulation.setDeviceMetricsOverride', {width: 360, height: 800, deviceScaleFactor: 1, mobile: true});
  await page('Page.navigate', {url: origin + '/login'});
  await waitFor('!!document.querySelector("#password")');
  await key('Tab', 9);
  assert.equal(await evaluate('document.activeElement.id'), 'password');
  await page('Input.insertText', {text: password});
  await key('Enter', 13);
  await waitFor('document.querySelector("h1")?.textContent === "服务概览"');
  assert(await evaluate('document.body.textContent.includes("在线") && document.body.textContent.includes("不可达")'));
  await evaluate('document.querySelector(\'a[href="/friends"]\').click()');
  await waitFor('document.querySelector("h1")?.textContent === "好友管理"');
  await evaluate(`document.querySelector('input[name="user"]').value = ${JSON.stringify(user)}; document.querySelector('form').requestSubmit()`);
  await waitFor('!!document.querySelector("tbody tr")');
  assert(await evaluate('document.documentElement.scrollWidth <= 360'), 'horizontal page overflow at 360px');
  assert.equal(await evaluate('document.querySelector(\'form[action="/friends/clear"] input[name="confirm"]\').validity.valueMissing'), true);
  await evaluate('const f = document.querySelector(\'form[action="/friends/clear"]\'); f.elements.confirm.value = "wrong-target"; f.requestSubmit()');
  await waitFor('document.body.textContent.includes("确认 ID 必须与目标用户一致")');
  await page('Page.navigate', {url: origin + '/friends?' + new URLSearchParams({user})});
  await waitFor('!!document.querySelector("tbody tr")');
  assert(await evaluate('document.querySelector("tbody").textContent.includes("active")'));
  console.log('Chrome: keyboard login, service status, friend lookup, clear validation and 360px layout passed.');
} finally {
  chrome.kill();
}
