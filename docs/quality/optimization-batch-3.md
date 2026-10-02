# 第 3 批变更记录

所有变更位于独立优化分支，没有执行迁移、生产写入或部署。

本批最终工作树验证：`GOPROXY=off make check` 与 `make quality` 通过（隔离数据库并发测试未运行）。

## 文件净行数

| 文件 | 净行数 |
|---|---:|
| `Makefile` | 2 |
| `db/upgrade_workbench_icons.sql` | 84 |
| `docs/quality/workbench-optimization.md` | 92 |
| `tests/unit/frontend/home-team-view-runtime.test.js` | 0 |
| `tests/unit/frontend/issue-risk-team-scope.test.js` | 0 |
| `tests/unit/frontend/submit-review-empty-candidates.test.js` | 0 |
| `web/static/css/auth/login.css` | -3 |
| `web/static/css/components/modal.css` | 5 |
| `web/static/css/tokens.css` | 138 |
| `web/static/js/agileteam/agileteam.js` | -21 |
| `web/static/js/app.js` | -17 |
| `web/static/js/components/components.js` | -19 |
| `web/static/js/components/form.js` | 0 |
| `web/static/js/components/search.js` | 0 |
| `web/static/js/debug/apilog.js` | -6 |
| `web/static/js/debug/sqllog.js` | -6 |
| `web/static/js/dept/list.js` | 0 |
| `web/static/js/layout/bottomtabbar.js` | 0 |
| `web/static/js/menu/create.js` | 0 |
| `web/static/js/menu/list.js` | 0 |
| `web/static/js/po/demand-detail-flow.js` | 0 |
| `web/static/js/po/demand-detail-render.js` | 0 |
| `web/static/js/po/po-profile.js` | -56 |
| `web/static/js/schedule/schedule-inline.js` | 0 |
| `web/static/js/schedule/schedulefetch.js` | -51 |
| `web/static/js/schedule/scheduleintegrated.js` | -2 |
| `web/static/js/schedule/scheduleintegratedshared.js` | -2 |
| `web/static/js/schedule/scheduleintegratedtasks.js` | -24 |
| `web/static/js/schedule/schedulemanage.js` | 0 |
| `web/static/js/schedule/scheduletasklistmodal.js` | -27 |
| `web/static/js/schedule/scheduletaskmodal.js` | -63 |
| `web/static/js/schedule/schedulewindow.js` | -4 |
| `web/static/js/schedule/schedulewindowdraft.js` | 0 |
| `web/static/js/ui.js` | -72 |
| `web/static/vendor/bootstrap-icons/bootstrap-icons.css` | -2078 |
| `web/static/vendor/bootstrap-icons/fonts/UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa0ZL7SUc.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa1ZL7.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa1pL7SUc.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa25L7SUc.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa2JL7SUc.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa2ZL7SUc.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa2pL7SUc.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/bootstrap-icons.woff` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/bootstrap-icons.woff2` | 二进制资源 |
| `web/static/vendor/bootstrap-icons/fonts/fonts.css` | -210 |
| `web/static/vendor/bootstrap/css/bootstrap.min.css` | -6 |
| `web/static/vendor/bootstrap/js/bootstrap.bundle.min.js` | -7 |
| `web/templates/components/exportmodal.html` | 0 |
| `web/templates/components/pager.html` | 0 |
| `web/templates/components/searchpanel.html` | 0 |
| `web/templates/components/ui/action_group.html` | 0 |
| `web/templates/dept/create.html` | 0 |
| `web/templates/dept/edit.html` | 0 |
| `web/templates/dept/list.html` | 0 |
| `web/templates/layout/auth.html` | 0 |
| `web/templates/layout/base.html` | 1 |
| `web/templates/layout/header.html` | 0 |
| `web/templates/layout/sidebar.html` | 0 |
| `web/templates/layout/themepicker.html` | 0 |
| `web/templates/layout/topnav.html` | 0 |
| `web/templates/loginlog/list.html` | 0 |
| `web/templates/menu/create.html` | 0 |
| `web/templates/menu/edit.html` | 0 |
| `web/templates/menu/list.html` | 0 |
| `web/templates/operationlog/list.html` | 0 |
| `web/templates/po/home.html` | -1 |
| `web/templates/po/linkstory.html` | 0 |
| `web/templates/role/create.html` | 0 |
| `web/templates/role/edit.html` | 0 |
| `web/templates/role/list.html` | 0 |
| `web/templates/schedule/index.html` | -1 |
| `web/templates/user/batchcreate.html` | 0 |
| `web/templates/user/create.html` | 0 |
| `web/templates/user/edit.html` | 0 |
| `web/templates/user/list.html` | 0 |

## 新增函数 / 方法声明


未添加第三方依赖；没有移动生产目录、压行或生成代码来掩盖增长。已有超过 500 行文件在本批均不得净增长。CSS 选择器变化在最终总报告单列。
