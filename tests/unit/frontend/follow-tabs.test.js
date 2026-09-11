const assert = require("assert");
const fs = require("fs");
const path = require("path");

const root = path.join(__dirname, "../../..");
const template = fs.readFileSync(path.join(root, "web/templates/po/follow.html"), "utf8");
const script = fs.readFileSync(path.join(root, "web/static/js/po/follow.js"), "utf8");
const demandScript = fs.readFileSync(path.join(root, "web/static/js/po/follow-demand.js"), "utf8");

assert.match(template, /class="category-tab follow-tab active" data-tab="demand"/, "业务需求必须是默认视图");
assert.match(template, /data-tab="weekly"/, "项目周报必须保留为已接入视图");
assert.doesNotMatch(template, /data-tab="all"/, "不应保留无数据来源的全部视图");
assert.doesNotMatch(template, /data-tab="risk"|data-tab="testtask"/, "不应展示未接入对象类型");
assert.match(script, /var currentTab = "demand"/, "脚本默认状态必须与页面一致");
assert.match(script, /tab !== "demand" && tab !== "weekly"/, "脚本只接受已接入视图");
assert.doesNotMatch(script, /tabCountAll/, "脚本不能再维护已移除的全部计数");

// 业需 Tab 验收断言
assert.match(template, /data-scope="open"/, "工具栏必须有全部未关闭按钮");
assert.match(demandScript, /scope:\s*"open"/, "业务需求默认 scope 必须为 open");
assert.doesNotMatch(template, /data-scope="key"/, "DOM 不得再有重点关注按钮");
assert.doesNotMatch(template, /data-scope="key_open"/, "DOM 不得再有未关闭+重点按钮");
assert.doesNotMatch(template, /data-scope="open_clean"/, "DOM 不得再有未关闭·正常推进按钮");
assert.doesNotMatch(template, />重点关注</, "DOM 不得再有重点关注一级文本");
assert.doesNotMatch(template, />未关闭\+重点</, "DOM 不得再有未关闭+重点一级文本");
assert.doesNotMatch(template, />未关闭·正常推进</, "DOM 不得再有未关闭·正常推进一级文本");

// 周报 Tab 验收断言
assert.match(script, /weeklyScope\s*=\s*"mine"/, "周报默认 scope 必须为 mine");
assert.match(template, /我的项目/, "顶部卡文案必须为我的项目");
assert.doesNotMatch(template, /data-filter="attention"/, "DOM 不得有状态关注筛选");
assert.doesNotMatch(template, /id="pwPeriodSelect"/, "DOM 不得有周期空下拉");
assert.doesNotMatch(template, /<th>本周投入<\/th>/, "DOM 不得有本周投入列");

// 关注动作与操作列样式统一断言
assert.doesNotMatch(script, /<span class=\\"tag\\"[^>]*>关注<\/span>/, "周报项目名旁不得再加关注标签");
assert.doesNotMatch(demandScript, /home-title-line[^"]*">\s*'\s*\+\s*starBtn/, "业务需求标题列不得放置关注星星按钮");
assert.match(demandScript, /data-watch-demand/, "业务需求操作列必须放置关注动作按钮");
assert.match(demandScript, /class="pw-action-btn pw-watch-btn/, "业务需求操作列按钮样式需统一");
assert.match(script, /class="pw-action-btn pw-watch-btn/, "周报操作列按钮样式需统一");
assert.match(script, /weeklyScope === "participated"/, "周报必须兼容处理我参与的项目");

console.log("PASS: follow tabs and semantic controls verified successfully");
