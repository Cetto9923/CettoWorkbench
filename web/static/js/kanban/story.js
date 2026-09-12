/* =============================================================================
   文件: web/static/js/kanban/story.js
   模块: 工作看板
   职责: 需求看板页交互（敏捷小组 chips 高亮切换）。
============================================================================= */
(function () {
  "use strict";

  var root = document.querySelector(".po-board");
  if (!root) return;

  var chipsWrap = root.querySelector("[data-teamgroup-chips]");
  if (!chipsWrap) return;

  chipsWrap.addEventListener("click", function (ev) {
    var chip = ev.target.closest(".chip");
    if (!chip || !chipsWrap.contains(chip)) return;

    var wasActive = chip.classList.contains("active");
    chipsWrap.querySelectorAll(".chip.active").forEach(function (el) {
      el.classList.remove("active");
    });
    if (!wasActive) {
      chip.classList.add("active");
    }
  });
})();
