(() => {
  function escapeRegExp(s) {
    return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }

  function initBatchFormTable(table) {
    const arrayKey = table.getAttribute("data-batch-array-key") || "users";
    const nameRe = new RegExp(escapeRegExp(arrayKey) + "\\[\\d+\\]\\.", "g");
    const rowSelector = "[data-batch-row]";
    const maleValue = "m";

    function getRows() {
      return Array.from(table.querySelectorAll(rowSelector));
    }

    function syncRemoveButtons() {
      const rows = getRows();
      const shouldShowRemove = rows.length > 1;
      rows.forEach((row) => {
        const removeBtn = row.querySelector("[data-batch-remove]");
        if (!removeBtn) {
          return;
        }
        removeBtn.hidden = !shouldShowRemove;
        removeBtn.disabled = !shouldShowRemove;
      });
    }

    function reindexRows() {
      const rows = getRows();
      rows.forEach((row, idx) => {
        row.querySelectorAll("input, select, label").forEach((el) => {
          const name = el.getAttribute("name");
          if (name) {
            el.setAttribute("name", name.replace(nameRe, `${arrayKey}[${idx}].`));
          }

          const id = el.getAttribute("id");
          if (id) {
            el.setAttribute("id", id.replace(/_\d+$/, `_${idx}`));
          }

          const htmlFor = el.getAttribute("for");
          if (htmlFor) {
            el.setAttribute("for", htmlFor.replace(/_\d+$/, `_${idx}`));
          }
        });
      });
    }

    function clearRowValues(row) {
      row.querySelectorAll('input[type="text"], input[type="email"], input[type="password"]').forEach((input) => {
        input.value = "";
        if (!input.hasAttribute("data-batch-preserve-readonly")) {
          input.removeAttribute("readonly");
        }
      });

      row.querySelectorAll('input[type="number"]').forEach((input) => {
        input.value = "";
      });

      row.querySelectorAll('input[type="date"],input[type="datetime-local"],input[type="time"]').forEach((input) => {
        input.value = "";
      });

      row.querySelectorAll('input[type="hidden"]').forEach((input) => {
        input.value = "";
      });

      row.querySelectorAll("select").forEach((select) => {
        const nm = select.getAttribute("name") || "";
        const tbl = select.closest(".batch-form-table");
        const resetAttr = tbl && tbl.hasAttribute("data-batch-reset-attr-on-clone");
        if (resetAttr && nm.indexOf("resourcePriceId") >= 0) {
          select.innerHTML = "";
          return;
        }
        select.selectedIndex = 0;
      });

      row.querySelectorAll('input[type="radio"]').forEach((radio) => {
        radio.checked = radio.value === maleValue;
      });

      row.querySelectorAll('input[type="checkbox"]').forEach((checkbox) => {
        checkbox.checked = false;
      });

      row.querySelectorAll(".form-multiselect-tags").forEach((tags) => {
        tags.innerHTML = "";
      });

      row.querySelectorAll(".form-tree-select-list").forEach((list) => {
        list.innerHTML = "";
      });
    }

    function prepareRowForReinit(row) {
      row.querySelectorAll("[data-form-multiselect-init]").forEach((el) => {
        el.removeAttribute("data-form-multiselect-init");
      });
      row.querySelectorAll("[data-form-tree-select-init]").forEach((el) => {
        el.removeAttribute("data-form-tree-select-init");
      });
    }

    table.addEventListener("click", (event) => {
      const addBtn = event.target.closest("[data-batch-add]");
      if (addBtn) {
        const currentRow = addBtn.closest(rowSelector);
        if (!currentRow) {
          return;
        }
        const newRow = currentRow.cloneNode(true);
        clearRowValues(newRow);
        prepareRowForReinit(newRow);
        currentRow.insertAdjacentElement("afterend", newRow);
        reindexRows();
        if (typeof window.initFormComponents === "function") {
          window.initFormComponents(newRow);
        }
        syncRemoveButtons();
        return;
      }

      const removeBtn = event.target.closest("[data-batch-remove]");
      if (removeBtn) {
        const rows = getRows();
        if (rows.length <= 1) {
          return;
        }
        const currentRow = removeBtn.closest(rowSelector);
        if (!currentRow) {
          return;
        }
        currentRow.remove();
        reindexRows();
        syncRemoveButtons();
      }
    });

    syncRemoveButtons();
  }

  document.querySelectorAll(".batch-form-table").forEach(initBatchFormTable);
})();
