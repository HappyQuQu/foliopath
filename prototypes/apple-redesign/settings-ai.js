(function () {
  'use strict';

  let indexPaused = false;
  let toastTimer;
  let downloadTimer;

  function showToast(message) {
    const toast = document.querySelector('[data-toast]');
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.hidden = false;
    toastTimer = window.setTimeout(function () { toast.hidden = true; }, 3600);
  }

  function openDialog(name) {
    const dialog = document.querySelector(`[data-ai-dialog="${name}"]`);
    if (!dialog) return;
    dialog.hidden = false;
    document.body.classList.add('dialog-open');
    const button = dialog.querySelector('button');
    if (button) button.focus();
  }

  function closeDialog(dialog) {
    const target = typeof dialog === 'string'
      ? document.querySelector(`[data-ai-dialog="${dialog}"]`)
      : dialog;
    if (!target) return;
    target.hidden = true;
    document.body.classList.remove('dialog-open');
  }

  function setIndexState(status, percent, count) {
    document.querySelector('[data-index-status]').textContent = status;
    document.querySelector('[data-index-percent]').textContent = percent + '%';
    document.querySelector('[data-index-bar]').style.width = percent + '%';
    document.querySelector('[data-index-count]').textContent = count;
  }

  document.addEventListener('DOMContentLoaded', function () {
    const master = document.querySelector('[data-library-enabled]');
    const capabilityGrid = document.querySelector('[data-capability-grid]');
    const capabilities = Array.from(document.querySelectorAll('[data-capability]'));
    const masterCopy = document.querySelector('[data-library-enabled-copy]');

    master.addEventListener('change', function () {
      capabilities.forEach(function (control) { control.disabled = !master.checked; });
      capabilityGrid.classList.toggle('disabled', !master.checked);
      masterCopy.textContent = master.checked ? '已启用' : '已关闭';
      showToast(master.checked ? '家庭影像的智能功能已启用' : '智能功能已关闭；现有派生索引暂时保留');
    });

    capabilities.forEach(function (control) {
      control.addEventListener('change', function () {
        const label = control.closest('.ai-capability').querySelector('strong').textContent;
        showToast(label + (control.checked ? '已启用' : '已关闭'));
      });
    });

    document.querySelector('[data-check-models]').addEventListener('click', function (event) {
      const button = event.currentTarget;
      button.disabled = true;
      button.textContent = '检查中…';
      window.setTimeout(function () {
        button.disabled = false;
        button.textContent = '检查运行环境';
        showToast('运行环境检查完成：2 个模型包均可用');
      }, 700);
    });

    document.querySelector('[data-repair-models]').addEventListener('click', function (event) {
      const button = event.currentTarget;
      button.disabled = true;
      button.textContent = '校验中…';
      window.setTimeout(function () {
        button.disabled = false;
        button.textContent = '校验并修复';
        showToast('模型包校验通过，无需修复');
      }, 800);
    });

    document.querySelector('[data-open-acquire-model]').addEventListener('click', function () {
      openDialog('acquire-model');
    });

    const sourceTabs = Array.from(document.querySelectorAll('[data-model-source-tab]'));
    const sourcePanels = Array.from(document.querySelectorAll('[data-model-source-panel]'));
    sourceTabs.forEach(function (tab) {
      tab.addEventListener('click', function () {
        sourceTabs.forEach(function (item) {
          const active = item === tab;
          item.classList.toggle('active', active);
          item.setAttribute('aria-selected', String(active));
        });
        sourcePanels.forEach(function (panel) {
          panel.hidden = panel.dataset.modelSourcePanel !== tab.dataset.modelSourceTab;
        });
      });
    });

    document.querySelector('[data-start-model-download]').addEventListener('click', function (event) {
      const button = event.currentTarget;
      const progress = document.querySelector('[data-download-progress]');
      const bar = document.querySelector('[data-download-bar]');
      const percent = document.querySelector('[data-download-percent]');
      const detail = document.querySelector('[data-download-detail]');
      const selected = document.querySelector('input[name="download-package"]:checked');
      let value = 0;
      button.disabled = true;
      button.textContent = '下载中…';
      progress.hidden = false;
      window.clearInterval(downloadTimer);
      downloadTimer = window.setInterval(function () {
        value = Math.min(100, value + 20);
        bar.style.width = value + '%';
        percent.textContent = value + '%';
        detail.textContent = value < 80 ? '断点续传已启用 · 正在接收模型包…' : '正在校验 SHA-256 与兼容清单…';
        if (value < 100) return;
        window.clearInterval(downloadTimer);
        button.disabled = false;
        button.textContent = '下载并校验';
        detail.textContent = '校验通过，已保存到 /app/data/models';
        showToast((selected.value === 'vision' ? '视觉理解' : '人脸特征') + '模型包已下载并验证');
      }, 220);
    });

    document.querySelector('[data-rescan-models]').addEventListener('click', function (event) {
      const button = event.currentTarget;
      button.disabled = true;
      button.textContent = '扫描中…';
      window.setTimeout(function () {
        button.disabled = false;
        button.textContent = '重新扫描';
        showToast('已扫描 /models：发现 2 个兼容包、1 个不兼容文件');
      }, 650);
    });

    document.querySelectorAll('input[name="local-model-mode"]').forEach(function (control) {
      control.addEventListener('change', function () {
        const note = document.querySelector('[data-local-mode-note]');
        note.textContent = control.value === 'direct'
          ? '直接使用只在 /models 为只读且 SHA-256 持续匹配时可用。文件缺失或变化后功能会停用，现有索引不会被删除。'
          : '导入需要额外 326 MiB 空间。复制完成前不会替换当前可用模型。';
      });
    });

    document.querySelector('[data-import-models]').addEventListener('click', function (event) {
      const selected = document.querySelector('[data-local-package]');
      if (!selected.checked) {
        showToast('请先选择要导入的兼容模型包');
        return;
      }
      const button = event.currentTarget;
      const mode = document.querySelector('input[name="local-model-mode"]:checked').value;
      button.disabled = true;
      button.textContent = mode === 'direct' ? '校验并启用中…' : '校验并导入中…';
      window.setTimeout(function () {
        closeDialog('acquire-model');
        button.disabled = false;
        button.textContent = '应用所选模型';
        showToast(mode === 'direct'
          ? '已从只读 /models 启用模型；版本和 SHA-256 已固定'
          : '模型已从 /models 导入托管目录；外部文件不会被修改');
      }, 800);
    });

    document.querySelector('[data-toggle-index]').addEventListener('click', function (event) {
      indexPaused = !indexPaused;
      event.currentTarget.textContent = indexPaused ? '继续' : '暂停';
      document.querySelector('[data-index-status]').textContent = indexPaused
        ? '已暂停 · 当前可靠索引仍可使用'
        : '正在后台建立索引 · 剩余约 18 分钟';
      showToast(indexPaused ? '智能索引任务已暂停' : '智能索引任务已继续');
    });

    document.querySelector('[data-open-rebuild]').addEventListener('click', function () { openDialog('rebuild'); });
    document.querySelector('[data-open-clear]').addEventListener('click', function () { openDialog('clear'); });
    document.querySelectorAll('[data-close-ai-dialog]').forEach(function (button) {
      button.addEventListener('click', function () { closeDialog(button.closest('[data-ai-dialog]')); });
    });
    document.querySelectorAll('[data-ai-dialog]').forEach(function (dialog) {
      dialog.addEventListener('click', function (event) {
        if (event.target === dialog) closeDialog(dialog);
      });
    });

    document.querySelector('[data-confirm-rebuild]').addEventListener('click', function () {
      closeDialog('rebuild');
      indexPaused = false;
      document.querySelector('[data-toggle-index]').textContent = '暂停';
      setIndexState('正在建立第 4 代索引 · 当前第 3 代仍在服务', 1, '48 / 4,978 项已处理 · 失败时自动保留第 3 代');
      showToast('已开始建立新的智能索引代次');
    });

    document.querySelector('[data-confirm-clear]').addEventListener('click', function () {
      closeDialog('clear');
      setIndexState('智能派生数据已清除', 0, '人物名称和用户确认标签已保留');
      showToast('智能派生数据已清除；原始媒体未受影响');
    });

    document.querySelector('[data-concurrency]').addEventListener('change', function (event) {
      showToast('后台并发已设为 ' + event.currentTarget.value);
    });
    document.querySelector('[data-idle-only]').addEventListener('change', function (event) {
      showToast(event.currentTarget.checked ? '已启用空闲时优先运行' : '已允许持续运行智能任务');
    });

    document.addEventListener('keydown', function (event) {
      if (event.key !== 'Escape') return;
      const visible = document.querySelector('[data-ai-dialog]:not([hidden])');
      if (visible) closeDialog(visible);
    });
  });
})();
