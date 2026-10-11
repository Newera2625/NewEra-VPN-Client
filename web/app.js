(function () {
  'use strict';
  var el = function (id) { return document.getElementById(id); };
  var state = { user: null, connected: false, nodes: [], nodeId: null };

  function api(path, opts) {
    opts = opts || {};
    var o = { method: opts.method || 'GET', headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin' };
    if (opts.body) o.body = JSON.stringify(opts.body);
    return fetch(path, o).then(function (r) { return r.json(); });
  }

  function avatarURL(a) {
    if (!a) return '';
    if (/^https?:\/\//i.test(a) || a.indexOf('//') === 0) return a;
    return 'https://sso.newera2625.top' + (a.charAt(0) === '/' ? a : '/' + a);
  }

  function renderUser() {
    var box = el('user-box');
    var av = el('avatar');
    if (state.user) {
      var url = avatarURL(state.user.avatar);
      if (url) { av.innerHTML = '<img src="' + url + '" alt="">'; }
      else { av.textContent = (state.user.nickname || state.user.username || '?').charAt(0); }
      el('uname').textContent = state.user.nickname || state.user.username;
      el('login-btn').style.display = 'none';
      el('logout-btn').style.display = '';
    } else {
      av.textContent = '';
      el('uname').textContent = '未登录';
      el('login-btn').style.display = '';
      el('logout-btn').style.display = 'none';
    }
  }

  function loadStatus() {
    api('/api/status').then(function (d) {
      state.user = d.logged_in ? d.user : null;
      state.connected = !!d.connected;
      el('ver').textContent = 'v' + d.version;
      el('about-ver').textContent = d.version;
      renderUser();
      renderConn();
      if (d.logged_in) loadNodes();
    });
  }

  function loadNodes() {
    el('node-list').innerHTML = '<div class="empty">加载中...</div>';
    api('/api/nodes').then(function (d) {
      if (d.code !== 200) { el('node-list').innerHTML = '<div class="empty">节点服务异常</div>'; return; }
      state.nodes = d.data.nodes || [];
      renderNodes();
    }).catch(function () { el('node-list').innerHTML = '<div class="empty">加载失败</div>'; });
  }

  function renderNodes() {
    var list = el('node-list');
    if (!state.nodes.length) { list.innerHTML = '<div class="empty">暂无可用节点，请稍后刷新</div>'; return; }
    list.innerHTML = state.nodes.map(function (n) {
      var latCls = n.latency < 100 ? ' fast' : '';
      var isConn = state.connected && state.nodeId === n.id;
      return '<div class="node-item" data-id="' + n.id + '">' +
        '<div class="node-main">' +
          '<div class="node-flag">' + (n.country_code || n.country.slice(0, 1)) + '</div>' +
          '<div class="node-info"><div class="n-country">' + n.country + '</div>' +
          '<div class="n-meta">' + (n.protocol || 'http').toUpperCase() + ' · 加密隧道 · 自动测速</div></div>' +
        '</div>' +
        '<div class="node-right">' +
          '<span class="lat' + latCls + '">' + n.latency + ' ms</span>' +
          '<button class="connect-btn' + (isConn ? ' connected' : '') + '" data-id="' + n.id + '">' + (isConn ? '断开' : '连接') + '</button>' +
        '</div>' +
      '</div>';
    }).join('');
  }

  function renderConn() {
    var badge = el('conn-badge');
    var detail = el('conn-detail');
    var dis = el('disconnect-btn');
    if (state.connected) {
      badge.textContent = '已连接';
      badge.classList.add('on');
      detail.textContent = '正在通过节点 "…" 加速网络\n当前系统代理已开启';
      dis.style.display = '';
    } else {
      badge.textContent = '未连接';
      badge.classList.remove('on');
      detail.textContent = '未连接任何节点';
      dis.style.display = 'none';
    }
  }

  el('node-list').addEventListener('click', function (e) {
    var btn = e.target.closest('.connect-btn');
    if (!btn) return;
    var id = btn.getAttribute('data-id');
    if (state.connected && state.nodeId === id) {
      api('/api/disconnect', { method: 'POST', body: {} }).then(function () {
        state.connected = false; state.nodeId = null; renderConn(); renderNodes();
      });
    } else {
      api('/api/connect', { method: 'POST', body: { id: id } }).then(function (d) {
        if (d.code === 200) { state.connected = true; state.nodeId = id; }
        else { alert(d.message || '连接失败'); }
        renderConn(); renderNodes();
      });
    }
  });

  el('login-btn').addEventListener('click', function () {
    window.location.href = '/api/login';
  });
  el('logout-btn').addEventListener('click', function () {
    api('/api/logout', { method: 'POST', body: {} }).then(function () {
      state.user = null; renderUser(); el('node-list').innerHTML = '<div class="empty">请先登录</div>';
    });
  });
  el('refresh-nodes').addEventListener('click', function () { loadNodes(); });
  el('disconnect-btn').addEventListener('click', function () {
    api('/api/disconnect', { method: 'POST', body: {} }).then(function () {
      state.connected = false; state.nodeId = null; renderConn(); renderNodes();
    });
  });

  // 更新
  function checkUpdate() {
    api('/api/check-update').then(function (d) {
      var st = el('update-state');
      if (d.code === 200 && d.data && d.data.update) {
        st.textContent = '发现新版本 ' + d.data.version;
        el('update-btn').textContent = '立即更新';
        el('update-btn').onclick = function () {
          api('/api/do-update', { method: 'POST', body: {} }).then(function (r) {
            st.textContent = r.message || '更新中...';
            setTimeout(function () { st.textContent = '更新完成后客户端将自动重启'; }, 1500);
          });
        };
      } else {
        st.textContent = '当前已是最新版本';
        el('update-btn').textContent = '检查更新';
        el('update-btn').onclick = checkUpdate;
      }
    }).catch(function () { el('update-state').textContent = '更新服务不可用'; });
  }
  el('update-btn').addEventListener('click', checkUpdate);

  // Tab
  el('tab-nodes').addEventListener('click', function () {
    el('panel-nodes').style.display = '';
    el('panel-settings').style.display = 'none';
    this.classList.add('active'); el('tab-settings').classList.remove('active');
  });
  el('tab-settings').addEventListener('click', function () {
    el('panel-nodes').style.display = 'none';
    el('panel-settings').style.display = '';
    this.classList.add('active'); el('tab-nodes').classList.remove('active');
    checkUpdate();
  });

  loadStatus();
  setInterval(function () { if (state.user) loadNodes(); }, 60000);
})();
