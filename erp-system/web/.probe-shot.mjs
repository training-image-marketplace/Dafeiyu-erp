import { chromium } from '@playwright/test';
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } });
const problems = [];
page.on('pageerror', (e) => problems.push('PAGEERROR ' + e.message));
page.on('response', (r) => { if (r.status() >= 500) problems.push('HTTP ' + r.status() + ' ' + r.url()); });
await page.goto('http://127.0.0.1:8899/', { waitUntil: 'networkidle' });
await page.fill('#login-username', 'admin');
await page.fill('#login-password', 'admin123');
await page.click('button[type=submit]');
await page.waitForTimeout(2500);
for (const p of ['/erp/switches', '/erp/android']) {
  problems.length = 0;
  await page.goto('http://127.0.0.1:8899' + p, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);
  const name = p.split('/').pop();
  await page.screenshot({ path: '/tmp/page-' + name + '.png', fullPage: true });
  console.log(p, problems.length ? 'PROBLEM: ' + problems.join(' | ') : 'ok');
}
console.log('apk head:', (await page.request.head('http://127.0.0.1:8899/apk/dafeiyu-erp.apk')).status());
console.log('apk readme:', (await page.request.get('http://127.0.0.1:8899/apk/README.txt')).status());
await browser.close();
