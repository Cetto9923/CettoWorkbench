/*
 * 文件: web/static/js/po/acceptancerelations.js
 * 模块: PO工作台
 * 职责: 验收抽屉关联表（转化研发需求/用户故事/评审信息/工单）回填。
 */
(function (root) {
  "use strict";

  function escapeHtml(text) {
    return String(text == null ? "" : text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function dash(value) {
    var text = String(value == null ? "" : value).trim();
    return text || "—";
  }

  function emptyTableRow(colspan, text) {
    return (
      '<tr><td colspan="' +
      colspan +
      '" class="text-muted">' +
      escapeHtml(text || "暂无") +
      "</td></tr>"
    );
  }

  function linkOrText(url, text) {
    var label = escapeHtml(text);
    var href = String(url || "").trim();
    if (!href) {
      return label || "—";
    }
    return (
      '<a href="' +
      escapeHtml(href) +
      '" target="_blank" rel="noopener noreferrer">' +
      (label || "—") +
      "</a>"
    );
  }

  function setTbody(id, html) {
    var el = document.getElementById(id);
    if (el) {
      el.innerHTML = html;
    }
  }

  function resetRelationTables(text) {
    setTbody("poDemandAcceptanceStories", emptyTableRow(5, text || "加载中…"));
    setTbody("poDemandAcceptanceUserStories", emptyTableRow(5, text || "加载中…"));
    setTbody("poDemandAcceptanceReviewRecords", emptyTableRow(6, text || "加载中…"));
    setTbody("poDemandAcceptanceTickets", emptyTableRow(4, text || "加载中…"));
  }

  function renderStories(list) {
    var rows = Array.isArray(list) ? list : [];
    if (!rows.length) {
      setTbody("poDemandAcceptanceStories", emptyTableRow(5, "暂无"));
      return;
    }
    setTbody(
      "poDemandAcceptanceStories",
      rows
        .map(function (s) {
          return (
            "<tr><td>" +
            escapeHtml(s && s.id) +
            '</td><td class="dd-table-title">' +
            linkOrText(s && s.zentaoUrl, s && s.title) +
            "</td><td>" +
            escapeHtml(dash(s && s.stageLabel)) +
            "</td><td>" +
            escapeHtml(dash(s && s.productName)) +
            "</td><td>" +
            escapeHtml(dash(s && s.releaseDate)) +
            "</td></tr>"
          );
        })
        .join("")
    );
  }

  function renderUserStories(list) {
    var rows = Array.isArray(list) ? list : [];
    if (!rows.length) {
      setTbody("poDemandAcceptanceUserStories", emptyTableRow(5, "暂无"));
      return;
    }
    setTbody(
      "poDemandAcceptanceUserStories",
      rows
        .map(function (s) {
          return (
            "<tr><td>" +
            escapeHtml(s && s.no) +
            "</td><td>" +
            escapeHtml(dash(s && s.role)) +
            '</td><td class="dd-table-title">' +
            escapeHtml(dash(s && s.gv)) +
            "</td><td>" +
            escapeHtml(dash(s && s.productName)) +
            "</td><td>" +
            escapeHtml(dash(s && s.pointLabel)) +
            "</td></tr>"
          );
        })
        .join("")
    );
  }

  function renderReviewRecords(list) {
    var rows = Array.isArray(list) ? list : [];
    if (!rows.length) {
      setTbody("poDemandAcceptanceReviewRecords", emptyTableRow(6, "暂无"));
      return;
    }
    setTbody(
      "poDemandAcceptanceReviewRecords",
      rows
        .map(function (r) {
          return (
            "<tr><td>" +
            escapeHtml(dash(r && r.reviewTypeLabel)) +
            "</td><td>" +
            escapeHtml(dash(r && r.reviewDate)) +
            '</td><td class="dd-table-title">' +
            escapeHtml(dash(r && r.reviewResult)) +
            "</td><td>" +
            escapeHtml(dash(r && r.createdByName)) +
            "</td><td>" +
            escapeHtml(dash(r && r.createdDate)) +
            "</td><td>" +
            escapeHtml(dash(r && r.reviewStatusLabel)) +
            "</td></tr>"
          );
        })
        .join("")
    );
  }

  function renderTickets(list) {
    var rows = Array.isArray(list) ? list : [];
    if (!rows.length) {
      setTbody("poDemandAcceptanceTickets", emptyTableRow(4, "暂无"));
      return;
    }
    setTbody(
      "poDemandAcceptanceTickets",
      rows
        .map(function (t) {
          return (
            "<tr><td>" +
            escapeHtml(t && t.id) +
            '</td><td class="dd-table-title">' +
            linkOrText(t && t.zentaoUrl, t && t.title) +
            "</td><td>" +
            escapeHtml(dash(t && t.pri)) +
            "</td><td>" +
            escapeHtml(dash(t && t.statusLabel)) +
            "</td></tr>"
          );
        })
        .join("")
    );
  }

  function renderRelationTables(data) {
    renderStories(data && data.stories);
    renderUserStories(data && data.userStories);
    renderReviewRecords(data && data.reviewRecords);
    renderTickets(data && data.tickets);
  }


  root.PoAcceptanceRelations = {
    reset: resetRelationTables,
    render: renderRelationTables
  };
})(typeof window !== "undefined" ? window : this);
