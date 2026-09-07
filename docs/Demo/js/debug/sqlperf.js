/**
 * SQL 性能分析页
 * 数据通过 /debug/sqlperf/requests 动态读取 logs/sql.log
 */
const SQLPERF_API = "/debug/sqlperf/requests";

const SQLPERF_TREND_ZOOM_THRESHOLD = 15;

function dateKey(timeText) {
  return timeText.slice(0, 10);
}

function sortRequests(requests) {
  return requests.slice().sort((a, b) => a.time.localeCompare(b.time));
}

function uniqueDates(requests) {
  const set = new Set();
  requests.forEach((item) => set.add(dateKey(item.time)));
  return Array.from(set).sort();
}

function filterByDateRange(requests, startDate, endDate) {
  if (!requests.length) {
    return requests;
  }
  const start = startDate || dateKey(requests[0].time);
  const end = endDate || dateKey(requests[requests.length - 1].time);
  return requests.filter((item) => {
    const day = dateKey(item.time);
    return day >= start && day <= end;
  });
}

function initDateRangeInputs(startInput, endInput, requests) {
  const dates = uniqueDates(requests);
  if (!dates.length) {
    return;
  }
  const minDate = dates[0];
  const maxDate = dates[dates.length - 1];
  startInput.min = minDate;
  startInput.max = maxDate;
  endInput.min = minDate;
  endInput.max = maxDate;
  startInput.value = maxDate;
  endInput.value = maxDate;
}

function routeKey(item) {
  return item.method + " " + item.route;
}

function hourKey(timeText) {
  return timeText.slice(0, 13);
}

function uniqueRoutes(requests) {
  const set = new Set();
  requests.forEach((item) => set.add(routeKey(item)));
  return Array.from(set).sort();
}

function uniqueHours(requests) {
  const set = new Set();
  requests.forEach((item) => set.add(hourKey(item.time)));
  return Array.from(set).sort();
}

function formatHourLabel(hour) {
  return hour.slice(0, 10) + " " + hour.slice(11) + ":00";
}

const SQLPERF_CHART_GRID = {
  left: 16,
  right: 16,
  top: 64,
  bottom: 16,
  containLabel: true,
};

const SQLPERF_TREND_GRID = {
  left: 16,
  right: 16,
  top: 64,
  bottom: 56,
  containLabel: true,
};

function formatTrendAxisLabel(timeText) {
  const space = timeText.indexOf(" ");
  if (space < 0) {
    return timeText;
  }
  return timeText.slice(space + 1, space + 9);
}

function formatCompareAxisLabel(label) {
  const space = label.indexOf(" ");
  const text = space >= 0 ? label.slice(space + 1) : label;
  if (text.length <= 14) {
    return text;
  }
  return text.slice(0, 12) + "…";
}

function renderSummary(allRequests, filteredRequests) {
  const el = document.getElementById("sqlperf-summary");
  if (!el) {
    return;
  }
  const routes = uniqueRoutes(filteredRequests);
  const times = filteredRequests.map((item) => item.time);
  const minTime = times.length ? times[0] : "-";
  const maxTime = times.length ? times[times.length - 1] : "-";
  const rangeHint =
    filteredRequests.length === allRequests.length
      ? ""
      : "（已筛选 " + filteredRequests.length + " / " + allRequests.length + " 条）";
  el.textContent =
    "当前范围 " +
    filteredRequests.length +
    " 条" +
    rangeHint +
    " · 涉及接口 " +
    routes.length +
    " 个 · 时间 " +
    minTime +
    " ~ " +
    maxTime +
    " · 数据源 logs/sql.log";
}

function renderLoadState(message, isError) {
  const summary = document.getElementById("sqlperf-summary");
  const app = document.getElementById("sqlperf-app");
  if (summary) {
    summary.textContent = message;
    summary.style.color = isError ? "var(--color-danger)" : "";
  }
  if (app) {
    app.style.display = isError ? "none" : "";
  }
}

function renderEmptyState() {
  renderLoadState("当前 logs/sql.log 中暂无请求汇总数据，请先产生 HTTP 请求后再查看。", false);
  const app = document.getElementById("sqlperf-app");
  if (app) {
    app.style.display = "none";
  }
}

async function loadSQLPerfRequests() {
  const res = await fetch(SQLPERF_API, {
    headers: { Accept: "application/json" },
    credentials: "same-origin",
  });
  if (res.status === 401) {
    window.location.href = "/login";
    return null;
  }
  const data = await res.json();
  if (!res.ok || !data.success) {
    throw new Error((data && data.message) || "加载 SQL 性能数据失败");
  }
  return Array.isArray(data.requests) ? data.requests : [];
}

function updateTrendContext(requests, selectedRoute) {
  const el = document.getElementById("sqlperf-trend-context");
  if (!el) {
    return;
  }
  const filtered = requests.filter((item) => routeKey(item) === selectedRoute);
  if (!filtered.length) {
    el.innerHTML = "当前接口 <strong>" + selectedRoute + "</strong> 暂无请求记录。";
    return;
  }
  const elapsedValues = filtered.map((item) => item.elapsed_ms);
  const minElapsed = Math.min.apply(null, elapsedValues);
  const maxElapsed = Math.max.apply(null, elapsedValues);
  const avgElapsed = (
    elapsedValues.reduce((sum, value) => sum + value, 0) / elapsedValues.length
  ).toFixed(2);
  const sqlValues = filtered.map((item) => item.sql_count);
  const minSQL = Math.min.apply(null, sqlValues);
  const maxSQL = Math.max.apply(null, sqlValues);
  const avgSQL = (
    sqlValues.reduce((sum, value) => sum + value, 0) / sqlValues.length
  ).toFixed(1);
  el.innerHTML =
    "上图仅分析接口 <strong>" +
    selectedRoute +
    "</strong>，共 <strong>" +
    filtered.length +
    "</strong> 次请求 · 耗时 " +
    minElapsed +
    " ~ " +
    maxElapsed +
    " ms（均 " +
    avgElapsed +
    " ms）· SQL 次数 " +
    minSQL +
    " ~ " +
    maxSQL +
    " 次（均 <strong>" +
    avgSQL +
    " 次</strong>）";
}

function updateCompareContext(requests, selectedHour) {
  const el = document.getElementById("sqlperf-compare-context");
  if (!el) {
    return;
  }
  const filtered = requests.filter((item) => hourKey(item.time) === selectedHour);
  const routeCount = new Set(filtered.map(routeKey)).size;
  if (!filtered.length) {
    el.innerHTML =
      "下图仅分析小时 <strong>" +
      formatHourLabel(selectedHour) +
      "</strong>，该时段暂无请求记录。";
    return;
  }
  const totalSQL = filtered.reduce((sum, item) => sum + item.sql_count, 0);
  const avgSQL = (totalSQL / filtered.length).toFixed(1);
  el.innerHTML =
    "下图仅分析小时 <strong>" +
    formatHourLabel(selectedHour) +
    "</strong>，共 <strong>" +
    filtered.length +
    "</strong> 次请求 · 涉及 <strong>" +
    routeCount +
    "</strong> 个接口 · SQL 查询均 <strong>" +
    avgSQL +
    " 次</strong>";
}

function buildTrendZoom(pointCount) {
  if (pointCount <= SQLPERF_TREND_ZOOM_THRESHOLD) {
    return [];
  }
  const endPercent = Math.min(100, Math.round((SQLPERF_TREND_ZOOM_THRESHOLD / pointCount) * 100));
  return [
    { type: "inside", xAxisIndex: 0 },
    {
      type: "slider",
      xAxisIndex: 0,
      height: 20,
      bottom: 4,
      start: 100 - endPercent,
      end: 100,
    },
  ];
}

function renderRouteTrend(chart, requests, selectedRoute) {
  const filtered = requests
    .filter((item) => routeKey(item) === selectedRoute)
    .sort((a, b) => a.time.localeCompare(b.time));
  const useZoom = filtered.length > SQLPERF_TREND_ZOOM_THRESHOLD;

  chart.setOption({
    color: ["#6366F1", "#F59E0B"],
    title: {
      text: selectedRoute,
      left: "center",
      top: 0,
      textStyle: { fontSize: 12, fontWeight: 500, color: "#6B7280" },
    },
    legend: {
      data: ["耗时", "SQL 次数"],
      top: 24,
      textStyle: { fontSize: 12 },
    },
    tooltip: {
      trigger: "axis",
      formatter: function (params) {
        const idx = params[0].dataIndex;
        const item = filtered[idx];
        if (!item) {
          return "";
        }
        const lines = [item.time];
        params.forEach(function (point) {
          if (point.seriesName === "耗时") {
            lines.push(point.marker + "耗时: " + item.elapsed);
          }
          if (point.seriesName === "SQL 次数") {
            lines.push(point.marker + "SQL 次数: " + item.sql_count);
          }
        });
        lines.push("request_id: " + item.request_id);
        return lines.join("<br/>");
      },
    },
    grid: useZoom ? SQLPERF_TREND_GRID : SQLPERF_CHART_GRID,
    dataZoom: buildTrendZoom(filtered.length),
    xAxis: {
      type: "category",
      name: "请求时间",
      data: filtered.map((item) => formatTrendAxisLabel(item.time)),
      axisLabel: { fontSize: 12, hideOverlap: true },
    },
    yAxis: [
      {
        type: "value",
        name: "耗时(ms)",
        position: "left",
        axisLabel: { fontSize: 12 },
      },
      {
        type: "value",
        name: "SQL 次数",
        position: "right",
        minInterval: 1,
        axisLabel: { fontSize: 12 },
      },
    ],
    series: [
      {
        name: "耗时",
        type: "line",
        smooth: true,
        symbolSize: 8,
        yAxisIndex: 0,
        data: filtered.map((item) => item.elapsed_ms),
      },
      {
        name: "SQL 次数",
        type: "line",
        smooth: true,
        symbolSize: 8,
        yAxisIndex: 1,
        data: filtered.map((item) => item.sql_count),
      },
    ],
  }, { replaceMerge: ["dataZoom"] });
}

function renderRouteCompare(chart, requests, selectedHour) {
  const grouped = {};
  requests
    .filter((item) => hourKey(item.time) === selectedHour)
    .forEach((item) => {
      const key = routeKey(item);
      if (!grouped[key]) {
        grouped[key] = { total: 0, count: 0, sql_count: 0 };
      }
      grouped[key].total += item.elapsed_ms;
      grouped[key].count += 1;
      grouped[key].sql_count += item.sql_count;
    });

  const labels = Object.keys(grouped).sort();
  const values = labels.map((key) => Number((grouped[key].total / grouped[key].count).toFixed(2)));
  const avgSQLCounts = labels.map((key) =>
    Number((grouped[key].sql_count / grouped[key].count).toFixed(1))
  );

  chart.setOption({
    color: ["#10B981", "#F59E0B"],
    title: {
      text: formatHourLabel(selectedHour),
      left: "center",
      top: 0,
      textStyle: { fontSize: 12, fontWeight: 500, color: "#6B7280" },
    },
    legend: {
      data: ["平均耗时", "平均 SQL 次数"],
      top: 24,
      textStyle: { fontSize: 12 },
    },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "cross" },
      formatter: function (params) {
        const idx = params[0].dataIndex;
        const lines = [labels[idx]];
        params.forEach(function (point) {
          if (point.seriesName === "平均耗时") {
            lines.push(point.marker + "平均耗时: " + values[idx] + " ms");
          }
          if (point.seriesName === "平均 SQL 次数") {
            lines.push(point.marker + "平均 SQL 次数: " + avgSQLCounts[idx]);
          }
        });
        return lines.join("<br/>");
      },
    },
    grid: SQLPERF_CHART_GRID,
    xAxis: {
      type: "category",
      name: "接口",
      data: labels,
      axisLabel: {
        fontSize: 12,
        hideOverlap: true,
        formatter: formatCompareAxisLabel,
      },
    },
    yAxis: [
      {
        type: "value",
        name: "平均耗时(ms)",
        position: "left",
        axisLabel: { fontSize: 12 },
      },
      {
        type: "value",
        name: "平均 SQL 次数",
        position: "right",
        axisLabel: { fontSize: 12 },
      },
    ],
    series: [
      {
        name: "平均耗时",
        type: "bar",
        yAxisIndex: 0,
        barMaxWidth: 40,
        data: values,
      },
      {
        name: "平均 SQL 次数",
        type: "line",
        yAxisIndex: 1,
        smooth: true,
        symbolSize: 8,
        data: avgSQLCounts,
      },
    ],
  });
}

function fillSelect(select, options, selectedValue, labelFormatter) {
  select.innerHTML = "";
  options.forEach((value) => {
    const option = document.createElement("option");
    option.value = value;
    option.textContent = labelFormatter ? labelFormatter(value) : value;
    if (value === selectedValue) {
      option.selected = true;
    }
    select.appendChild(option);
  });
}

function initSQLPerfPage(allRequests) {
  const app = document.getElementById("sqlperf-app");
  if (app) {
    app.style.display = "";
  }
  const dateStartInput = document.getElementById("sqlperf-date-start");
  const dateEndInput = document.getElementById("sqlperf-date-end");
  const routeSelect = document.getElementById("sqlperf-route");
  const hourSelect = document.getElementById("sqlperf-hour");
  const trendEl = document.getElementById("sqlperf-route-trend");
  const compareEl = document.getElementById("sqlperf-route-compare");

  if (!dateStartInput || !dateEndInput || !routeSelect || !hourSelect || !trendEl || !compareEl) {
    return;
  }

  initDateRangeInputs(dateStartInput, dateEndInput, allRequests);

  let filteredRequests = filterByDateRange(
    allRequests,
    dateStartInput.value,
    dateEndInput.value
  );

  const trendChart = echarts.init(trendEl);
  const compareChart = echarts.init(compareEl);

  function refreshFilters() {
    const routes = uniqueRoutes(filteredRequests);
    const hours = uniqueHours(filteredRequests);
    const currentRoute = routeSelect.value;
    const currentHour = hourSelect.value;
    const selectedRoute = routes.includes(currentRoute) ? currentRoute : routes[0] || "";
    const selectedHour = hours.includes(currentHour) ? currentHour : hours[hours.length - 1] || "";
    fillSelect(routeSelect, routes, selectedRoute, null);
    fillSelect(hourSelect, hours, selectedHour, formatHourLabel);
  }

  function refreshTrendChart() {
    updateTrendContext(filteredRequests, routeSelect.value);
    renderRouteTrend(trendChart, filteredRequests, routeSelect.value);
  }

  function refreshCompareChart() {
    updateCompareContext(filteredRequests, hourSelect.value);
    renderRouteCompare(compareChart, filteredRequests, hourSelect.value);
  }

  function refreshPage() {
    refreshFilters();
    refreshTrendChart();
    refreshCompareChart();
  }

  function applyDateRange() {
    if (dateStartInput.value && dateEndInput.value && dateStartInput.value > dateEndInput.value) {
      dateEndInput.value = dateStartInput.value;
    }
    filteredRequests = filterByDateRange(
      allRequests,
      dateStartInput.value,
      dateEndInput.value
    );
    refreshPage();
  }

  dateStartInput.addEventListener("change", applyDateRange);
  dateEndInput.addEventListener("change", applyDateRange);
  routeSelect.addEventListener("change", refreshTrendChart);
  hourSelect.addEventListener("change", refreshCompareChart);
  window.addEventListener("resize", function () {
    trendChart.resize();
    compareChart.resize();
  });

  refreshPage();
}

async function bootstrapSQLPerfPage() {
  renderLoadState("正在加载 SQL 性能数据…", false);
  const app = document.getElementById("sqlperf-app");
  if (app) {
    app.style.display = "none";
  }
  try {
    const requests = await loadSQLPerfRequests();
    if (requests === null) {
      return;
    }
    const sorted = sortRequests(requests);
    if (!sorted.length) {
      renderEmptyState();
      return;
    }
    initSQLPerfPage(sorted);
  } catch (err) {
    renderLoadState(err.message || "加载 SQL 性能数据失败", true);
  }
}

document.addEventListener("DOMContentLoaded", bootstrapSQLPerfPage);
