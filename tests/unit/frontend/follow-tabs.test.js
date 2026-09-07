const assert = require("assert");
const fs = require("fs");
const path = require("path");

const root = path.join(__dirname, "../../..");
const template = fs.readFileSync(path.join(root, "web/templates/po/follow.html"), "utf8");
const script = fs.readFileSync(path.join(root, "web/static/js/po/follow.js"), "utf8");

assert.match(template, /class="category-tab follow-tab active" data-tab="demand"/, "业务需求必须是默认视图");
assert.match(template, /data-tab="weekly"/, "项目周报必须保留为已接入视图");
assert.doesNotMatch(template, /data-tab="all"/, "不应保留无数据来源的全部视图");
assert.doesNotMatch(template, /data-tab="risk"|data-tab="testtask"/, "不应展示未接入对象类型");
assert.match(script, /var currentTab = "demand"/, "脚本默认状态必须与页面一致");
assert.match(script, /tab !== "demand" && tab !== "weekly"/, "脚本只接受已接入视图");
assert.doesNotMatch(script, /tabCountAll/, "脚本不能再维护已移除的全部计数");

console.log("PASS: follow tabs only expose connected demand and project-weekly views");
