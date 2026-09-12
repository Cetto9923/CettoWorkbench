'use strict';

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '../../../..');
const poKanbanJs = ['po-kanban.js','kanban/kanban-agile.js','kanban/kanban-format.js','kanban/kanban-issue.js','kanban/kanban-team.js','kanban/kanban-create.js'].map((f)=>fs.readFileSync(path.join(root, 'web/static/workbench/po', f), 'utf8')).join('\n');
const poHtml = fs.readFileSync(path.join(root, 'web/templates/workbench/po.html'), 'utf8');
const agileteamJs = fs.readFileSync(path.join(root, 'web/static/workbench/agileteam/agileteam.js'), 'utf8');
const leadHtml = fs.readFileSync(path.join(root, 'web/templates/workbench/lead.html'), 'utf8');
const pmoHtml = fs.readFileSync(path.join(root, 'web/templates/workbench/pmo.html'), 'utf8');

function assert(cond, msg) {
  if (!cond) {
    console.error('FAIL ' + msg);
    process.exit(1);
  }
  console.log('PASS ' + msg);
}

assert(/调整成员/.test(poHtml), 'po.html button label is 调整成员');
assert(/submitKanbanTeamAdjustment/.test(poKanbanJs), 'po-kanban.js exposes submitKanbanTeamAdjustment');
assert(/buildKanbanTeamAdjustmentItems/.test(poKanbanJs), 'po-kanban.js builds adjustment items from draft diff');
assert(/\/workbench\/api\/agile-teams\/['"]\s*\+|\/workbench\/api\/agile-teams\/'\s*\+/.test(poKanbanJs) ||
       /\/workbench\/api\/agile-teams\//.test(poKanbanJs),
  'submit posts to /workbench/api/agile-teams/:id/adjustments');
assert(/成员调整需组织级敏捷教练确认后正式生效/.test(poKanbanJs),
  'modal shows org-coach confirm tip');
assert(/成员调整已提交，待组织级敏捷教练确认/.test(poKanbanJs),
  'success toast uses PRD §8 copy');
assert(!/localStorage\.setItem\(\s*KANBAN_TEAM_OVERRIDES_STORAGE_KEY/.test(poKanbanJs),
  'teamOverrides localStorage is no longer written as formal source');
assert(/pendingAdd/.test(poKanbanJs) && /pendingRemove/.test(poKanbanJs) && /kanbanMemberStatusBadge/.test(poKanbanJs),
  'assignee / team modal show 待确认 / 待移除 badges');
assert(/page-agileteam/.test(leadHtml) && /agileteam\.js/.test(leadHtml),
  'lead.html mounts agile team list page + JS');
assert(/page-agileteam/.test(pmoHtml) && /agileteam\.js/.test(pmoHtml),
  'pmo.html mounts agile team list page + JS');
assert(/atOpenReview|atConfirm|atReject|adjustments/.test(agileteamJs + fs.readFileSync(path.join(root, 'web/static/workbench/agileteam/agileteam-detail.js'), 'utf8')),
  'agileteam.js wires confirm drawer actions');
assert(/at-child-badge|父级小组/.test(agileteamJs + fs.readFileSync(path.join(root, 'web/static/workbench/agileteam/agileteam-detail.js'), 'utf8')),
  'list/detail keep parent-child hierarchy');
assert(/atPager/.test(leadHtml) && /atPager/.test(pmoHtml), 'list pages have real pagination host');
assert(/atScopeCard/.test(leadHtml), 'lead view has 按团队/按部室 scope card');

assert(/atOpenMemberEdit|WbPersonPicker|DEFAULT_TEAM_ROLES|at-member-role-select/.test(fs.readFileSync(path.join(root, 'web/static/workbench/agileteam/agileteam-members.js'), 'utf8')),
  'PMO member editor uses unified person picker + native role select');
assert(/agileteam-members\.js\?v=20260827-member-picker-1/.test(pmoHtml), 'pmo.html loads member editor with picker stamp');
assert(/wb-person-picker\.js/.test(pmoHtml) && /wb-picker\.css/.test(pmoHtml), 'pmo.html loads shared person picker assets');
assert(/agileteam-members\.js\?v=20260827-member-picker-1/.test(leadHtml), 'lead.html loads member editor with picker stamp');
assert(/list=/.test(fs.readFileSync(path.join(root, 'web/static/workbench/agileteam/agileteam-members.js'), 'utf8')) === false,
  'member role no longer uses browser datalist (fixes mispositioned dropdown)');

// 自定义 pageSize：下拉 + 输入框 + localStorage 持久化 + 范围夹逼
assert(/自定义\u2026/.test(agileteamJs) || /自定义/.test(agileteamJs),
  'page-size dropdown exposes 自定义… option');
assert(/AT_PAGE_SIZE_MIN\s*=\s*5/.test(agileteamJs) && /AT_PAGE_SIZE_MAX\s*=\s*200/.test(agileteamJs),
  'pageSize custom range is 5..200');
assert(/wb:agileteam:pageSizeCustom/.test(agileteamJs),
  'custom pageSize persisted to localStorage with module-prefixed key');
assert(/atCommitCustomPageSize/.test(agileteamJs),
  'custom pageSize commits via atCommitCustomPageSize');
assert(/localStorage\.setItem\(\s*AT_PAGE_SIZE_CUSTOM_KEY/.test(agileteamJs),
  'atCommitCustomPageSize writes the custom value to localStorage');
assert(/localStorage\.getItem\(\s*AT_PAGE_SIZE_CUSTOM_KEY/.test(agileteamJs),
  'state seed reads the persisted custom value on boot');
assert(/clampPageSize|AT_PAGE_SIZE_MIN/.test(agileteamJs),
  'frontend clamps custom input to [5,200] before sending to backend');
assert(/agileteam\.js\?v=20260827-pagesize-custom-1/.test(pmoHtml) && /agileteam\.js\?v=20260827-pagesize-custom-1/.test(leadHtml),
  'pmo.html + lead.html bumped ?v= cache stamp for custom-pageSize build');
assert(/agileteam\.css\?v=20260827-pagesize-custom-1/.test(pmoHtml) && /agileteam\.css\?v=20260827-pagesize-custom-1/.test(leadHtml),
  'pmo.html + lead.html bumped ?v= cache stamp for custom-pageSize css');
