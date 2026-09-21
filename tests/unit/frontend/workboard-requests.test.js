const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

function element() {
  return { innerHTML: '', textContent: '', hidden: false, style: {}, dataset: {},
    classList: { toggle() {}, add() {}, remove() {}, contains() { return false; } },
    addEventListener() {}, appendChild() {}, querySelectorAll() { return []; },
    querySelector() { return element(); } };
}
const nodes = new Map(), requests = [], pagers = new Map(), events = {};
const document = {
  getElementById(id) { if (!nodes.has(id)) nodes.set(id, element()); return nodes.get(id); },
  querySelector() { return null; }, querySelectorAll() { return []; }, createElement: element,
  addEventListener(type, cb) { (events[type] ||= []).push(cb); }
};
let mode = 'task';
const WB = { renderDemandMatrix() {}, applyDemandFilters() {}, toggleFlag() {} };
const localStorage = {getItem: () => '1', setItem() {}};
const window = { PoWB: WB, escapeHtml: String, showToast() {},
  PersonalList: { priorityBadge: String, objectTypeBadge: String,
    renderPagination(opts) { pagers.set(opts.container, opts); } } };
const context = { window, document, localStorage, location: { pathname: '/board/task' }, console,
  URLSearchParams, setTimeout, requestAnimationFrame: cb => cb(),
  fetch(url) { let resolve, reject; const promise = new Promise((a,b) => {resolve=a;reject=b;});
    requests.push({url, resolve: body => resolve({ok:true,json:async()=>body}), reject}); return promise; }
};
vm.createContext(context);
for (const file of ['workboard-core.js','workboard.js']) {
  vm.runInContext(fs.readFileSync(path.join(__dirname,'../../../web/static/js/po',file),'utf8'),context);
}
const flush = () => new Promise(resolve => setImmediate(resolve));
const payload = group => ({success:true,selectedTeamgroupId:group,teamgroups:[{id:group,name:'g'+group}],owners:[],columns:[],summary:{filteredTotal:120,total:120,overdue:61}});
(async () => {
  assert.equal(requests.length,1);
  requests[0].resolve(payload(1)); await flush();
  requests[1].resolve({success:true,groupName:'g1',hasGroup:false}); await flush();
  const initial = requests.length;
  WB.onGroupPick(2,'g2');
  assert.equal(requests.length,initial+1,'group click should send only list before metrics');
  requests.at(-1).resolve(payload(2)); await flush();
  assert.equal(requests.length,initial+2,'exactly one metrics request per group load');
  assert.match(requests.at(-1).url,/metrics\?teamgroupId=2/);
  requests.at(-1).resolve({success:true,groupName:'g2',hasGroup:false}); await flush();

  WB.onGroupPick(3,'g3'); const older=requests.at(-1);
  WB.onGroupPick(4,'g4'); const newer=requests.at(-1);
  newer.resolve(payload(4)); await flush();
  const afterNewer=requests.length;
  older.resolve(payload(3)); await flush();
  assert.equal(WB.state.teamgroup,4,'late group response cannot overwrite current selection');
  assert.equal(requests.length,afterNewer,'stale response cannot issue metrics request');

  const pager=pagers.get(nodes.get('boardPagination'));
  assert.equal(pager.total,120,'pager must use server total rather than visible card count');
  pager.onPageChange(2);
  assert.equal(new URL(requests.at(-1).url,'http://test').searchParams.get('page'),'2');
  requests.at(-1).resolve(payload(4)); await flush();
  for (const handler of events.click) handler({target:{closest: selector => selector==='#taskStats [data-flag]' ? {dataset:{flag:'overdue'}} : null}});
  const filterURL=new URL(requests.at(-1).url,'http://test');
  assert.equal(filterURL.searchParams.get('focus'),'overdue','focus must filter in SQL, not just visible cards');
  assert.equal(filterURL.searchParams.get('page'),'1','changing focus resets page');

  WB.openTaskDrawer(10,'story',''); const drawerOld=requests.at(-1);
  WB.openTaskDrawer(11,'story',''); const drawerNew=requests.at(-1);
  drawerNew.resolve(payload(4)); await flush();
  const currentHTML=nodes.get('drawerTaskList').innerHTML;
  drawerOld.reject(new Error('old request failed')); await flush();
  assert.equal(nodes.get('drawerTaskList').innerHTML,currentHTML,'old drawer failure cannot erase new data');
  const drawerPager=pagers.get(nodes.get('drawerPagination'));
  drawerPager.onPageChange(2);
  assert.match(requests.at(-1).url,/storyId=11&pageSize=50&page=2/);
  console.log('PASS board requests: single metrics, stale response guard, server focus and list/drawer pagination');
})().catch(err => {console.error(err);process.exitCode=1;});
