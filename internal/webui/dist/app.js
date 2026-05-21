const $ = (id) => document.getElementById(id);
const state = {
  adminToken: localStorage.getItem("tp_admin_token") || "",
  userToken: localStorage.getItem("tp_user_token") || "",
  selectedDeviceID: localStorage.getItem("tp_selected_device") || "",
  user: null,
  devices: [],
  todos: [],
  telemetry: []
};

function init() {
  $("adminToken").value = state.adminToken;
  $("userToken").value = state.userToken;
  bindEvents();
  checkHealth();
  if (state.userToken) loadMe();
  refreshDevices();
  refreshTodos();
}

function bindEvents() {
  document.querySelectorAll(".tabs button").forEach((btn) => {
    btn.addEventListener("click", () => switchTab(btn.dataset.tab));
  });
  $("saveTokensBtn").onclick = saveTokens;
  $("createUserBtn").onclick = createUser;
  $("loadMeBtn").onclick = loadMe;
  $("bindDeviceBtn").onclick = bindDevice;
  $("refreshDevicesBtn").onclick = refreshDevices;
  $("renameDeviceBtn").onclick = renameDevice;
  $("deleteDeviceBtn").onclick = deleteDevice;
  $("sendMessageBtn").onclick = sendMessage;
  $("refreshMessagesBtn").onclick = refreshMessages;
  $("createTodoBtn").onclick = createTodo;
  $("refreshTodosBtn").onclick = refreshTodos;
  $("refreshWeatherBtn").onclick = refreshWeather;
  $("refreshSnapshotBtn").onclick = refreshSnapshot;
  $("refreshTelemetryBtn").onclick = refreshTelemetry;
  $("clearLogBtn").onclick = () => $("rawLog").textContent = "Ready.";
  $("closeTelemetryModalBtn").onclick = closeTelemetryModal;
  $("telemetryModal").onclick = (event) => {
    if (event.target === $("telemetryModal")) closeTelemetryModal();
  };
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") closeTelemetryModal();
  });
}

function switchTab(name) {
  document.querySelectorAll(".tabs button").forEach((btn) => btn.classList.toggle("active", btn.dataset.tab === name));
  document.querySelectorAll(".tab-panel").forEach((panel) => panel.classList.toggle("active", panel.id === "tab-" + name));
}

function saveTokens() {
  state.adminToken = $("adminToken").value.trim();
  state.userToken = $("userToken").value.trim();
  localStorage.setItem("tp_admin_token", state.adminToken);
  localStorage.setItem("tp_user_token", state.userToken);
  notice("identityNotice", "已保存到浏览器。", "ok");
  updateHero();
}

async function api(path, options = {}) {
  const headers = new Headers(options.headers || {});
  if (options.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const res = await fetch(path, { ...options, headers });
  let data = null;
  if (res.status !== 204) {
    const text = await res.text();
    data = text ? JSON.parse(text) : null;
  }
  log({ method: options.method || "GET", path, status: res.status, response: data });
  if (!res.ok) {
    const message = data?.error || res.statusText;
    throw new Error(message);
  }
  return data;
}

function adminHeaders() {
  return { Authorization: "Bearer " + $("adminToken").value.trim() };
}

function userHeaders() {
  return { Authorization: "Bearer " + $("userToken").value.trim() };
}

async function checkHealth() {
  try {
    await api("/healthz");
    $("healthDot").classList.add("ok");
  } catch {
    $("healthDot").classList.remove("ok");
  }
}

async function createUser() {
  try {
    const body = {
      name: $("newUserName").value.trim(),
      email: $("newUserEmail").value.trim(),
      api_token: $("newUserToken").value.trim()
    };
    const data = await api("/api/v1/admin/users", {
      method: "POST",
      headers: adminHeaders(),
      body: JSON.stringify(body)
    });
    $("userToken").value = data.api_token;
    saveTokens();
    state.user = data.user;
    notice("identityNotice", "用户已创建并切换。", "ok");
    updateHero();
  } catch (err) {
    notice("identityNotice", err.message, "error");
  }
}

async function loadMe() {
  try {
    const data = await api("/api/v1/me", { headers: userHeaders() });
    state.user = data;
    notice("identityNotice", `当前用户：${data.name || data.id}`, "ok");
    updateHero();
  } catch (err) {
    notice("identityNotice", err.message, "error");
  }
}

async function bindDevice() {
  try {
    const data = await api("/api/v1/devices/bind", {
      method: "POST",
      headers: userHeaders(),
      body: JSON.stringify({
        bind_code: $("bindCode").value.trim(),
        name: $("bindName").value.trim()
      })
    });
    state.selectedDeviceID = data.id;
    localStorage.setItem("tp_selected_device", data.id);
    notice("deviceNotice", "设备已绑定。", "ok");
    await refreshDevices();
  } catch (err) {
    notice("deviceNotice", err.message, "error");
  }
}

async function refreshDevices() {
  try {
    if (!$("userToken").value.trim()) return;
    state.devices = await api("/api/v1/devices", { headers: userHeaders() });
    if (!state.selectedDeviceID && state.devices.length) state.selectedDeviceID = state.devices[0].id;
    if (state.selectedDeviceID && !state.devices.some((d) => d.id === state.selectedDeviceID)) {
      state.selectedDeviceID = state.devices[0]?.id || "";
    }
    localStorage.setItem("tp_selected_device", state.selectedDeviceID);
    renderDevices();
    updateHero();
    if (state.selectedDeviceID) {
      refreshMessages();
      refreshTelemetry();
    }
  } catch (err) {
    renderError("devicesList", err.message);
  }
}

function renderDevices() {
  const box = $("devicesList");
  if (!state.devices.length) {
    box.innerHTML = `<div class="empty">暂无设备。先让设备 Hello，再用绑定码绑定。</div>`;
    return;
  }
  box.innerHTML = state.devices.map((d) => `
      <div class="item">
        <div class="item-row">
          <div>
            <div class="item-title">${escapeHTML(d.name || d.id)}</div>
            <div class="muted mono">${escapeHTML(d.id)}</div>
          </div>
          <button class="${d.id === state.selectedDeviceID ? "accent" : "secondary"}" data-select-device="${escapeAttr(d.id)}">选择</button>
        </div>
        <div class="muted">last seen: ${formatDate(d.last_seen_at)}</div>
      </div>
    `).join("");
  box.querySelectorAll("[data-select-device]").forEach((btn) => {
    btn.onclick = () => {
      state.selectedDeviceID = btn.dataset.selectDevice;
      localStorage.setItem("tp_selected_device", state.selectedDeviceID);
      $("renameDeviceName").value = selectedDevice()?.name || "";
      renderDevices();
      updateHero();
      refreshMessages();
      refreshTelemetry();
    };
  });
}

async function renameDevice() {
  const device = selectedDevice();
  if (!device) return;
  try {
    await api(`/api/v1/devices/${encodeURIComponent(device.id)}`,
      {
        method: "PATCH",
        headers: userHeaders(),
        body: JSON.stringify({ name: $("renameDeviceName").value.trim() })
      });
    await refreshDevices();
  } catch (err) {
    renderError("devicesList", err.message);
  }
}

async function deleteDevice() {
  const device = selectedDevice();
  if (!device) return;
  if (!confirm(`删除设备 ${device.name || device.id}？`)) return;
  try {
    await api(`/api/v1/devices/${encodeURIComponent(device.id)}`, { method: "DELETE", headers: userHeaders() });
    state.selectedDeviceID = "";
    await refreshDevices();
  } catch (err) {
    renderError("devicesList", err.message);
  }
}

async function sendMessage() {
  const device = selectedDevice();
  if (!device) return alert("请先选择设备");
  try {
    await api(`/api/v1/devices/${encodeURIComponent(device.id)}/messages`, {
      method: "POST",
      headers: userHeaders(),
      body: JSON.stringify({
        body: $("messageBody").value.trim(),
        priority: $("messagePriority").value
      })
    });
    $("messageBody").value = "";
    await refreshMessages();
  } catch (err) {
    renderError("messageHistory", err.message);
  }
}

async function refreshMessages() {
  const device = selectedDevice();
  if (!device) return renderEmpty("messageHistory", "请选择设备。");
  try {
    const data = await api(`/api/v1/devices/${encodeURIComponent(device.id)}/messages?limit=50`, { headers: userHeaders() });
    renderMessages("messageHistory", data);
  } catch (err) {
    renderError("messageHistory", err.message);
  }
}

function renderMessages(id, messages) {
  const box = $(id);
  if (!messages || !messages.length) return renderEmpty(id, "暂无消息。");
  box.innerHTML = messages.map((m) => `
      <div class="item">
        <div class="item-row">
          <div class="item-title">${escapeHTML(m.body)}</div>
          <span class="pill">${escapeHTML(m.priority || "normal")}</span>
        </div>
        <div class="muted">#${m.id} ${escapeHTML(m.status || "pending")} · ${formatDate(m.created_at)}</div>
      </div>
    `).join("");
}

async function createTodo() {
  try {
    await api("/api/v1/todos", {
      method: "POST",
      headers: userHeaders(),
      body: JSON.stringify({ text: $("todoText").value.trim(), status: Number($("todoStatus").value) })
    });
    $("todoText").value = "";
    await refreshTodos();
  } catch (err) {
    renderError("todosList", err.message);
  }
}

async function refreshTodos() {
  try {
    if (!$("userToken").value.trim()) return;
    state.todos = await api("/api/v1/todos", { headers: userHeaders() });
    $("metricTodos").textContent = state.todos.length;
    renderTodos();
  } catch (err) {
    renderError("todosList", err.message);
  }
}

function renderTodos() {
  const box = $("todosList");
  if (!state.todos.length) return renderEmpty("todosList", "暂无 TODO。");
  box.innerHTML = state.todos.map((t) => `
      <div class="item">
        <div class="item-row">
          <div>
            <div class="item-title">${escapeHTML(t.text)}</div>
            <div class="muted">version ${t.version}</div>
          </div>
          <select data-todo-status="${t.id}">
            <option value="0" ${t.status === 0 ? "selected" : ""}>未完成</option>
            <option value="1" ${t.status === 1 ? "selected" : ""}>正在完成</option>
            <option value="2" ${t.status === 2 ? "selected" : ""}>已完成</option>
          </select>
        </div>
        <div class="actions">
          <button class="secondary" data-save-todo="${t.id}" data-version="${t.version}">保存状态</button>
          <button class="danger" data-delete-todo="${t.id}" data-version="${t.version}">删除</button>
        </div>
      </div>
    `).join("");
  box.querySelectorAll("[data-save-todo]").forEach((btn) => {
    btn.onclick = () => updateTodo(Number(btn.dataset.saveTodo), Number(btn.dataset.version));
  });
  box.querySelectorAll("[data-delete-todo]").forEach((btn) => {
    btn.onclick = () => deleteTodo(Number(btn.dataset.deleteTodo), Number(btn.dataset.version));
  });
}

async function updateTodo(id, version) {
  const status = Number(document.querySelector(`[data-todo-status="${id}"]`).value);
  try {
    await api(`/api/v1/todos/${id}`, {
      method: "PATCH",
      headers: userHeaders(),
      body: JSON.stringify({ version, status })
    });
    await refreshTodos();
  } catch (err) {
    renderError("todosList", err.message);
  }
}

async function deleteTodo(id, version) {
  try {
    await api(`/api/v1/todos/${id}`, {
      method: "DELETE",
      headers: userHeaders(),
      body: JSON.stringify({ version })
    });
    await refreshTodos();
  } catch (err) {
    renderError("todosList", err.message);
  }
}

async function refreshWeather() {
  try {
    const data = await api("/api/v1/weather", { headers: userHeaders() });
    $("weatherBox").innerHTML = `
        <div class="item">
          <div class="item-row">
            <div>
              <div class="item-title">${escapeHTML(data.condition || "unknown")}</div>
              <div class="muted">${escapeHTML(data.location || "")}</div>
            </div>
            <strong style="font-size:32px">${Number(data.temperature || 0).toFixed(0)}°</strong>
          </div>
          <div class="muted">humidity ${data.humidity ?? "-"} · updated ${formatDate(data.updated_at)}</div>
        </div>
      `;
  } catch (err) {
    renderError("weatherBox", err.message);
  }
}

async function refreshSnapshot() {
  try {
    const data = await api("/api/v1/snapshot?include=weather,messages,todos,telemetry", { headers: userHeaders() });
    $("snapshotRaw").textContent = JSON.stringify(data, null, 2);
  } catch (err) {
    $("snapshotRaw").textContent = err.message;
  }
}

async function refreshTelemetry() {
  const device = selectedDevice();
  if (!device) return renderEmpty("telemetryList", "请选择设备。");
  try {
    const data = await api(`/api/v1/devices/${encodeURIComponent(device.id)}/telemetry?limit=20`, { headers: userHeaders() });
    state.telemetry = data;
    if (!data.length) return renderEmpty("telemetryList", "暂无遥测。");
    $("telemetryList").innerHTML = data.map((t, index) => `
        <div class="item">
          <div class="item-row">
            <div class="item-title">#${t.id} seq ${t.sequence}</div>
            <div class="item-actions">
              <span class="pill">${formatDate(t.received_at)}</span>
              <button class="icon-button" title="查看详情" data-telemetry-index="${index}">i</button>
            </div>
          </div>
          <div class="muted">battery ${t.power?.battery?.percentage ?? "-"}% · temp ${t.environment?.shtc3?.temperature_c ?? "-"}°C</div>
        </div>
      `).join("");
    $("telemetryList").querySelectorAll("[data-telemetry-index]").forEach((btn) => {
      btn.onclick = () => openTelemetryModal(Number(btn.dataset.telemetryIndex));
    });
  } catch (err) {
    renderError("telemetryList", err.message);
  }
}

function openTelemetryModal(index) {
  const payload = state.telemetry[index] ?? {};
  $("telemetryRaw").textContent = JSON.stringify(payload, null, 2);
  $("telemetryModal").classList.add("active");
  $("telemetryModal").setAttribute("aria-hidden", "false");
}

function closeTelemetryModal() {
  $("telemetryModal").classList.remove("active");
  $("telemetryModal").setAttribute("aria-hidden", "true");
}

function selectedDevice() {
  return state.devices.find((d) => d.id === state.selectedDeviceID) || null;
}

function updateHero() {
  const device = selectedDevice();
  $("metricDevices").textContent = state.devices.length;
  $("metricPending").textContent = "0";
  $("metricTodos").textContent = state.todos.length;
  $("heroTitle").textContent = device?.name || device?.id || "未选择设备";
  $("heroDeviceID").textContent = "device: " + (device?.id || "-");
  $("heroBound").textContent = device ? "已绑定" : "未绑定";
  $("heroUser").textContent = "user: " + (state.user?.name || state.user?.id || "-");
  if (device) $("renameDeviceName").value = device.name || "";
}

function renderEmpty(id, text) {
  $(id).innerHTML = `<div class="empty">${escapeHTML(text)}</div>`;
}

function renderError(id, text) {
  $(id).innerHTML = `<div class="notice error">${escapeHTML(text)}</div>`;
}

function notice(id, text, type) {
  const el = $(id);
  el.className = "notice " + (type || "");
  el.textContent = text;
}

function log(value) {
  $("rawLog").textContent = JSON.stringify(value, null, 2);
}

function formatDate(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (ch) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;"
  }[ch]));
}

function escapeAttr(value) {
  return escapeHTML(value).replace(/`/g, "&#96;");
}

init();
