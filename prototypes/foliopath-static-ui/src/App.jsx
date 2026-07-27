import { forwardRef, useEffect, useMemo, useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  ArrowsOut,
  CaretDown,
  CaretRight,
  Check,
  Columns,
  FolderSimple,
  Funnel,
  GearSix,
  GridFour,
  House,
  ImageSquare,
  Info,
  List,
  MagnifyingGlass,
  Moon,
  Pause,
  Play,
  PushPin,
  SidebarSimple,
  Sun,
  X,
} from "@phosphor-icons/react";

const folders = [
  { name: "清水寺", count: 312 },
  { name: "伏见稻荷大社", count: 268 },
  { name: "金阁寺", count: 189 },
  { name: "岚山", count: 245 },
  { name: "祇园", count: 176 },
  { name: "二条城", count: 132 },
  { name: "其他", count: 98 },
];

const media = [
  ["2026-07-21 19-12-43.jpg", "2026-07-21 19:12", "京都 / 八坂塔", "kyoto-pagoda-clean.jpg", "image", null, "20.3 MB", "6000 × 3376"],
  ["2026-07-21 16-45-10.mp4", "2026-07-21 16:45", "京都 / 伏见稻荷大社", "torii-tunnel.jpg", "video", "00:18", "24.3 MB", "1920 × 1080"],
  ["2026-07-20 10-22-31.jpg", "2026-07-20 10:22", "京都 / 金阁寺", "golden-pavilion.jpg", "image", null, "14.8 MB", "5472 × 3648"],
  ["2026-07-19 09-11-05.mp4", "2026-07-19 09:11", "京都 / 岚山", "bamboo-grove.jpg", "video", "00:27", "31.7 MB", "1920 × 1080"],
  ["2026-07-18 18-33-27.jpg", "2026-07-18 18:33", "京都 / 祇园", "gion-kimono.jpg", "image", null, "12.6 MB", "4000 × 6000"],
  ["2026-07-18 09-02-14.mp4", "2026-07-18 09:02", "京都 / 岚山", "arashiyama-river.jpg", "video", "00:15", "19.2 MB", "1920 × 1080"],
  ["2026-07-17 19-47-56.jpg", "2026-07-17 19:47", "京都 / 祇园", "gion-evening.jpg", "image", null, "18.4 MB", "6000 × 4000"],
  ["2026-07-17 11-08-32.jpg", "2026-07-17 11:08", "京都 / 龙安寺", "zen-garden.jpg", "image", null, "16.9 MB", "6000 × 4000"],
  ["2026-07-16 20-15-09.mp4", "2026-07-16 20:15", "京都 / 先斗町", "pontocho-night.jpg", "video", "00:22", "28.1 MB", "1920 × 1080"],
  ["2026-07-16 08-40-21.jpg", "2026-07-16 08:40", "京都 / 紫阳花园", "ajisai.jpg", "image", null, "15.2 MB", "5184 × 3456"],
  ["2026-07-15 17-30-11.jpg", "2026-07-15 17:30", "京都 / 东山", "noren.jpg", "image", null, "13.5 MB", "4000 × 6000"],
  ["2026-07-15 12-05-33.mp4", "2026-07-15 12:05", "京都 / 花见小路", "kyoto-lane.jpg", "video", "00:19", "22.6 MB", "1920 × 1080"],
  ["2026-07-14 14-22-18.jpg", "2026-07-14 14:22", "京都 / 瑠璃光院", "green-pavilion.jpg", "image", null, "17.7 MB", "6000 × 4000"],
  ["2026-07-14 06-10-27.jpg", "2026-07-14 06:10", "京都 / 鸭川", "kamo-sunset.jpg", "image", null, "11.9 MB", "6000 × 4000"],
  ["2026-07-13 15-55-42.mp4", "2026-07-13 15:55", "京都 / 贵船", "forest-shrine.jpg", "video", "00:16", "20.8 MB", "1920 × 1080"],
].map(([name, date, location, file, kind, duration, size, dimensions]) => ({
  name,
  date,
  location,
  src: `/media/${file}`,
  videoSrc: kind === "video" ? `/media/${file.replace(".jpg", ".mp4")}` : null,
  kind,
  duration,
  size,
  dimensions,
}));

const themeStorageKey = "foliopath-prototype-theme";

function useTheme() {
  const [theme, setTheme] = useState(() => {
    const saved = localStorage.getItem(themeStorageKey);
    if (saved === "light" || saved === "dark") return saved;
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  });

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  return [theme, setTheme];
}

const IconButton = forwardRef(function IconButton(
  { label, children, className = "", ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      className={`icon-button ${className}`}
      type="button"
      aria-label={label}
      title={label}
      {...props}
    >
      {children}
    </button>
  );
});

function Sidebar({ open, onClose }) {
  return (
    <>
      <button
        className={`drawer-backdrop ${open ? "is-open" : ""}`}
        type="button"
        aria-label="关闭目录面板"
        onClick={onClose}
      />
      <aside className={`sidebar ${open ? "is-open" : ""}`} aria-label="目录导航">
        <div className="sidebar-header">
          <a className="brand" href="#main" aria-label="FolioPath 首页">
            FolioPath
          </a>
          <IconButton label="关闭目录面板" className="sidebar-close" onClick={onClose}>
            <X size={20} />
          </IconButton>
        </div>

        <div className="sidebar-section-label">媒体库</div>
        <button className="library-switcher" type="button">
          <span className="library-icon">
            <ImageSquare size={22} />
          </span>
          <span className="library-copy">
            <strong>家庭影像</strong>
            <small>正在扫描 · 已发现 12,480 项</small>
          </span>
          <CaretDown size={17} />
        </button>

        <div className="sidebar-section-label directory-label">目录</div>
        <nav className="tree" aria-label="家庭影像目录">
          <button className="tree-row" type="button">
            <CaretRight size={14} />
            <ImageSquare size={18} />
            <span>全部媒体</span>
          </button>
          <button className="tree-row" type="button">
            <CaretDown size={14} />
            <FolderSimple size={18} />
            <span>家庭影像</span>
          </button>
          <button className="tree-row tree-level-1" type="button">
            <CaretRight size={14} />
            <FolderSimple size={18} />
            <span>按年份</span>
          </button>
          <button className="tree-row tree-level-1" type="button">
            <CaretDown size={14} />
            <FolderSimple size={18} />
            <span>旅行</span>
          </button>
          <button className="tree-row tree-level-2" type="button">
            <CaretDown size={14} />
            <FolderSimple size={18} />
            <span>日本</span>
          </button>
          {["东京", "京都", "大阪", "北海道"].map((name) => (
            <button
              className={`tree-row tree-level-3 ${name === "京都" ? "is-current" : ""}`}
              type="button"
              aria-current={name === "京都" ? "page" : undefined}
              onClick={name === "京都" ? onClose : undefined}
              key={name}
            >
              <CaretRight size={14} />
              <FolderSimple size={18} />
              <span>{name}</span>
            </button>
          ))}
          {["欧洲", "美洲", "国内", "生活", "活动"].map((name) => (
            <button className="tree-row" type="button" key={name}>
              <CaretRight size={14} />
              <FolderSimple size={18} />
              <span>{name}</span>
            </button>
          ))}
        </nav>

        <div className="sidebar-footer">
          <button className="tree-row" type="button">
            <GearSix size={19} />
            <span>设置</span>
          </button>
        </div>
      </aside>
    </>
  );
}

function PreviewPane({
  item,
  pinned,
  onPinnedChange,
  onClose,
  onPrevious,
  onNext,
  onResizeStart,
}) {
  const mediaRef = useRef(null);
  const [playing, setPlaying] = useState(false);

  useEffect(() => {
    const onKeyDown = (event) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  useEffect(() => {
    setPlaying(false);
  }, [item.name]);

  const enterFullscreen = async () => {
    if (!document.fullscreenElement) {
      await mediaRef.current?.requestFullscreen?.();
    } else {
      await document.exitFullscreen?.();
    }
  };

  return (
    <aside className="preview-pane" aria-label={`正在预览 ${item.name}`}>
      <button
        className="preview-resizer"
        type="button"
        aria-label="调整预览栏宽度"
        onPointerDown={onResizeStart}
      >
        <span />
      </button>

      <header className="preview-header">
        <strong>预览</strong>
        <div className="preview-header-actions">
          <button
            className={`pin-button ${pinned ? "is-active" : ""}`}
            type="button"
            aria-pressed={pinned}
            onClick={() => onPinnedChange(!pinned)}
          >
            <PushPin size={17} weight={pinned ? "fill" : "regular"} />
            固定预览
          </button>
          <IconButton label="关闭预览" onClick={onClose}>
            <X size={19} />
          </IconButton>
        </div>
      </header>

      <div className="preview-scroll">
        <div className={`preview-stage is-${item.kind}`} ref={mediaRef}>
          {item.kind === "video" ? (
            <>
              <video
                key={item.name}
                src={item.videoSrc}
                poster={item.src}
                controls
                playsInline
                onPlay={() => setPlaying(true)}
                onPause={() => setPlaying(false)}
              />
              <span className="playback-status" aria-live="polite">
                {playing ? <Pause size={14} weight="fill" /> : <Play size={14} weight="fill" />}
                {playing ? "正在播放" : "已暂停"}
              </span>
            </>
          ) : (
            <img src={item.src} alt={`${item.location}的媒体预览`} />
          )}
          <IconButton label="全屏查看" className="preview-fullscreen" onClick={enterFullscreen}>
            <ArrowsOut size={19} />
          </IconButton>
        </div>

        <div className="preview-navigation">
          <button type="button" onClick={onPrevious}>
            <ArrowLeft size={17} />
            上一项
          </button>
          <span>1 / 312</span>
          <button type="button" onClick={onNext}>
            下一项
            <ArrowRight size={17} />
          </button>
        </div>

        <section className="preview-details" aria-label="基本信息">
          <div className="preview-title">
            <strong>{item.name}</strong>
            <span className="type-chip">{item.kind === "video" ? "视频 · MP4" : "图片 · JPG"}</span>
          </div>
          <dl>
            <div>
              <dt>拍摄日期</dt>
              <dd>{item.date}:43</dd>
            </div>
            <div>
              <dt>文件位置</dt>
              <dd>旅行 / 日本 / {item.location.split(" / ")[1]}</dd>
            </div>
            <div>
              <dt>分辨率</dt>
              <dd>{item.dimensions}</dd>
            </div>
            <div>
              <dt>大小</dt>
              <dd>{item.size}</dd>
            </div>
            {item.kind === "video" && (
              <div>
                <dt>时长</dt>
                <dd>{item.duration}</dd>
              </div>
            )}
          </dl>
        </section>

        <div className={`pin-state ${pinned ? "is-pinned" : ""}`}>
          <PushPin size={20} weight={pinned ? "fill" : "regular"} />
          <div>
            <strong>{pinned ? "已固定预览" : "预览跟随选择"}</strong>
            <span>
              {pinned
                ? "单击只选择；双击缩略图才会切换当前预览。"
                : "点击其他缩略图即可更新这里的内容。"}
            </span>
          </div>
        </div>
      </div>
    </aside>
  );
}

export function App() {
  const [theme, setTheme] = useTheme();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [recursive, setRecursive] = useState(
    () => new URLSearchParams(window.location.search).get("recursive") === "1",
  );
  const [layout, setLayout] = useState("grid");
  const [sort, setSort] = useState("date-desc");
  const [filter, setFilter] = useState("all");
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(media[0]);
  const [previewItem, setPreviewItem] = useState(null);
  const [previewPinned, setPreviewPinned] = useState(false);
  const [previewWidth, setPreviewWidth] = useState(406);

  const visibleMedia = useMemo(() => {
    const items = media.filter((item) => {
      const matchesKind = filter === "all" || item.kind === filter;
      const text = `${item.name} ${item.location}`.toLowerCase();
      return matchesKind && text.includes(query.trim().toLowerCase());
    });

    return [...items].sort((a, b) => {
      if (sort === "name-asc") return a.name.localeCompare(b.name, "zh-CN");
      return b.date.localeCompare(a.date);
    });
  }, [filter, query, sort]);

  const resetControls = () => {
    setRecursive(false);
    setLayout("grid");
    setSort("date-desc");
    setFilter("all");
    setQuery("");
  };

  const movePreview = (direction) => {
    if (!previewItem || visibleMedia.length < 2) return;
    const index = visibleMedia.findIndex((item) => item.name === previewItem.name);
    const nextIndex = (index + direction + visibleMedia.length) % visibleMedia.length;
    const nextItem = visibleMedia[nextIndex];
    setSelected(nextItem);
    setPreviewItem(nextItem);
  };

  const beginPreviewResize = (event) => {
    event.preventDefault();
    const startX = event.clientX;
    const startWidth = previewWidth;

    const move = (moveEvent) => {
      const maxWidth = Math.min(620, window.innerWidth * 0.48);
      setPreviewWidth(Math.max(360, Math.min(maxWidth, startWidth + startX - moveEvent.clientX)));
    };
    const stop = () => {
      document.body.classList.remove("is-resizing-preview");
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", stop);
    };

    document.body.classList.add("is-resizing-preview");
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", stop);
  };

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main">
        跳到主要内容
      </a>
      <Sidebar open={drawerOpen} onClose={() => setDrawerOpen(false)} />

      <main id="main" className="main-content">
        <div className="scan-banner" role="status" aria-live="polite">
          <span className="scan-icon">
            <Info size={18} />
          </span>
          <span>正在扫描媒体库“家庭影像”。您可以浏览内容，扫描将在后台继续进行。</span>
          <strong>已发现 12,480 项 · 跳过 38 项</strong>
          <IconButton label="收起扫描状态">
            <X size={17} />
          </IconButton>
        </div>

        <header className="content-header">
          <div className="header-leading">
            <IconButton
              label="打开目录面板"
              className="drawer-trigger"
              onClick={() => setDrawerOpen(true)}
            >
              <SidebarSimple size={21} />
            </IconButton>
            <nav className="breadcrumbs" aria-label="面包屑">
              <a href="#main" aria-label="家庭影像根目录">
                <House size={18} />
              </a>
              <CaretRight size={14} />
              <a href="#main">旅行</a>
              <CaretRight size={14} />
              <a href="#main">日本</a>
              <CaretRight size={14} />
              <span aria-current="page">京都</span>
            </nav>
          </div>
          <div className="header-actions">
            <label className="search-box">
              <MagnifyingGlass size={19} />
              <span className="visually-hidden">搜索当前文件夹</span>
              <input
                type="search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="搜索当前文件夹"
              />
              <kbd>/</kbd>
            </label>
            <IconButton
              label={theme === "light" ? "切换到深色主题" : "切换到浅色主题"}
              onClick={() => setTheme(theme === "light" ? "dark" : "light")}
            >
              {theme === "light" ? <Moon size={20} /> : <Sun size={20} />}
            </IconButton>
            <IconButton label="高级筛选">
              <Funnel size={20} />
            </IconButton>
          </div>
        </header>

        <div
          className={`browse-workspace ${previewItem ? "has-preview" : ""}`}
          style={{ "--preview-width": `${previewWidth}px` }}
        >
          <div className="browse-column">
            <section className="toolbar" aria-label="浏览工具">
          <button
            className={`toggle-button ${recursive ? "is-active" : ""}`}
            type="button"
            aria-pressed={recursive}
            onClick={() => setRecursive((value) => !value)}
          >
            <span className="checkbox" aria-hidden="true">
              {recursive && <Check size={14} weight="bold" />}
            </span>
            <span className="toggle-label">包含子目录</span>
          </button>

          <div className="segmented" aria-label="布局">
            <IconButton
              label="规则网格"
              className={layout === "grid" ? "is-active" : ""}
              aria-pressed={layout === "grid"}
              onClick={() => setLayout("grid")}
            >
              <GridFour size={19} />
            </IconButton>
            <IconButton
              label="瀑布流预览"
              className={layout === "masonry" ? "is-active" : ""}
              aria-pressed={layout === "masonry"}
              onClick={() => setLayout("masonry")}
            >
              <Columns size={19} />
            </IconButton>
            <IconButton label="紧凑列表">
              <List size={19} />
            </IconButton>
          </div>

          <label className="select-control">
            <span>排序：</span>
            <select value={sort} onChange={(event) => setSort(event.target.value)}>
              <option value="date-desc">拍摄日期（从新到旧）</option>
              <option value="name-asc">文件名（自然顺序）</option>
            </select>
          </label>

          <label className="select-control filter-control">
            <span>筛选：</span>
            <select value={filter} onChange={(event) => setFilter(event.target.value)}>
              <option value="all">全部</option>
              <option value="image">仅图片</option>
              <option value="video">仅视频</option>
            </select>
          </label>

          <button className="reset-button" type="button" onClick={resetControls}>
            重置
          </button>
            </section>

            <section className="content-scroll" aria-label="京都的媒体">
          <div className="folder-grid" aria-label="子目录">
            {folders.map((folder) => (
              <button className="folder-card" type="button" key={folder.name}>
                <FolderSimple size={38} weight="fill" />
                <strong>{folder.name}</strong>
                <span>{folder.count} 项</span>
              </button>
            ))}
          </div>

          {visibleMedia.length ? (
            <div
              className={`media-grid ${layout === "masonry" ? "layout-masonry" : ""}`}
              role="list"
            >
              {visibleMedia.map((item) => {
                const isSelected = selected?.name === item.name;
                return (
                  <button
                    className={`media-card ${isSelected ? "is-selected" : ""}`}
                    type="button"
                    key={item.name}
                    role="listitem"
                    aria-label={
                      previewPinned
                        ? `选择 ${item.name}，双击切换固定预览，${item.location}`
                        : `预览 ${item.name}，${item.location}`
                    }
                    onClick={() => {
                      setSelected(item);
                      if (!previewItem || !previewPinned) setPreviewItem(item);
                    }}
                    onDoubleClick={() => {
                      if (previewPinned) setPreviewItem(item);
                    }}
                  >
                    <span className="media-frame">
                      <img src={item.src} alt="" />
                      {item.kind === "video" && (
                        <span className="video-badge" aria-hidden="true">
                          <Play size={12} weight="fill" />
                          {item.duration}
                        </span>
                      )}
                      {isSelected && (
                        <span className="selection-check" aria-hidden="true">
                          <Check size={14} weight="bold" />
                        </span>
                      )}
                    </span>
                    <span className="media-copy">
                      <strong>{item.name}</strong>
                      <span>{recursive ? item.location : item.date}</span>
                    </span>
                  </button>
                );
              })}
            </div>
          ) : (
            <div className="empty-state">
              <MagnifyingGlass size={30} />
              <h2>没有匹配的媒体</h2>
              <p>请清除搜索词或调整筛选条件。</p>
              <button type="button" onClick={resetControls}>
                清除筛选
              </button>
            </div>
          )}
            </section>
          </div>

          {previewItem && (
            <PreviewPane
              item={previewItem}
              pinned={previewPinned}
              onPinnedChange={setPreviewPinned}
              onClose={() => {
                setPreviewItem(null);
                setPreviewPinned(false);
              }}
              onPrevious={() => movePreview(-1)}
              onNext={() => movePreview(1)}
              onResizeStart={beginPreviewResize}
            />
          )}
        </div>
      </main>
    </div>
  );
}
