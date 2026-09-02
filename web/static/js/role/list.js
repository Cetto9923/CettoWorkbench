(() => {
  "use strict";

  const panel = document.querySelector(".role-table-panel");
  if (!panel) {
    return;
  }

  const searchInput = panel.querySelector("#roleListSearch");
  const tbody = panel.querySelector(".table tbody");
  if (!tbody) {
    return;
  }

  const getDataRows = () => Array.from(tbody.querySelectorAll("tr[data-role-search]"));
  const noMatchRow = tbody.querySelector("tr.role-js-no-match");

  const applyClientFilter = () => {
    const rows = getDataRows();
    if (rows.length === 0) {
      if (noMatchRow) {
        noMatchRow.hidden = true;
      }
      return;
    }

    const kw = String(searchInput ? searchInput.value : "").trim().toLowerCase();
    let visible = 0;

    rows.forEach((row) => {
      const hay = String(row.getAttribute("data-role-search") || "").toLowerCase();
      const show = !kw || hay.includes(kw);
      row.hidden = !show;
      if (show) {
        visible += 1;
      }
    });

    if (noMatchRow) {
      const showNoMatch = Boolean(kw) && visible === 0;
      noMatchRow.hidden = !showNoMatch;
    }
  };

  if (searchInput) {
    searchInput.addEventListener("input", applyClientFilter);
    if (String(searchInput.value || "").trim()) {
      applyClientFilter();
    }
  }
})();
