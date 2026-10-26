/* FolioPath management-center interaction prototype */
(function () {
  'use strict';

  const STATE_KEY = 'foliopath-management-prototype-v1';
  const DEFAULT_STATE = {
    general: {
      theme: 'system',
      language: 'browser',
      layout: 'grid',
      previewPinned: true
    },
    scan: {
      scheduled: true,
      interval: 24
    },
    cache: {
      quota: 10,
      used: 6.4
    },
    operations: [
      {
        id: 'thumbnail-missing',
        title: '补齐缺失缩略图',
        description: '为已索引但缺少缩略图的图片生成派生缓存。',
        status: 'waiting',
        progress: 0,
        completed: 0,
        total: 38,
        updatedAt: '等待扫描任务释放资源'
      },
      {
        id: 'video-posters',
        title: '生成视频封面',
        description: '为缺少预览封面的视频提取安全的静态画面。',
        status: 'running',
        progress: 41,
        completed: 86,
        total: 210,
        updatedAt: '刚刚'
      },
      {
        id: 'cache-cleanup',
        title: '缓存水位清理',
        description: '按最近最少使用顺序清理可重建缓存并保留安全余量。',
        status: 'success',
        progress: 100,
        completed: 1240,
        total: 1240,
        updatedAt: '今天 03:12'
      }
    ],
    maintenance: {
      backup: {
        enabled: true,
        schedule: 'daily-0300',
        retention: 7
      },
      backups: [
        { id: 'backup-1', createdAt: '2026-07-30 03:00', size: '18.6 MiB', status: 'ready' },
        { id: 'backup-2', createdAt: '2026-07-29 03:00', size: '18.4 MiB', status: 'ready' }
      ],
      checks: [
        {
          id: 'missing',
          title: '索引中的缺失文件',
          description: '确认索引记录的媒体是否仍能从只读媒体库访问。',
          status: 'attention',
          result: '38 项待复核',
          updatedAt: '今天 02:42'
        },
        {
          id: 'untracked',
          title: '未跟踪的新文件',
          description: '查找媒体库中尚未进入可靠索引的受支持媒体。',
          status: 'healthy',
          result: '没有发现问题',
          updatedAt: '今天 02:44'
        },
        {
          id: 'derived',
          title: '派生缓存完整性',
          description: '检查缩略图、视频封面与源指纹是否匹配。',
          status: 'attention',
          result: '124 个缓存可补齐',
          updatedAt: '昨天 23:18'
        }
      ]
    },
    account: {
      displayName: '管理员',
      username: 'admin'
    },
    libraries: [
      {
        id: 'family',
        name: '家庭影像',
        path: '家庭影像',
        status: 'ready',
        itemCount: 12480,
        directoryCount: 684,
        lastScan: '2026-07-29 22:18',
        progress: 100,
        issue: ''
      },
      {
        id: 'travel',
        name: '旅行摄影',
        path: '旅行',
        status: 'scanning',
        itemCount: 5320,
        directoryCount: 312,
        lastScan: '正在扫描',
        progress: 64,
        issue: ''
      },
      {
        id: 'work',
        name: '工作素材',
        path: '工作',
        status: 'offline',
        itemCount: 3210,
        directoryCount: 146,
        lastScan: '2026-07-27 09:42',
        progress: 0,
        issue: '根目录当前不可读；最后可靠索引已保留。'
      }
    ],
    recentJobs: [
      { id: 'job-family', libraryId: 'family', result: 'success', finishedAt: '2026-07-29 22:18', summary: '新增 214 项，跳过 38 项' },
      { id: 'job-work', libraryId: 'work', result: 'offline', finishedAt: '2026-07-30 08:10', summary: '根目录不可读，未清理旧索引' }
    ]
  };

  const PATH_OPTIONS = [
    { path: '家庭影像', label: '家庭影像', note: '已被媒体库“家庭影像”使用', disabled: true },
    { path: '旅行', label: '旅行', note: '已被媒体库“旅行摄影”使用', disabled: true },
    { path: '工作', label: '工作', note: '目录当前不可读', disabled: true },
    { path: '活动', label: '活动', note: '12 个子目录', disabled: false },
    { path: '老照片', label: '老照片', note: '8 个子目录', disabled: false },
    { path: '共享/相册', label: '共享 / 相册', note: '符号链接目录，不可选择', disabled: true }
  ];

  const NAV_ITEMS = [
    {
      id: 'general',
      label: '通用',
      href: '07-settings-general.html',
      icon: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 00.3 1.8l.1.1a2 2 0 01-2.8 2.8l-.1-.1a1.7 1.7 0 00-1.8-.3 1.7 1.7 0 00-1 1.5V21a2 2 0 01-4 0v-.2a1.7 1.7 0 00-1-1.5 1.7 1.7 0 00-1.8.3l-.1.1a2 2 0 01-2.8-2.8l.1-.1a1.7 1.7 0 00.3-1.8 1.7 1.7 0 00-1.5-1H3a2 2 0 010-4h.2a1.7 1.7 0 001.5-1 1.7 1.7 0 00-.3-1.8l-.1-.1a2 2 0 012.8-2.8l.1.1a1.7 1.7 0 001.8.3 1.7 1.7 0 001-1.5V3a2 2 0 014 0v.2a1.7 1.7 0 001 1.5 1.7 1.7 0 001.8-.3l.1-.1a2 2 0 012.8 2.8l-.1.1a1.7 1.7 0 00-.3 1.8 1.7 1.7 0 001.5 1h.2a2 2 0 010 4h-.2a1.7 1.7 0 00-1.5 1z"/>'
    },
    {
      id: 'libraries',
      label: '媒体库',
      href: '06-settings-libraries.html',
      icon: '<path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/>'
    },
    {
      id: 'storage',
      label: '扫描与缓存',
      href: '08-settings-storage.html',
      icon: '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v6c0 1.7 4 3 9 3s9-1.3 9-3V5"/><path d="M3 11v6c0 1.7 4 3 9 3s9-1.3 9-3v-6"/>'
    },
    {
      id: 'intelligence',
      label: '智能功能',
      href: '13-settings-ai.html',
      icon: '<path d="M12 3v3M12 18v3M3 12h3M18 12h3"/><circle cx="12" cy="12" r="5"/><path d="M8.5 5.5 6.4 3.4M17.6 20.6l-2.1-2.1M18.5 5.5l2.1-2.1M3.4 20.6l2.1-2.1"/>'
    },
    {
      id: 'maintenance',
      label: '系统维护',
      href: '10-settings-maintenance.html',
      icon: '<path d="M14.7 6.3a4 4 0 00-5 5L3 18l3 3 6.7-6.7a4 4 0 005-5l-2.4 2.4-3-3z"/>'
    },
    {
      id: 'account',
      label: '账户',
      href: '09-settings-account.html',
      icon: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0116 0"/>'
    }
  ];

  let state = loadState();
  let toastTimer;
  let scanTimer;
  let operationTimer;

  function clone(value) {
    return JSON.parse(JSON.stringify(value));
  }

  function mergeState(saved) {
    const next = clone(DEFAULT_STATE);
    if (!saved || typeof saved !== 'object') return next;
    next.general = Object.assign(next.general, saved.general || {});
    next.scan = Object.assign(next.scan, saved.scan || {});
    next.cache = Object.assign(next.cache, saved.cache || {});
    next.account = Object.assign(next.account, saved.account || {});
    next.maintenance = Object.assign(next.maintenance, saved.maintenance || {});
    next.maintenance.backup = Object.assign(
      clone(DEFAULT_STATE.maintenance.backup),
      saved.maintenance && saved.maintenance.backup || {}
    );
    if (saved.maintenance && Array.isArray(saved.maintenance.backups)) {
      next.maintenance.backups = saved.maintenance.backups;
    }
    if (saved.maintenance && Array.isArray(saved.maintenance.checks)) {
      next.maintenance.checks = saved.maintenance.checks;
    }
    if (Array.isArray(saved.libraries)) next.libraries = saved.libraries;
    if (Array.isArray(saved.recentJobs)) next.recentJobs = saved.recentJobs;
    if (Array.isArray(saved.operations)) next.operations = saved.operations;
    return next;
  }

  function loadState() {
    try {
      const next = mergeState(JSON.parse(localStorage.getItem(STATE_KEY)));
      if (!localStorage.getItem(STATE_KEY)) {
        const theme = localStorage.getItem('foliopath-theme');
        if (theme === 'system' || theme === 'light' || theme === 'dark') next.general.theme = theme;
      }
      return next;
    } catch (error) {
      return clone(DEFAULT_STATE);
    }
  }

  function saveState() {
    localStorage.setItem(STATE_KEY, JSON.stringify(state));
  }

  function formatNumber(value) {
    return new Intl.NumberFormat('zh-CN').format(value);
  }

  function escapeHtml(value) {
    return String(value)
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#039;');
  }

  function showToast(message) {
    const toast = document.querySelector('[data-toast]');
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.hidden = false;
    toastTimer = window.setTimeout(function () {
      toast.hidden = true;
    }, 4200);
  }

  function renderNavigation() {
    const sidebar = document.querySelector('[data-settings-sidebar]');
    if (!sidebar) return;
    const active = sidebar.dataset.active;
    sidebar.innerHTML = `
      <h2>管理中心</h2>
      <nav aria-label="管理中心导航">
        <ul class="settings-nav-list">
          ${NAV_ITEMS.map(function (item) {
            const current = item.id === active;
            return `
              <li>
                <a href="${item.href}" class="settings-nav-item${current ? ' active' : ''}"${current ? ' aria-current="page"' : ''}>
                  <span class="settings-nav-icon" aria-hidden="true">
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">${item.icon}</svg>
                  </span>
                  ${item.label}
                </a>
              </li>`;
          }).join('')}
        </ul>
      </nav>`;
  }

  function updateHeaderAccount() {
    const label = document.querySelector('.admin-label');
    const avatar = document.querySelector('.admin-avatar');
    const menuName = document.querySelector('.account-menu-header strong');
    if (label) label.textContent = state.account.displayName;
    if (avatar) avatar.textContent = state.account.displayName.slice(0, 1);
    if (menuName) menuName.textContent = state.account.displayName;
  }

  function openDialog(name, focusSelector) {
    const dialog = document.querySelector(`[data-dialog="${name}"]`);
    if (!dialog) return;
    dialog.hidden = false;
    document.body.classList.add('dialog-open');
    const focusTarget = focusSelector ? dialog.querySelector(focusSelector) : dialog.querySelector('input, button, select');
    if (focusTarget) window.setTimeout(function () { focusTarget.focus(); }, 0);
  }

  function closeDialog(dialog) {
    const target = typeof dialog === 'string'
      ? document.querySelector(`[data-dialog="${dialog}"]`)
      : dialog;
    if (!target) return;
    target.hidden = true;
    document.body.classList.remove('dialog-open');
    target.querySelectorAll('.field-error').forEach(function (error) {
      error.hidden = true;
      error.textContent = '';
    });
  }

  function bindDialogs() {
    document.querySelectorAll('[data-close-dialog]').forEach(function (button) {
      button.addEventListener('click', function () {
        closeDialog(button.closest('[data-dialog]'));
      });
    });

    document.querySelectorAll('[data-dialog]').forEach(function (scrim) {
      scrim.addEventListener('click', function (event) {
        if (event.target === scrim) closeDialog(scrim);
      });
    });

    document.addEventListener('keydown', function (event) {
      if (event.key !== 'Escape') return;
      const visible = document.querySelector('[data-dialog]:not([hidden])');
      if (visible) closeDialog(visible);
      closeAllMenus();
    });
  }

  function setError(form, key, message) {
    const error = form.querySelector(`[data-error-for="${key}"]`);
    if (!error) return;
    error.textContent = message;
    error.hidden = !message;
  }

  function serializeGeneral(form) {
    return {
      theme: form.elements.theme.value,
      language: form.elements.language.value,
      layout: form.elements.layout.value,
      previewPinned: form.elements.previewPinned.checked
    };
  }

  function generalEqual(a, b) {
    return a.theme === b.theme &&
      a.language === b.language &&
      a.layout === b.layout &&
      a.previewPinned === b.previewPinned;
  }

  function applyLanguagePreference(preference) {
    const resolved = preference === 'browser'
      ? (navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
      : preference;
    document.documentElement.lang = resolved;
  }

  function initGeneral() {
    const form = document.querySelector('[data-general-form]');
    if (!form) return;
    const saveButton = form.querySelector('[data-save-general]');
    const stateLabel = form.querySelector('[data-save-state]');

    function fill() {
      form.elements.theme.value = state.general.theme;
      form.elements.language.value = state.general.language;
      form.elements.layout.value = state.general.layout;
      form.elements.previewPinned.checked = state.general.previewPinned;
      updateDirty();
    }

    function updateDirty() {
      const dirty = !generalEqual(serializeGeneral(form), state.general);
      saveButton.disabled = !dirty;
      stateLabel.textContent = dirty ? '有未保存的更改' : '没有未保存的更改';
      stateLabel.classList.toggle('dirty', dirty);
    }

    form.addEventListener('input', function () {
      if (form.elements.theme.value && window.previewThemePreference) {
        window.previewThemePreference(form.elements.theme.value);
      }
      applyLanguagePreference(form.elements.language.value);
      updateDirty();
    });

    form.addEventListener('submit', function (event) {
      event.preventDefault();
      state.general = serializeGeneral(form);
      saveState();
      if (window.setThemePreference) window.setThemePreference(state.general.theme);
      applyLanguagePreference(state.general.language);
      updateDirty();
      showToast('通用设置已保存');
    });

    form.querySelector('[data-reset-general]').addEventListener('click', function () {
      fill();
      if (window.previewThemePreference) window.previewThemePreference(state.general.theme);
      applyLanguagePreference(state.general.language);
    });

    fill();
  }

  function statusLabel(status) {
    return {
      ready: '就绪',
      scanning: '扫描中',
      running: '运行中',
      waiting: '等待中',
      success: '已完成',
      healthy: '正常',
      attention: '需处理',
      creating: '创建中',
      offline: '离线',
      failed: '扫描失败',
      cancelled: '已取消'
    }[status] || status;
  }

  function operationStatusClass(status) {
    if (status === 'ready' || status === 'success' || status === 'healthy') return 'ready';
    if (status === 'running' || status === 'scanning' || status === 'creating') return 'scanning';
    if (status === 'waiting') return 'waiting';
    if (status === 'attention') return 'failed';
    return status;
  }

  function currentTimestamp() {
    return new Intl.DateTimeFormat('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    }).format(new Date()).replaceAll('/', '-');
  }

  function closeAllMenus() {
    document.querySelectorAll('[data-library-menu]').forEach(function (menu) {
      menu.hidden = true;
    });
    document.querySelectorAll('[data-library-menu-trigger]').forEach(function (button) {
      button.setAttribute('aria-expanded', 'false');
    });
  }

  function renderLibraries() {
    const list = document.querySelector('[data-library-list]');
    const summary = document.querySelector('[data-library-summary]');
    if (!list) return;
    const scanning = state.libraries.filter(function (library) { return library.status === 'scanning'; }).length;
    const offline = state.libraries.filter(function (library) { return library.status === 'offline'; }).length;
    if (summary) {
      summary.textContent = `${state.libraries.length} 个媒体库 · ${scanning} 个扫描中 · ${offline} 个需处理`;
    }

    if (!state.libraries.length) {
      list.innerHTML = `
        <div class="settings-empty">
          <strong>还没有媒体库</strong>
          <span>创建媒体库后，FolioPath 会立即开始第一次完整扫描。</span>
          <button class="btn btn-primary" type="button" data-empty-add-library>新建媒体库</button>
        </div>`;
      list.querySelector('[data-empty-add-library]').addEventListener('click', prepareAddLibrary);
      return;
    }

    list.innerHTML = state.libraries.map(function (library) {
      const safeName = escapeHtml(library.name);
      const safePath = escapeHtml(library.path);
      const safeIssue = escapeHtml(library.issue);
      const scanningMarkup = library.status === 'scanning'
        ? `<div class="scan-progress" aria-label="扫描进度 ${library.progress}%"><span style="width:${library.progress}%"></span></div>
           <span class="library-progress-copy">正在扫描 · ${library.progress}% · 已发现 ${formatNumber(library.itemCount)} 项</span>`
        : '';
      const issueMarkup = library.issue
        ? `<div class="library-issue">${safeIssue}</div>`
        : '';
      return `
        <article class="library-management-card" data-library-id="${library.id}">
          <div class="library-card-main">
            <div class="library-card-heading">
              <div>
                <span class="library-status ${library.status}"><span class="status-dot" aria-hidden="true"></span>${statusLabel(library.status)}</span>
                <h3>${safeName}</h3>
              </div>
              <div class="library-action-menu">
                <button class="btn btn-secondary btn-sm library-menu-trigger" type="button" aria-label="${safeName}的更多操作" aria-haspopup="menu" aria-expanded="false" data-library-menu-trigger="${library.id}">操作</button>
                <div class="library-menu" role="menu" hidden data-library-menu="${library.id}">
                  <button type="button" role="menuitem" data-library-action="rename">重命名</button>
                  ${library.status === 'scanning'
                    ? '<button type="button" role="menuitem" data-library-action="cancel">取消扫描</button>'
                    : '<button type="button" role="menuitem" data-library-action="scan">重新扫描</button>'}
                  ${library.status === 'offline' || library.status === 'failed'
                    ? '<button type="button" role="menuitem" data-library-action="retry">重新检查连接</button>'
                    : ''}
                  <a href="08-settings-storage.html" role="menuitem">查看扫描详情</a>
                  <button class="danger" type="button" role="menuitem" data-library-action="remove">从 FolioPath 移除</button>
                </div>
              </div>
            </div>
            <dl class="library-metadata">
              <div><dt>根目录</dt><dd>/library/${safePath}</dd></div>
              <div><dt>索引</dt><dd>${formatNumber(library.itemCount)} 项 · ${formatNumber(library.directoryCount)} 个目录</dd></div>
              <div><dt>最近扫描</dt><dd>${library.lastScan}</dd></div>
            </dl>
            ${scanningMarkup}
            ${issueMarkup}
          </div>
        </article>`;
    }).join('');

    list.querySelectorAll('[data-library-menu-trigger]').forEach(function (button) {
      button.addEventListener('click', function () {
        const menu = list.querySelector(`[data-library-menu="${button.dataset.libraryMenuTrigger}"]`);
        const shouldOpen = menu.hidden;
        closeAllMenus();
        menu.hidden = !shouldOpen;
        button.setAttribute('aria-expanded', String(shouldOpen));
      });
    });

    list.querySelectorAll('[data-library-action]').forEach(function (button) {
      button.addEventListener('click', function () {
        const card = button.closest('[data-library-id]');
        handleLibraryAction(card.dataset.libraryId, button.dataset.libraryAction);
      });
    });
  }

  function renderPathOptions() {
    const container = document.querySelector('[data-path-options]');
    if (!container) return;
    const usedPaths = new Set(state.libraries.map(function (library) { return library.path; }));
    container.innerHTML = PATH_OPTIONS.map(function (option) {
      const disabled = option.disabled || usedPaths.has(option.path);
      const note = usedPaths.has(option.path)
        ? `已被媒体库“${state.libraries.find(function (library) { return library.path === option.path; }).name}”使用`
        : option.note;
      const safeLabel = escapeHtml(option.label);
      const safeNote = escapeHtml(note);
      const safePath = escapeHtml(option.path);
      return `
        <button class="path-option" type="button" data-path="${safePath}"${disabled ? ' disabled' : ''}>
          <span><strong>${safeLabel}</strong><small>${safeNote}</small></span>
        </button>`;
    }).join('');

    container.querySelectorAll('.path-option:not(:disabled)').forEach(function (button) {
      button.addEventListener('click', function () {
        container.querySelectorAll('.path-option').forEach(function (option) {
          option.classList.remove('selected');
        });
        button.classList.add('selected');
        const form = document.querySelector('[data-add-library-form]');
        form.elements.path.value = button.dataset.path;
        document.querySelector('[data-selected-path]').textContent = `/ ${button.dataset.path}`;
        setError(form, 'path', '');
      });
    });
  }

  function prepareAddLibrary() {
    const form = document.querySelector('[data-add-library-form]');
    form.reset();
    document.querySelector('[data-selected-path]').textContent = '尚未选择';
    renderPathOptions();
    openDialog('add-library', 'input[name="name"]');
  }

  function startScan(library) {
    if (library.status === 'scanning') {
      showToast('扫描已在进行，没有启动重复任务');
      return;
    }
    library.status = 'scanning';
    library.progress = 8;
    library.lastScan = '正在扫描';
    library.issue = '';
    saveState();
    renderLibraries();
    renderScanTasks();
    renderOperationTasks();
    renderStorageHealth();
    ensureScanTimer();
    showToast(`已开始扫描“${library.name}”`);
  }

  function cancelScan(library) {
    library.status = 'cancelled';
    library.lastScan = '扫描已取消 · 保留最后可靠索引';
    library.progress = 0;
    saveState();
    renderLibraries();
    renderScanTasks();
    renderOperationTasks();
    renderStorageHealth();
    showToast(`已取消“${library.name}”的扫描`);
  }

  function retryLibrary(library) {
    library.status = 'ready';
    library.issue = '';
    library.lastScan = '刚刚重新检查 · 连接已恢复';
    saveState();
    renderLibraries();
    renderScanTasks();
    renderOperationTasks();
    renderStorageHealth();
    showToast(`“${library.name}”已恢复连接`);
  }

  function handleLibraryAction(id, action) {
    closeAllMenus();
    const library = state.libraries.find(function (item) { return item.id === id; });
    if (!library) return;
    if (action === 'scan') startScan(library);
    if (action === 'cancel') cancelScan(library);
    if (action === 'retry') retryLibrary(library);
    if (action === 'rename') {
      const form = document.querySelector('[data-rename-library-form]');
      form.elements.id.value = id;
      form.elements.name.value = library.name;
      openDialog('rename-library', 'input[name="name"]');
    }
    if (action === 'remove') {
      const form = document.querySelector('[data-remove-library-form]');
      form.reset();
      form.elements.id.value = id;
      document.querySelector('[data-remove-library-copy]').textContent =
        `“${library.name}”使用 /library/${library.path}。移除后 FolioPath 将不再显示它。`;
      openDialog('remove-library');
    }
  }

  function ensureScanTimer() {
    window.clearInterval(scanTimer);
    if (!state.libraries.some(function (library) { return library.status === 'scanning'; })) return;
    scanTimer = window.setInterval(function () {
      let changed = false;
      state.libraries.forEach(function (library) {
        if (library.status !== 'scanning') return;
        library.progress = Math.min(100, library.progress + 4);
        library.itemCount += 27;
        changed = true;
        if (library.progress >= 100) {
          library.status = 'ready';
          library.lastScan = '刚刚完成';
          state.recentJobs.unshift({
            id: `job-${library.id}-${Date.now()}`,
            libraryId: library.id,
            result: 'success',
            finishedAt: '刚刚',
            summary: `索引 ${formatNumber(library.itemCount)} 项`
          });
        }
      });
      if (!changed) return;
      saveState();
      renderLibraries();
      renderScanTasks();
      renderOperationTasks();
      renderStorageHealth();
      renderTaskDetail();
      if (!state.libraries.some(function (library) { return library.status === 'scanning'; })) {
        window.clearInterval(scanTimer);
        showToast('扫描任务已完成');
      }
    }, 1500);
  }

  function initLibraries() {
    const openButton = document.querySelector('[data-open-add-library]');
    if (!openButton) return;
    openButton.addEventListener('click', prepareAddLibrary);
    renderLibraries();

    const addForm = document.querySelector('[data-add-library-form]');
    addForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const name = addForm.elements.name.value.trim();
      const path = addForm.elements.path.value;
      const duplicate = state.libraries.some(function (library) {
        return library.name.toLocaleLowerCase() === name.toLocaleLowerCase();
      });
      setError(addForm, 'name', !name ? '请输入媒体库名称。' : duplicate ? '名称已被使用，请换一个名称。' : '');
      setError(addForm, 'path', path ? '' : '请选择一个可用目录。');
      if (!name || !path || duplicate) return;
      state.libraries.push({
        id: `library-${Date.now()}`,
        name: name,
        path: path,
        status: 'scanning',
        itemCount: 0,
        directoryCount: 0,
        lastScan: '正在进行首次扫描',
        progress: 4,
        issue: ''
      });
      saveState();
      closeDialog('add-library');
      renderLibraries();
      ensureScanTimer();
      showToast(`已创建“${name}”，首次扫描已开始`);
    });

    const renameForm = document.querySelector('[data-rename-library-form]');
    renameForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const id = renameForm.elements.id.value;
      const name = renameForm.elements.name.value.trim();
      const duplicate = state.libraries.some(function (library) {
        return library.id !== id && library.name.toLocaleLowerCase() === name.toLocaleLowerCase();
      });
      setError(renameForm, 'rename', !name ? '请输入媒体库名称。' : duplicate ? '名称已被其他媒体库使用。' : '');
      if (!name || duplicate) return;
      const library = state.libraries.find(function (item) { return item.id === id; });
      library.name = name;
      saveState();
      closeDialog('rename-library');
      renderLibraries();
      showToast('媒体库名称已更新');
    });

    const removeForm = document.querySelector('[data-remove-library-form]');
    removeForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const id = removeForm.elements.id.value;
      const library = state.libraries.find(function (item) { return item.id === id; });
      if (!library) return;
      state.libraries = state.libraries.filter(function (item) { return item.id !== id; });
      state.recentJobs = state.recentJobs.filter(function (job) { return job.libraryId !== id; });
      state.cache.used = Math.max(0.2, Number((state.cache.used - 0.8).toFixed(1)));
      saveState();
      closeDialog('remove-library');
      renderLibraries();
      showToast(`已从 FolioPath 移除“${library.name}”，原始文件未受影响`);
    });

    document.addEventListener('click', function (event) {
      if (!event.target.closest('.library-action-menu')) closeAllMenus();
    });
    ensureScanTimer();
  }

  function renderScanTasks() {
    const list = document.querySelector('[data-scan-task-list]');
    if (!list) return;
    list.innerHTML = state.libraries.map(function (library) {
      const safeName = escapeHtml(library.name);
      const safeIssue = escapeHtml(library.issue);
      const progress = library.status === 'scanning'
        ? `<div class="scan-progress" aria-label="扫描进度 ${library.progress}%"><span style="width:${library.progress}%"></span></div>`
        : '';
      const action = library.status === 'scanning'
        ? `<button class="btn btn-secondary btn-sm" type="button" data-task-action="cancel">取消扫描</button>`
        : library.status === 'offline' || library.status === 'failed'
          ? `<button class="btn btn-secondary btn-sm" type="button" data-task-action="retry">重新检查</button>`
          : `<button class="btn btn-secondary btn-sm" type="button" data-task-action="scan">立即扫描</button>`;
      return `
        <article class="scan-task-card" data-library-id="${library.id}">
          <div class="scan-task-top">
            <div>
              <span class="library-status ${library.status}"><span class="status-dot" aria-hidden="true"></span>${statusLabel(library.status)}</span>
              <h3>${safeName}</h3>
              <p>${library.status === 'scanning'
                ? `${library.progress}% · 已发现 ${formatNumber(library.itemCount)} 项`
                : safeIssue || `最近扫描：${escapeHtml(library.lastScan)}`}</p>
            </div>
            ${action}
          </div>
          ${progress}
          <div class="scan-task-stats">
            <span>${formatNumber(library.directoryCount)} 个目录</span>
            <span>${formatNumber(library.itemCount)} 个媒体</span>
            <span>${library.issue ? '1 个问题' : '0 个问题'}</span>
          </div>
        </article>`;
    }).join('');

    list.querySelectorAll('[data-task-action]').forEach(function (button) {
      button.addEventListener('click', function () {
        const id = button.closest('[data-library-id]').dataset.libraryId;
        const library = state.libraries.find(function (item) { return item.id === id; });
        const action = button.dataset.taskAction;
        if (action === 'scan') startScan(library);
        if (action === 'cancel') cancelScan(library);
        if (action === 'retry') retryLibrary(library);
      });
    });
  }

  function getOperationTasks() {
    const libraryTasks = state.libraries.map(function (library) {
      return {
        id: `scan-${library.id}`,
        kind: 'scan',
        libraryId: library.id,
        title: `${library.name}完整扫描`,
        description: library.issue || '增量遍历目录并更新可靠索引。',
        status: library.status,
        progress: library.status === 'scanning' ? library.progress : library.status === 'ready' ? 100 : 0,
        completed: library.status === 'ready'
          ? library.itemCount
          : library.status === 'scanning'
            ? Math.round(library.itemCount * library.progress / 100)
            : 0,
        total: library.itemCount,
        updatedAt: library.lastScan
      };
    });
    return libraryTasks.concat(state.operations);
  }

  function renderStorageHealth() {
    const container = document.querySelector('[data-storage-health]');
    if (!container) return;
    const tasks = getOperationTasks();
    const running = tasks.filter(function (task) {
      return task.status === 'running' || task.status === 'scanning' || task.status === 'waiting';
    }).length;
    const attention = tasks.filter(function (task) {
      return task.status === 'offline' || task.status === 'failed' || task.status === 'cancelled';
    }).length;
    const online = state.libraries.filter(function (library) { return library.status !== 'offline'; }).length;
    const cachePercent = Math.round((state.cache.used / state.cache.quota) * 100);
    container.innerHTML = `
      <article class="health-card">
        <span class="health-label">任务</span>
        <strong>${running}</strong>
        <span>${running ? '正在运行或等待' : '当前没有活动任务'}</span>
      </article>
      <article class="health-card${attention ? ' attention' : ''}">
        <span class="health-label">需处理</span>
        <strong>${attention}</strong>
        <span>${attention ? '保留最后可靠状态' : '没有失败或离线任务'}</span>
      </article>
      <article class="health-card">
        <span class="health-label">媒体库</span>
        <strong>${online}/${state.libraries.length}</strong>
        <span>当前可访问</span>
      </article>
      <article class="health-card">
        <span class="health-label">缓存</span>
        <strong>${cachePercent}%</strong>
        <span>${state.cache.used.toFixed(1)} / ${state.cache.quota} GiB</span>
      </article>`;
  }

  function taskMatchesFilter(task, filter) {
    if (filter === 'all') return true;
    if (filter === 'attention') {
      return ['offline', 'failed', 'cancelled', 'attention'].includes(task.status);
    }
    return ['running', 'scanning', 'waiting', 'creating'].includes(task.status);
  }

  function renderOperationTasks(filter) {
    const list = document.querySelector('[data-operation-task-list]');
    if (!list) return;
    const selectedFilter = filter || list.dataset.filter || 'active';
    list.dataset.filter = selectedFilter;
    const tasks = getOperationTasks().filter(function (task) {
      return taskMatchesFilter(task, selectedFilter);
    });

    document.querySelectorAll('[data-task-filter]').forEach(function (button) {
      const active = button.dataset.taskFilter === selectedFilter;
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    });

    if (!tasks.length) {
      list.innerHTML = `
        <div class="settings-empty compact-empty">
          <strong>${selectedFilter === 'attention' ? '没有需要处理的任务' : '当前没有活动任务'}</strong>
          <span>${selectedFilter === 'attention'
            ? '失败记录会保留在这里，直到重试成功。'
            : '可以切换到“全部”查看任务历史。'}</span>
        </div>`;
      return;
    }

    list.innerHTML = tasks.map(function (task) {
      const statusClass = operationStatusClass(task.status);
      const progress = ['running', 'scanning', 'creating'].includes(task.status)
        ? `<div class="scan-progress" aria-label="${escapeHtml(task.title)}进度 ${task.progress}%"><span style="width:${task.progress}%"></span></div>`
        : '';
      let action = '';
      if (task.kind === 'scan' && task.status === 'scanning') {
        action = '<button class="btn btn-secondary btn-sm" type="button" data-operation-action="cancel">取消</button>';
      } else if (task.kind === 'scan' && ['offline', 'failed', 'cancelled'].includes(task.status)) {
        action = '<button class="btn btn-secondary btn-sm" type="button" data-operation-action="retry">重试</button>';
      } else if (task.kind === 'scan' && task.status === 'ready') {
        action = '<button class="btn btn-secondary btn-sm" type="button" data-operation-action="start">再次扫描</button>';
      } else if (['running', 'waiting'].includes(task.status)) {
        action = '<button class="btn btn-secondary btn-sm" type="button" data-operation-action="cancel">取消</button>';
      } else if (['failed', 'cancelled'].includes(task.status)) {
        action = '<button class="btn btn-secondary btn-sm" type="button" data-operation-action="retry">重试</button>';
      }
      return `
        <article class="operation-task-card" data-operation-id="${escapeHtml(task.id)}">
          <div class="operation-task-main">
            <span class="library-status ${statusClass}"><span class="status-dot" aria-hidden="true"></span>${statusLabel(task.status)}</span>
            <div class="operation-task-copy">
              <h3>${escapeHtml(task.title)}</h3>
              <p>${escapeHtml(task.description)}</p>
            </div>
          </div>
          <div class="operation-task-progress">
            <span>${task.progress}%</span>
            ${progress}
            <small>${escapeHtml(task.updatedAt)}</small>
          </div>
          <div class="operation-task-actions">
            ${action}
            <a class="btn btn-ghost btn-sm" href="11-task-detail.html?id=${encodeURIComponent(task.id)}">查看详情</a>
          </div>
        </article>`;
    }).join('');

    list.querySelectorAll('[data-operation-action]').forEach(function (button) {
      button.addEventListener('click', function () {
        handleOperationAction(
          button.closest('[data-operation-id]').dataset.operationId,
          button.dataset.operationAction
        );
      });
    });
  }

  function handleOperationAction(id, action) {
    if (id.startsWith('scan-')) {
      const library = state.libraries.find(function (item) { return `scan-${item.id}` === id; });
      if (!library) return;
      if (action === 'cancel') cancelScan(library);
      if (action === 'retry') retryLibrary(library);
      if (action === 'start') startScan(library);
    } else {
      const operation = state.operations.find(function (item) { return item.id === id; });
      if (!operation) return;
      if (action === 'cancel') {
        operation.status = 'cancelled';
        operation.updatedAt = '刚刚取消 · 已生成的安全缓存保留';
      }
      if (action === 'retry' || action === 'start') {
        operation.status = 'running';
        operation.progress = Math.max(4, operation.progress || 0);
        operation.updatedAt = '刚刚重新开始';
      }
      saveState();
      showToast(action === 'cancel' ? `已取消“${operation.title}”` : `已开始“${operation.title}”`);
    }
    renderOperationTasks();
    renderStorageHealth();
    renderTaskDetail();
    ensureOperationTimer();
  }

  function startCacheOperation(id, rebuild) {
    let operation = state.operations.find(function (item) { return item.id === id; });
    if (!operation) {
      operation = {
        id: id,
        title: rebuild ? '重建全部缓存' : '补齐缺失缓存',
        description: rebuild
          ? '重新生成全部缩略图与视频封面，新文件成功后再替换现有缓存。'
          : '仅生成缺失或源指纹已变化的缩略图与视频封面。',
        status: 'running',
        progress: 3,
        completed: 0,
        total: rebuild ? 17800 : 162,
        updatedAt: '刚刚开始'
      };
      state.operations.unshift(operation);
    } else {
      operation.status = 'running';
      operation.progress = 3;
      operation.completed = 0;
      operation.total = rebuild ? 17800 : Math.max(operation.total, 162);
      operation.description = rebuild
        ? '重新生成全部缩略图与视频封面，新文件成功后再替换现有缓存。'
        : '仅生成缺失或源指纹已变化的缩略图与视频封面。';
      operation.updatedAt = '刚刚开始';
    }
    saveState();
    renderOperationTasks();
    renderStorageHealth();
    ensureOperationTimer();
    showToast(rebuild ? '全部缓存重建任务已创建' : '缺失缓存补齐任务已创建');
  }

  function ensureOperationTimer() {
    window.clearInterval(operationTimer);
    if (!state.operations.some(function (operation) {
      return operation.status === 'running' || operation.status === 'waiting';
    })) return;
    operationTimer = window.setInterval(function () {
      let changed = false;
      const hasRunning = state.operations.some(function (operation) { return operation.status === 'running'; });
      state.operations.forEach(function (operation) {
        if (operation.status === 'waiting' && !hasRunning) {
          operation.status = 'running';
          operation.updatedAt = '已获得处理资源';
          changed = true;
          return;
        }
        if (operation.status !== 'running') return;
        operation.progress = Math.min(100, operation.progress + 3);
        operation.completed = Math.min(
          operation.total,
          Math.round(operation.total * operation.progress / 100)
        );
        operation.updatedAt = '刚刚';
        changed = true;
        if (operation.progress >= 100) {
          operation.status = 'success';
          operation.updatedAt = '刚刚完成';
        }
      });
      if (!changed) return;
      saveState();
      renderOperationTasks();
      renderStorageHealth();
      renderTaskDetail();
      if (!state.operations.some(function (operation) {
        return operation.status === 'running' || operation.status === 'waiting';
      })) {
        window.clearInterval(operationTimer);
        showToast('派生媒体任务已完成');
      }
    }, 1200);
  }

  function taskDetailMarkup(task) {
    const statusClass = operationStatusClass(task.status);
    const active = ['running', 'scanning', 'creating'].includes(task.status);
    const reliableCopy = task.kind === 'scan'
      ? '失败、离线、中断或取消都不会清理最后可靠索引。'
      : '已完成的派生文件会保留；原始媒体始终只读。';
    return `
      <nav class="detail-breadcrumb" aria-label="面包屑">
        <a href="08-settings-storage.html">扫描与缓存</a>
        <span aria-hidden="true">/</span>
        <span>${escapeHtml(task.title)}</span>
      </nav>
      <div class="settings-hero task-detail-hero">
        <div>
          <span class="library-status ${statusClass}"><span class="status-dot" aria-hidden="true"></span>${statusLabel(task.status)}</span>
          <h1>${escapeHtml(task.title)}</h1>
          <p>${escapeHtml(task.description)}</p>
        </div>
        <div class="hero-actions">
          ${active
            ? '<button class="btn btn-secondary" type="button" data-detail-action="cancel">取消任务</button>'
            : ['failed', 'cancelled', 'offline'].includes(task.status)
              ? '<button class="btn btn-primary" type="button" data-detail-action="retry">重新执行</button>'
              : '<button class="btn btn-secondary" type="button" data-detail-action="start">再次执行</button>'}
        </div>
      </div>
      <section aria-labelledby="detail-progress-title">
        <h2 class="settings-section-title" id="detail-progress-title">当前进度</h2>
        <div class="task-detail-progress-card">
          <div class="task-detail-progress-value">
            <strong>${task.progress}%</strong>
            <span>${formatNumber(task.completed || 0)} / ${formatNumber(task.total || 0)} 项</span>
          </div>
          <div class="usage-track"><span style="width:${task.progress}%"></span></div>
          <p>${escapeHtml(task.updatedAt)}</p>
        </div>
      </section>
      <section aria-labelledby="detail-stage-title">
        <h2 class="settings-section-title" id="detail-stage-title">执行阶段</h2>
        <ol class="task-stage-list">
          <li class="done"><strong>任务已入队</strong><span>记录任务参数并检查重复任务。</span></li>
          <li class="${task.progress > 8 ? 'done' : active ? 'active' : ''}"><strong>读取可靠状态</strong><span>验证媒体库连接、索引代次和缓存水位。</span></li>
          <li class="${task.progress >= 100 ? 'done' : task.progress > 8 ? 'active' : ''}"><strong>${task.kind === 'scan' ? '扫描并提交批次' : '生成并原子替换缓存'}</strong><span>以有界批次处理，可协作取消。</span></li>
          <li class="${task.progress >= 100 ? 'done' : ''}"><strong>完成与记录</strong><span>保存结果、跳过数量和可恢复错误摘要。</span></li>
        </ol>
      </section>
      <div class="inline-status ${['failed', 'offline'].includes(task.status) ? 'warning' : 'info'} settings-note">
        <span>${reliableCopy}</span>
      </div>`;
  }

  function renderTaskDetail() {
    const container = document.querySelector('[data-task-detail]');
    if (!container) return;
    const id = new URLSearchParams(window.location.search).get('id') || 'video-posters';
    const task = getOperationTasks().find(function (item) { return item.id === id; });
    if (!task) {
      container.innerHTML = `
        <div class="settings-empty">
          <strong>找不到这个任务</strong>
          <span>任务可能已从原型状态中移除。</span>
          <a class="btn btn-primary" href="08-settings-storage.html">返回任务中心</a>
        </div>`;
      return;
    }
    container.innerHTML = taskDetailMarkup(task);
    const action = container.querySelector('[data-detail-action]');
    if (action) {
      action.addEventListener('click', function () {
        handleOperationAction(task.id, action.dataset.detailAction);
      });
    }
  }

  function renderCache() {
    const usageCopy = document.querySelector('[data-cache-usage-copy]');
    if (!usageCopy) return;
    const percent = Math.min(100, Math.round((state.cache.used / state.cache.quota) * 100));
    usageCopy.textContent = `${state.cache.used.toFixed(1)} GiB / ${state.cache.quota} GiB`;
    document.querySelector('[data-cache-percent]').textContent = `${percent}%`;
    document.querySelector('[data-cache-usage-bar]').style.width = `${percent}%`;
    const form = document.querySelector('[data-cache-form]');
    form.elements.quota.value = state.cache.quota;
    form.querySelector('[data-save-cache]').disabled = true;
    renderStorageHealth();
  }

  function initStorage() {
    const scanForm = document.querySelector('[data-scan-settings-form]');
    if (!scanForm) return;
    const scanSave = scanForm.querySelector('[data-save-scan-settings]');
    const scanState = scanForm.querySelector('[data-scan-save-state]');

    function fillScan() {
      scanForm.elements.scheduled.checked = state.scan.scheduled;
      scanForm.elements.interval.value = state.scan.interval;
      scanForm.elements.interval.disabled = !state.scan.scheduled;
      scanSave.disabled = true;
      scanState.textContent = '没有未保存的更改';
      scanState.classList.remove('dirty');
    }

    function scanDirty() {
      const scheduled = scanForm.elements.scheduled.checked;
      const interval = Number(scanForm.elements.interval.value);
      scanForm.elements.interval.disabled = !scheduled;
      const dirty = scheduled !== state.scan.scheduled || interval !== state.scan.interval;
      scanSave.disabled = !dirty;
      scanState.textContent = dirty ? '有未保存的更改' : '没有未保存的更改';
      scanState.classList.toggle('dirty', dirty);
    }

    scanForm.addEventListener('input', scanDirty);
    scanForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const scheduled = scanForm.elements.scheduled.checked;
      const interval = Number(scanForm.elements.interval.value);
      const invalid = scheduled && (!Number.isFinite(interval) || interval < 1 || interval > 8760);
      setError(scanForm, 'scan-settings', invalid ? '扫描间隔必须在 1 到 8760 小时之间。' : '');
      if (invalid) return;
      state.scan.scheduled = scheduled;
      state.scan.interval = interval || state.scan.interval;
      saveState();
      fillScan();
      showToast(scheduled ? `定时扫描已保存：每 ${state.scan.interval} 小时` : '定时完整扫描已关闭');
    });

    const cacheForm = document.querySelector('[data-cache-form]');
    const cacheSave = cacheForm.querySelector('[data-save-cache]');
    cacheForm.addEventListener('input', function () {
      cacheSave.disabled = Number(cacheForm.elements.quota.value) === state.cache.quota;
    });
    cacheForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const quota = Number(cacheForm.elements.quota.value);
      const invalid = !Number.isFinite(quota) || quota < 1 || quota > 1024;
      setError(cacheForm, 'quota', invalid ? '缓存配额必须在 1 到 1024 GiB 之间。' : '');
      if (invalid) return;
      state.cache.quota = quota;
      if (state.cache.used > quota) state.cache.used = Number((quota * 0.8).toFixed(1));
      saveState();
      renderCache();
      showToast('缓存配额已保存');
    });

    document.querySelector('[data-open-clear-cache]').addEventListener('click', function () {
      openDialog('clear-cache');
    });
    document.querySelector('[data-confirm-clear-cache]').addEventListener('click', function () {
      state.cache.used = 0.2;
      saveState();
      closeDialog('clear-cache');
      renderCache();
      showToast('可重建缓存已清空，原始媒体和索引未受影响');
    });

    document.querySelectorAll('[data-task-filter]').forEach(function (button) {
      button.addEventListener('click', function () {
        renderOperationTasks(button.dataset.taskFilter);
      });
    });

    const missingCacheButton = document.querySelector('[data-start-missing-cache]');
    if (missingCacheButton) {
      missingCacheButton.addEventListener('click', function () {
        startCacheOperation('cache-missing', false);
      });
    }
    const rebuildButton = document.querySelector('[data-open-rebuild-cache]');
    if (rebuildButton) {
      rebuildButton.addEventListener('click', function () {
        openDialog('rebuild-cache');
      });
    }
    const confirmRebuild = document.querySelector('[data-confirm-rebuild-cache]');
    if (confirmRebuild) {
      confirmRebuild.addEventListener('click', function () {
        closeDialog('rebuild-cache');
        startCacheOperation('cache-rebuild', true);
      });
    }

    fillScan();
    renderScanTasks();
    renderStorageHealth();
    renderOperationTasks();
    renderCache();
    ensureScanTimer();
    ensureOperationTimer();
  }

  function renderMaintenanceHealth() {
    const container = document.querySelector('[data-maintenance-health]');
    if (!container) return;
    const issues = state.maintenance.checks.filter(function (check) {
      return check.status === 'attention' || check.status === 'failed';
    }).length;
    const offline = state.libraries.filter(function (library) { return library.status === 'offline'; }).length;
    container.innerHTML = `
      <article class="health-card${offline ? ' attention' : ''}">
        <span class="health-label">媒体根目录</span>
        <strong>${offline ? '需检查' : '只读可用'}</strong>
        <span>${offline ? `${offline} 个媒体库离线` : '未发现嵌套挂载问题'}</span>
      </article>
      <article class="health-card">
        <span class="health-label">应用数据</span>
        <strong>4.8 GiB</strong>
        <span>可用空间 72.4 GiB</span>
      </article>
      <article class="health-card${issues ? ' attention' : ''}">
        <span class="health-label">完整性</span>
        <strong>${issues ? `${issues} 项` : '正常'}</strong>
        <span>${issues ? '报告需要管理员复核' : '所有检查均通过'}</span>
      </article>
      <article class="health-card">
        <span class="health-label">FolioPath</span>
        <strong>v0.1.0</strong>
        <span>数据库结构版本 12</span>
      </article>`;
  }

  function renderIntegrityChecks() {
    const list = document.querySelector('[data-integrity-list]');
    if (!list) return;
    list.innerHTML = state.maintenance.checks.map(function (check) {
      const statusClass = operationStatusClass(check.status);
      const running = check.status === 'running';
      return `
        <article class="integrity-card" data-check-id="${check.id}">
          <div class="integrity-card-main">
            <span class="library-status ${statusClass}"><span class="status-dot" aria-hidden="true"></span>${statusLabel(check.status)}</span>
            <div>
              <h3>${escapeHtml(check.title)}</h3>
              <p>${escapeHtml(check.description)}</p>
            </div>
          </div>
          <div class="integrity-result">
            <strong>${escapeHtml(check.result)}</strong>
            <span>${escapeHtml(check.updatedAt)}</span>
          </div>
          <button class="btn btn-secondary btn-sm" type="button" data-run-check="${check.id}"${running ? ' disabled' : ''}>
            ${running ? '检查中…' : '重新检查'}
          </button>
        </article>`;
    }).join('');

    list.querySelectorAll('[data-run-check]').forEach(function (button) {
      button.addEventListener('click', function () {
        runIntegrityCheck(button.dataset.runCheck);
      });
    });
  }

  function runIntegrityCheck(id) {
    const check = state.maintenance.checks.find(function (item) { return item.id === id; });
    if (!check || check.status === 'running') return;
    check.status = 'running';
    check.result = '正在读取可靠状态';
    check.updatedAt = '刚刚开始';
    saveState();
    renderIntegrityChecks();
    window.setTimeout(function () {
      if (id === 'missing') {
        check.status = 'attention';
        check.result = '38 项待复核';
      } else if (id === 'derived') {
        check.status = 'attention';
        check.result = '124 个缓存可补齐';
      } else {
        check.status = 'healthy';
        check.result = '没有发现问题';
      }
      check.updatedAt = '刚刚完成';
      saveState();
      renderIntegrityChecks();
      renderMaintenanceHealth();
      showToast(`“${check.title}”检查已完成`);
    }, 1100);
  }

  function renderBackups() {
    const list = document.querySelector('[data-backup-list]');
    if (!list) return;
    if (!state.maintenance.backups.length) {
      list.innerHTML = '<div class="settings-empty compact-empty"><strong>还没有备份</strong><span>立即创建第一份应用数据备份。</span></div>';
      return;
    }
    list.innerHTML = `
      <div class="backup-list-header">
        <strong>最近备份</strong>
        <span>最多保留 ${state.maintenance.backup.retention} 份</span>
      </div>
      ${state.maintenance.backups.map(function (backup) {
        return `
          <article class="backup-row">
            <div>
              <strong>${backup.status === 'creating' ? '正在创建备份' : escapeHtml(backup.createdAt)}</strong>
              <span>${backup.status === 'creating' ? '数据库快照与设置正在写入临时文件' : `${escapeHtml(backup.size)} · 校验通过`}</span>
            </div>
            <span class="library-status ${backup.status === 'creating' ? 'scanning' : 'ready'}">
              <span class="status-dot" aria-hidden="true"></span>${backup.status === 'creating' ? '创建中' : '可用'}
            </span>
          </article>`;
      }).join('')}`;
  }

  function createBackup() {
    const backup = {
      id: `backup-${Date.now()}`,
      createdAt: currentTimestamp(),
      size: '计算中',
      status: 'creating'
    };
    state.maintenance.backups.unshift(backup);
    saveState();
    renderBackups();
    showToast('应用数据备份已开始，可以继续使用 FolioPath');
    window.setTimeout(function () {
      backup.status = 'ready';
      backup.size = '18.8 MiB';
      state.maintenance.backups = state.maintenance.backups.slice(
        0,
        state.maintenance.backup.retention
      );
      saveState();
      renderBackups();
      showToast('应用数据备份已创建并校验');
    }, 1500);
  }

  function exportDiagnostics() {
    const payload = {
      generatedAt: new Date().toISOString(),
      application: { name: 'FolioPath', version: '0.1.0', schemaVersion: 12 },
      libraries: state.libraries.map(function (library) {
        return {
          id: library.id,
          status: library.status,
          items: library.itemCount,
          directories: library.directoryCount
        };
      }),
      jobs: getOperationTasks().map(function (task) {
        return { id: task.id, status: task.status, progress: task.progress };
      }),
      note: 'Prototype diagnostics. Credentials, host paths and original-media metadata are excluded.'
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = 'foliopath-diagnostics.json';
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.setTimeout(function () { URL.revokeObjectURL(link.href); }, 0);
    showToast('脱敏诊断包已导出');
  }

  function initMaintenance() {
    const form = document.querySelector('[data-backup-form]');
    if (!form) return;
    const saveButton = form.querySelector('[data-save-backup-settings]');
    const stateLabel = form.querySelector('[data-backup-save-state]');

    function fillBackupForm() {
      form.elements.enabled.checked = state.maintenance.backup.enabled;
      form.elements.schedule.value = state.maintenance.backup.schedule;
      form.elements.retention.value = state.maintenance.backup.retention;
      form.elements.schedule.disabled = !state.maintenance.backup.enabled;
      saveButton.disabled = true;
      stateLabel.textContent = '没有未保存的更改';
      stateLabel.classList.remove('dirty');
    }

    function updateBackupDirty() {
      form.elements.schedule.disabled = !form.elements.enabled.checked;
      const dirty = form.elements.enabled.checked !== state.maintenance.backup.enabled ||
        form.elements.schedule.value !== state.maintenance.backup.schedule ||
        Number(form.elements.retention.value) !== state.maintenance.backup.retention;
      saveButton.disabled = !dirty;
      stateLabel.textContent = dirty ? '有未保存的更改' : '没有未保存的更改';
      stateLabel.classList.toggle('dirty', dirty);
    }

    form.addEventListener('input', updateBackupDirty);
    form.addEventListener('submit', function (event) {
      event.preventDefault();
      const retention = Number(form.elements.retention.value);
      const invalid = !Number.isInteger(retention) || retention < 1 || retention > 30;
      setError(form, 'backup-settings', invalid ? '备份保留数量必须是 1 到 30 的整数。' : '');
      if (invalid) return;
      state.maintenance.backup.enabled = form.elements.enabled.checked;
      state.maintenance.backup.schedule = form.elements.schedule.value;
      state.maintenance.backup.retention = retention;
      state.maintenance.backups = state.maintenance.backups.slice(0, retention);
      saveState();
      fillBackupForm();
      renderBackups();
      showToast(state.maintenance.backup.enabled ? '定时备份计划已保存' : '定时备份已关闭');
    });

    document.querySelector('[data-run-all-checks]').addEventListener('click', function () {
      state.maintenance.checks.forEach(function (check) {
        if (check.status !== 'running') runIntegrityCheck(check.id);
      });
    });
    document.querySelector('[data-create-backup]').addEventListener('click', function () {
      openDialog('create-backup');
    });
    document.querySelector('[data-confirm-create-backup]').addEventListener('click', function () {
      closeDialog('create-backup');
      createBackup();
    });
    document.querySelector('[data-export-diagnostics]').addEventListener('click', exportDiagnostics);

    fillBackupForm();
    renderMaintenanceHealth();
    renderIntegrityChecks();
    renderBackups();
  }

  function initAccount() {
    const profileForm = document.querySelector('[data-profile-form]');
    if (!profileForm) return;
    const profileSave = profileForm.querySelector('[data-save-profile]');
    const profileState = profileForm.querySelector('[data-profile-state]');
    profileForm.elements.displayName.value = state.account.displayName;

    profileForm.addEventListener('input', function () {
      const dirty = profileForm.elements.displayName.value.trim() !== state.account.displayName;
      profileSave.disabled = !dirty;
      profileState.textContent = dirty ? '有未保存的更改' : '没有未保存的更改';
      profileState.classList.toggle('dirty', dirty);
    });

    profileForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const displayName = profileForm.elements.displayName.value.trim();
      setError(profileForm, 'displayName', displayName ? '' : '请输入显示名称。');
      if (!displayName) return;
      state.account.displayName = displayName;
      saveState();
      profileSave.disabled = true;
      profileState.textContent = '没有未保存的更改';
      profileState.classList.remove('dirty');
      updateHeaderAccount();
      showToast('管理员资料已保存');
    });

    const passwordForm = document.querySelector('[data-password-form]');
    passwordForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const current = passwordForm.elements.currentPassword.value;
      const next = passwordForm.elements.newPassword.value;
      const confirmation = passwordForm.elements.confirmPassword.value;
      setError(passwordForm, 'currentPassword', current ? '' : '请输入当前密码。');
      setError(passwordForm, 'newPassword', next.length >= 8 ? '' : '新密码至少需要 8 个字符。');
      setError(passwordForm, 'confirmPassword', confirmation === next ? '' : '两次输入的新密码不一致。');
      if (!current || next.length < 8 || confirmation !== next) return;
      passwordForm.reset();
      showToast('密码已更新，其他会话需要重新登录');
    });

    document.querySelector('[data-open-logout]').addEventListener('click', function () {
      openDialog('logout');
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    renderNavigation();
    bindDialogs();
    updateHeaderAccount();
    initGeneral();
    initLibraries();
    initStorage();
    initMaintenance();
    initAccount();
    renderTaskDetail();
    ensureOperationTimer();
  });
})();
