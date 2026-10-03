/**
 * 敏捷小组挂靠团队：复用公共检索下拉。
 */
(function () {
  let options;
  document.addEventListener("agileteam:rows", async function (event) {
    const host = event.target;
    if (!host.querySelector(".at-org-team-picker")) return;
    const inputs = Array.from(host.querySelectorAll(".at-org-team-picker"));
    try {
      options = options || window.appJson("/workbench/api/agile-teams/organization-teams");
      const response = await options;
      const teams = (response.data || []).map(function (team) {
        return { value: String(team.id), label: team.name };
      });
      inputs.forEach(function (input) {
        if (!input.isConnected) return;
        const team = event.detail.find(function (row) { return input.id === "atOrgTeam" + row.id; });
        const hidden = host.querySelector("#atOrgTeamValue" + team.id);
        window.initAutocomplete(input.id, hidden.id, [
          { value: "0", label: team.orgDeptInherited ? "继承父级" : "未挂靠" }
        ].concat(teams), { labelOnly: true, value: hidden.value, label: input.value });
        hidden.addEventListener("change", function () {
          if (hidden.value !== "") window.atMapOrgTeam(team.id, hidden.value);
        });
      });
    } catch (error) {
      options = null;
      window.alert(error.message || "挂靠团队加载失败");
    }
  });
})();
