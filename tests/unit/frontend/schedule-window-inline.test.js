const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
const path = require("node:path");

async function run() {
  let draft;
  let success = true;
  let saved = 0;
  const requests = [];
  const $ = () => ({ text() { return this; }, on() { return this; } });
  const window = {
    location: { href: "/home" },
    showToast() {}, openShowModals() {}, closeShowModals() {},
    ScheduleWindowDraft: {
      createDraft: () => ({ online: "2026-10-01", name: "窗口", start: "2026-09-20", teamgroupId: "7", productIds: [] }),
      setDraft: value => { draft = value; }, getDraft: () => draft,
      resetDraft() {}, fillForm() {}
    },
    scheduleFetch: async (url, options) => {
      requests.push({ url, options });
      return { ok: true, json: async () => ({ success, redirectUrl: "/schedule" }) };
    }
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, "../../../web/static/js/schedule/schedulewindow.js"), "utf8"), {
    window, jQuery: $, document: { getElementById: () => null }
  });
  const flush = () => new Promise(resolve => setImmediate(resolve));
  window.openScheduleCreateVersionWindowModal(payload => { saved++; assert.equal(payload.teamgroupId, "7"); });
  window.saveScheduleVersionWindowModal();
  await flush();
  assert.equal(saved, 1);
  assert.equal(window.location.href, "/home", "Inline create must retain the home page");
  assert.equal(requests[0].url, "/schedule/windows");
  assert.equal(requests[0].options.method, "POST");
  success = false;
  window.openScheduleCreateVersionWindowModal(() => saved++);
  window.saveScheduleVersionWindowModal();
  await flush();
  assert.equal(saved, 1, "Failure must not refresh or imply a saved window");
  success = true;
  window.closeScheduleVersionWindowModal();
  window.openScheduleCreateVersionWindowModal();
  window.saveScheduleVersionWindowModal();
  await flush();
  assert.equal(window.location.href, "/schedule", "Standalone create retains its existing redirect");
  console.log("PASS: inline window save, failed save and standalone create");
}
run().catch(error => { console.error(error); process.exitCode = 1; });
