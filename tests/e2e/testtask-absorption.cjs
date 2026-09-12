// Isolated browser regression: real form, scripts and events; synthetic API only.
// Run: NODE_PATH=<runtime node_modules> node tests/e2e/testtask-absorption.cjs
const { chromium } = require('playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const repo = path.resolve(__dirname, '../..');
(async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    for (const scenario of ['new-independent', 'existing-joint', 'partial', 'network']) {
      const page = await browser.newPage();
      const errors = [];
      page.on('pageerror', e => { errors.push(e.message); console.error('PAGE ERROR',e.message); });
      const template = fs.readFileSync(path.join(repo, 'web/templates/components/po_testtask_modal.html'), 'utf8');
      const form = template.split('{{ define "po/testtask" }}')[0].replace(/{{[\s\S]*?}}/g, '');
      await page.setContent('<div id="poTesttaskModal">' + form + '</div>');
      for (const css of ['layout/variables.css', 'layout/base.css', 'components/form.css', 'components/autocomplete.css', 'components/button-system.css', 'po/testtask.css']) {
        await page.addStyleTag({ path: path.join(repo, 'web/static/css', css) });
      }
      await page.addScriptTag({ path: path.join(repo, 'web/static/vendor/jquery/jquery.min.js') });
      await page.evaluate(scenario => {
        window.calls = [];
        window.openShowModals = () => {};
        window.closeShowModals = () => {}; window.showToast = (m) => console.log('toast',m);
        window.appFetch = async (url, options = {}) => {
          const body = options.body ? JSON.parse(options.body) : null;
          window.calls.push({ url, body });
          let data;
          if (url.endsWith('/testtask')) data = { demandId: 123, title: '测试需求', qd: 'qa', qdName: '测试员', estimateLaunch: '2026-12-30', users: [{ account: 'qa', realname: '测试员' }], systems: [{ id: 1, name: '系统A', isMain: true, stories: [{ id: 11, title: '研发需求A' }] }, { id: 2, name: '系统B', isMain: false, stories: [] }] };
          else if (url.endsWith('/executions')) data = [{ value: '10-20', label: '项目/执行', projectId: 10, executionId: 20 }];
          else if (options.method === 'POST' && url.endsWith('/builds')) data = { builds: [{ productId: body.builds[0].productId, buildId: 101, name: '新版本' }] };
          else if (url.endsWith('/builds')) data = [{ value: '101', label: '版本A' }, { value: '102', label: '版本B' }];
          else if (url.endsWith('/tasks')) {
            if (scenario === 'network') throw new Error('connection lost');
            if (scenario === 'partial') return new Response(JSON.stringify({ success: false, message: '结果未知', succeeded: [{ testtaskId: 901 }] }), { status: 207 });
            data = { tasks: [{ testtaskId: 901 }] };
          } else data = [];
          return new Response(JSON.stringify({ success: true, data }), { status: 200 });
        };
      }, scenario);
      for (const file of ['components/autocomplete-options.js', 'ui.js', 'po/testtask-fields.js', 'po/testtask-render.js', 'po/testtask.js', 'po/testtask-submit.js', 'po/testtask-builds.js']) {
        await page.addScriptTag({ path: path.join(repo, 'web/static/js', file) });
      }
      await page.evaluate(() => window.openPoSubmitTestModal({ id: 123 }));
      await page.waitForFunction(() => document.querySelector('[data-tt-owner-account="1"]')?.value === 'qa', null, {timeout:5000}).catch(async e => { console.error(await page.evaluate(() => ({calls:window.calls, html:document.body.innerHTML.slice(-2500)}))); throw e; });
      if (scenario === 'existing-joint') await page.evaluate(() => window.jQuery('[name="isJointTest"][value="1"]').prop('checked', true).trigger('change'));
      async function next(step) {
        await page.evaluate(() => document.querySelector('#poTesttaskNextBtn').click());
        await page.waitForFunction(step => !document.querySelector('[data-tt-panel="' + step + '"]').classList.contains('po-testtask-hidden'), step, { timeout: 5000 });
      }
      await next(2);
      await page.waitForFunction(() => document.querySelector('[data-tt-exist-ver="1"] option[value="101"]'));
      if (scenario === 'existing-joint') {
        await page.evaluate(() => {
          for (const unit of ['1', '2']) {
            window.jQuery('[name="ver' + unit + 'Mode"][value="exist"]').prop('checked', true).trigger('change');
            window.jQuery('[data-tt-exist-ver="' + unit + '"]').val(unit === '1' ? ['101', '102'] : ['101']);
          }
        });
      } else await page.selectOption('[data-tt-exec="1"]', '10-20');
      await next(3);
      await next(4);
      const ok = await page.evaluate(() => window.PoTesttaskSubmit.submit());
      assert.equal(ok, scenario !== 'partial' && scenario !== 'network', scenario + ' submission');
      const calls = await page.evaluate(() => window.calls);
      const submission = calls.find(c => c.url.endsWith('/tasks'));
      assert.ok(submission, 'must create a testtask, not only a build');
      if (scenario === 'existing-joint') {
        assert.equal(submission.body.joint, 1);
        assert.deepEqual(submission.body.builds, [[101, 102], [101]]);
        assert.equal(submission.body.owner, 'qa');
        assert.equal(calls.filter(c => c.body && c.url.endsWith('/builds')).length, 0);
      } else assert.equal(submission.body.tasks[0].owner, 'qa');
      assert.ok(await page.locator('#poTesttaskResult').textContent().then(s => s.includes(scenario === 'network' ? '结果未知' : '#901')));
      const before = calls.length;
      await page.evaluate(() => window.PoTesttaskSubmit.submit());
      assert.equal(await page.evaluate(() => window.calls.length), before, 'must not resubmit after success/partial');
      assert.deepEqual(errors, []);
      await page.evaluate(() => window.PoTesttaskSubmit.reset());
      await page.evaluate(() => window.PoTesttaskSubmit.submit());
      assert.equal(await page.evaluate(() => window.calls.length), before, 'reopening must retain the known outcome');
      if (scenario === 'existing-joint' && process.env.WB_TESTTASK_SCREENSHOTS) {
        for (const theme of ['light', 'dark']) {
          await page.evaluate(theme => document.documentElement.dataset.theme = theme, theme);
          await page.locator('#poTesttaskFormRoot').screenshot({ path: path.join(process.env.WB_TESTTASK_SCREENSHOTS, 'testtask-' + theme + '.png') });
        }
      }
      console.log('PASS', scenario);
      await page.close();
    }
  } finally { await browser.close(); }
})().catch(err => { console.error(err); process.exitCode = 1; });
