#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const assert = require('assert');

const root = path.resolve(__dirname, '../../..');

const filterJs = fs.readFileSync(path.join(root, 'web/static/js/schedule/schedulefilter.js'), 'utf8');
const scheduleJs = fs.readFileSync(path.join(root, 'web/static/js/schedule/schedule.js'), 'utf8');
const integratedJs = fs.readFileSync(path.join(root, 'web/static/js/schedule/scheduleintegrated.js'), 'utf8');
const taskModalJs = fs.readFileSync(path.join(root, 'web/static/js/schedule/scheduletaskmodal.js'), 'utf8');
const indexHtml = fs.readFileSync(path.join(root, 'web/templates/schedule/index.html'), 'utf8');
const filterCss = fs.readFileSync(path.join(root, 'web/static/css/schedule/schedulefilter.css'), 'utf8');

// 1. 窗口卡片点击进入本窗口模式（锁定该窗 + 切换全部未关闭）
assert(
  /filter:\s*['"]all_open['"]/.test(filterJs) && /windows:\s*windowId/.test(filterJs),
  'schedulefilter.js: 点击窗口卡片必须锁定 windows 参数并切至 all_open（不得停留在 unscheduled）'
);

// 2. 点击待排期或返回待排期清除 windows 锁定
assert(
  /btnReturnUnscheduled/.test(scheduleJs) && /overrides\.windows\s*=\s*null/.test(scheduleJs),
  'schedule.js: 点击待排期或返回待排期按钮必须清除 windows 参数并回到待排期'
);

// 3. 角标计数请求必须透传 windows, groups, products, stages 参数
assert(
  /buildFilterCountsURL/.test(scheduleJs) &&
  /windows/.test(scheduleJs) &&
  /groups/.test(scheduleJs) &&
  /products/.test(scheduleJs) &&
  /stages/.test(scheduleJs),
  'schedule.js: buildFilterCountsURL 必须透传 windows/groups/products/stages 参数'
);

// 4. 保存排期跳转目标窗口并高亮刚保存行
assert(
  /\/schedule\?windows=/.test(integratedJs) &&
  /filter=all_open/.test(integratedJs) &&
  /highlight=/.test(integratedJs) &&
  /toast\(.*保留当前视图/.test(integratedJs),
  'scheduleintegrated.js: 保存成功后必须跳至目标窗口的本窗口模式并高亮，无窗口时保留当前视图'
);

// 5. 任务弹窗保存后跳至目标窗口并高亮
assert(
  /\/schedule\?windows=/.test(taskModalJs) &&
  /filter=all_open/.test(taskModalJs) &&
  /highlight=/.test(taskModalJs) &&
  /toast\(.*保留当前视图/.test(taskModalJs),
  'scheduletaskmodal.js: 任务保存成功后若存在目标窗口必须跳至该窗口并高亮，无窗口时保留当前视图'
);

// 6. 现网模板结构与隐藏阻塞0
assert(
  !/阻塞\s*0/.test(indexHtml),
  'index.html: 窗口卡片必须隐藏写死的阻塞0'
);
assert(
  /windowActiveBanner/.test(indexHtml) && /schedule-window-active-banner/.test(indexHtml),
  'index.html: 必须渲染本窗口模式专属摘要条 #windowActiveBanner'
);
assert(
  /action-btn--secondary\s+js-change-window/.test(indexHtml),
  'index.html: 本窗口模式下必须渲染次要样式换窗口按钮'
);

// 7. 样式规则：高度约束、无折行、高亮呼吸动效
assert(
  /scheduleHighlightPulse/.test(filterCss) && /schedule-row-highlighted/.test(filterCss),
  'schedulefilter.css: 必须定义 scheduleHighlightPulse 动画与 schedule-row-highlighted 类'
);
assert(
  /schedule-window-active-banner/.test(filterCss) && /height:\s*36px/.test(filterCss),
  'schedulefilter.css: 摘要条高度必须 <= 40px'
);
assert(
  /white-space:\s*nowrap/.test(filterCss),
  'schedulefilter.css: 排期阶段等标签必须设置 nowrap 防止折行'
);

console.log('schedule workbench & window capacity tests passed');
