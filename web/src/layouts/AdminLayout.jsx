import { useEffect, useState, useRef, useMemo } from "react";
import { NavLink, Outlet, useNavigate, useLocation } from "react-router-dom";
import { clearAuthToken, isLoggedIn, adminApi, userApi } from "../lib/api";
import { DEFAULT_BRAND, SETTINGS_UPDATED_EVENT, normalizeBrand, setAppTitle } from "../lib/brand";

const NAV_GROUPS = [
  {
    label: "运营",
    id: "ops",
    minRole: "user",
    items: [
      { path: "/admin/dashboard", label: "运营总览", icon: "◎", minRole: "user" },
      { path: "/admin/campaigns", label: "活动管理", icon: "▦", minRole: "user" },
      { path: "/admin/cdks", label: "CDK 库存", icon: "◇", minRole: "user" },
      { path: "/admin/nodes", label: "分发网络", icon: "⎈", minRole: "user" },
      { path: "/admin/projects", label: "项目管理", icon: "◫", minRole: "user" },
    ],
  },
  {
    label: "数据",
    id: "data",
    minRole: "user",
    items: [
      { path: "/admin/claims", label: "领取记录", icon: "◷", minRole: "user" },
      { path: "/admin/analytics", label: "数据分析", icon: "↗", minRole: "user" },
      { path: "/admin/logs", label: "系统日志", icon: "☰", minRole: "admin" },
    ],
  },
  {
    label: "安全",
    id: "security",
    minRole: "admin",
    items: [
      { path: "/admin/risk", label: "风控中心", icon: "◈", minRole: "admin" },
      { path: "/admin/captcha", label: "验证码配置", icon: "▣", minRole: "admin" },
    ],
  },
  {
    label: "系统",
    id: "system",
    minRole: "admin",
    items: [
      { path: "/admin/settings", label: "系统设置", icon: "⚙", minRole: "admin" },
    ],
  },
];

const ROLE_LEVEL = { user: 0, viewer: 1, operator: 2, admin: 3 };

function hasPermission(userRole, minRole) {
  return (ROLE_LEVEL[userRole] || 0) >= (ROLE_LEVEL[minRole] || 0);
}

// 面包屑映射
const BREADCRUMB_MAP = {
  "/admin/dashboard": ["运营总览"],
  "/admin/campaigns": ["运营", "活动管理"],
  "/admin/cdks": ["运营", "CDK 库存"],
  "/admin/nodes": ["运营", "分发网络"],
  "/admin/projects": ["运营", "项目管理"],
  "/admin/projects/create": ["运营", "项目管理", "新建项目"],
  "/admin/claims": ["数据", "领取记录"],
  "/admin/analytics": ["数据", "数据分析"],
  "/admin/logs": ["数据", "系统日志"],
  "/admin/risk": ["安全", "风控中心"],
  "/admin/captcha": ["安全", "验证码配置"],
  "/admin/settings": ["系统", "系统设置"],
};

// 全局搜索候选项
const SEARCH_ITEMS = NAV_GROUPS.flatMap((g) =>
  g.items.map((item) => ({ ...item, group: g.label }))
);

export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [collapsedGroups, setCollapsedGroups] = useState({});
  const [user, setUser] = useState(null);
  const [alertCount, setAlertCount] = useState(0);
  const [settings, setSettings] = useState(DEFAULT_BRAND);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const searchRef = useRef(null);
  const sidebarRef = useRef(null);

  useEffect(() => {
    if (!isLoggedIn()) {
      navigate(`/login?returnUrl=${encodeURIComponent(window.location.pathname)}`, { replace: true });
    }
  }, [navigate]);

  // 获取用户信息
  useEffect(() => {
    if (!isLoggedIn()) return;
    userApi.getMe().then((data) => {
      setUser(data);
    }).catch(() => setUser({ username: "管理员", role: "admin" }));
  }, []);

  // 用户角色
  const userRole = user?.role || user?.user?.role || "admin";

  // 获取异常提醒数量
  useEffect(() => {
    if (!isLoggedIn()) return;
    adminApi.dashboard().then((data) => {
      setAlertCount((data?.alerts || []).length);
    }).catch(() => {});
  }, []);

  // 获取系统设置（品牌名称）
  useEffect(() => {
    let alive = true;
    async function loadSettings() {
      if (!isLoggedIn()) {
        setAppTitle();
        return;
      }
      try {
        const data = normalizeBrand(await adminApi.settings());
        if (!alive) return;
        setSettings(data);
        setAppTitle(data);
      } catch {
        setAppTitle(DEFAULT_BRAND);
      }
    }
    function handleSettingsUpdated(event) {
      const data = normalizeBrand(event.detail);
      setSettings(data);
      setAppTitle(data);
    }
    loadSettings();
    window.addEventListener(SETTINGS_UPDATED_EVENT, handleSettingsUpdated);
    window.addEventListener("focus", loadSettings);
    return () => {
      alive = false;
      window.removeEventListener(SETTINGS_UPDATED_EVENT, handleSettingsUpdated);
      window.removeEventListener("focus", loadSettings);
    };
  }, []);

  // 路由变化时关闭移动端侧边栏
  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  // 全局快捷键：Ctrl+K 打开搜索
  useEffect(() => {
    function handleKeyDown(e) {
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        setSearchOpen((prev) => !prev);
      }
      if (e.key === "Escape") {
        setSearchOpen(false);
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  // 搜索打开时聚焦输入框
  useEffect(() => {
    if (searchOpen && searchRef.current) {
      searchRef.current.focus();
    }
  }, [searchOpen]);

  // 点击外部关闭移动端侧边栏
  useEffect(() => {
    if (!mobileOpen) return;
    function handleClick(e) {
      if (sidebarRef.current && !sidebarRef.current.contains(e.target)) {
        setMobileOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [mobileOpen]);

  // 搜索过滤
  const searchResults = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return SEARCH_ITEMS;
    return SEARCH_ITEMS.filter(
      (item) =>
        item.label.toLowerCase().includes(q) ||
        item.group.toLowerCase().includes(q) ||
        item.path.toLowerCase().includes(q)
    );
  }, [searchQuery]);

  // 面包屑
  const breadcrumbs = useMemo(() => {
    const path = location.pathname;
    if (BREADCRUMB_MAP[path]) return BREADCRUMB_MAP[path];
    if (path.startsWith("/admin/nodes/")) return ["运营", "分发网络", "节点详情"];
    if (path.startsWith("/admin/projects/")) return ["运营", "项目管理", "项目详情"];
    return ["运营总览"];
  }, [location.pathname]);

  if (!isLoggedIn()) return null;

  function handleLogout() {
    clearAuthToken();
    navigate("/login");
  }

  function toggleGroup(groupId) {
    setCollapsedGroups((prev) => ({ ...prev, [groupId]: !prev[groupId] }));
  }

  function handleSearchSelect(item) {
    navigate(item.path);
    setSearchOpen(false);
    setSearchQuery("");
  }

  return (
    <div className={`admin-shell ${collapsed ? "admin-shell--collapsed" : ""}`}>
      {/* 移动端遮罩 */}
      {mobileOpen && <div className="admin-mobile-backdrop" onClick={() => setMobileOpen(false)} />}

      {/* 侧边栏 */}
      <aside
        ref={sidebarRef}
        className={`admin-sidebar ${collapsed ? "admin-sidebar--collapsed" : ""} ${mobileOpen ? "admin-sidebar--mobile-open" : ""}`}
      >
        {/* 品牌区域 */}
        <div className="admin-sidebar__brand">
          <img src="/logo.png" alt="Logo" width="44" height="44" className="admin-sidebar__logo" style={{ objectFit: 'cover', background: 'transparent', boxShadow: 'none' }} />
          {!collapsed && (
            <div className="admin-sidebar__brand-text">
              <h1>{settings?.systemName || DEFAULT_BRAND.systemName}</h1>
              <span>{settings?.brandEnglishName || DEFAULT_BRAND.brandEnglishName}</span>
            </div>
          )}
          <button
            className="admin-sidebar__toggle btn--text"
            onClick={() => setCollapsed(!collapsed)}
            title={collapsed ? "展开侧边栏" : "收起侧边栏"}
            aria-label={collapsed ? "展开侧边栏" : "收起侧边栏"}
          >
            {collapsed ? "»" : "«"}
          </button>
        </div>

        {/* 导航区域 */}
        <nav className="admin-sidebar__nav" aria-label="主导航">
          {NAV_GROUPS.filter((group) => hasPermission(userRole, group.minRole)).map((group) => (
            <div className="admin-nav-group" key={group.id}>
              {!collapsed && (
                <button
                  className="admin-nav-group__label"
                  onClick={() => toggleGroup(group.id)}
                  aria-expanded={!collapsedGroups[group.id]}
                >
                  <span>{group.label}</span>
                  <span className={`admin-nav-group__arrow ${collapsedGroups[group.id] ? "admin-nav-group__arrow--collapsed" : ""}`}>
                    ▾
                  </span>
                </button>
              )}
              {!collapsedGroups[group.id] && group.items.filter((item) => hasPermission(userRole, item.minRole)).map((item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  end={item.path === "/admin/dashboard"}
                  title={collapsed ? item.label : undefined}
                  className={({ isActive }) =>
                    `admin-nav-item ${isActive ? "admin-nav-item--active" : ""}`
                  }
                >
                  <span className="admin-nav-item__icon">{item.icon}</span>
                  {!collapsed && <span className="admin-nav-item__label">{item.label}</span>}
                  {item.path === "/admin/risk" && alertCount > 0 && !collapsed && (
                    <span className="admin-nav-item__badge">{alertCount > 9 ? "9+" : alertCount}</span>
                  )}
                  {item.path === "/admin/risk" && alertCount > 0 && collapsed && (
                    <span className="admin-nav-item__badge-dot" />
                  )}
                </NavLink>
              ))}
            </div>
          ))}
        </nav>

        {/* 用户信息 & 退出 */}
        <div className="admin-sidebar__footer">
          {!collapsed && user && (
            <div className="admin-sidebar__user">
              {user.avatar || user.user?.avatar ? (
                <img src={user.avatar || user.user?.avatar} alt="Avatar" className="admin-sidebar__avatar" style={{ objectFit: 'cover', background: 'transparent', padding: 0 }} />
              ) : (
                <div className="admin-sidebar__avatar">
                  {(user.username || user.user?.username || "U").charAt(0).toUpperCase()}
                </div>
              )}
              <div className="admin-sidebar__user-info">
                <strong>{user.username || user.user?.username || "管理员"}</strong>
                <small>{userRole === "admin" ? "管理员" : "用户"}</small>
              </div>
            </div>
          )}
          <button className="admin-nav-item admin-nav-item--logout" onClick={handleLogout} title="退出系统">
            <span className="admin-nav-item__icon">⏏</span>
            {!collapsed && <span className="admin-nav-item__label">退出系统</span>}
          </button>
        </div>
      </aside>

      {/* 主内容区 */}
      <div className="admin-main">
        {/* 顶部导航栏 */}
        <header className="admin-topbar">
          <div className="admin-topbar__left">
            <button
              className="admin-topbar__hamburger"
              onClick={() => setMobileOpen(!mobileOpen)}
              aria-label="打开导航菜单"
            >
              <span className="hamburger-line" />
              <span className="hamburger-line" />
              <span className="hamburger-line" />
            </button>

            <nav className="admin-breadcrumb" aria-label="面包屑导航">
              <span className="admin-breadcrumb__home" onClick={() => navigate("/admin/dashboard")}>
                ⌂
              </span>
              {breadcrumbs.map((crumb, index) => (
                <span key={index} className="admin-breadcrumb__item">
                  <span className="admin-breadcrumb__sep">/</span>
                  <span className={index === breadcrumbs.length - 1 ? "admin-breadcrumb__current" : ""}>
                    {crumb}
                  </span>
                </span>
              ))}
            </nav>
          </div>

          <div className="admin-topbar__right">
            <button
              className="admin-topbar__search-btn"
              onClick={() => setSearchOpen(true)}
              title="搜索导航 (Ctrl+K)"
            >
              <span className="admin-topbar__search-icon">⌕</span>
              <span className="admin-topbar__search-text">搜索...</span>
              <kbd className="admin-topbar__search-kbd">⌘K</kbd>
            </button>

            <button
              className="admin-topbar__notification"
              onClick={() => navigate("/admin/risk")}
              title={`${alertCount} 条异常提醒`}
              aria-label={`${alertCount} 条异常提醒`}
            >
              <span className="admin-topbar__bell">🔔</span>
              {alertCount > 0 && (
                <span className="admin-topbar__notification-badge">
                  {alertCount > 9 ? "9+" : alertCount}
                </span>
              )}
            </button>

            <div className="admin-topbar__user-mobile">
              {user?.avatar || user?.user?.avatar ? (
                <img src={user?.avatar || user?.user?.avatar} alt="Avatar" className="admin-topbar__avatar" style={{ objectFit: 'cover', background: 'transparent', padding: 0 }} />
              ) : (
                <div className="admin-topbar__avatar">
                  {(user?.username || user?.user?.username || "U").charAt(0).toUpperCase()}
                </div>
              )}
            </div>
          </div>
        </header>

        {/* 页面内容 */}
        <main className="admin-content">
          <Outlet />
        </main>
      </div>

      {/* 全局搜索弹窗 */}
      {searchOpen && (
        <div className="admin-search-overlay" onClick={() => setSearchOpen(false)}>
          <div className="admin-search-modal" onClick={(e) => e.stopPropagation()}>
            <div className="admin-search-modal__header">
              <span className="admin-search-modal__icon">⌕</span>
              <input
                ref={searchRef}
                className="admin-search-modal__input"
                placeholder="搜索页面、功能..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && searchResults.length > 0) {
                    handleSearchSelect(searchResults[0]);
                  }
                  if (e.key === "Escape") {
                    setSearchOpen(false);
                  }
                }}
              />
              <kbd className="admin-search-modal__esc">ESC</kbd>
            </div>
            <div className="admin-search-modal__results">
              {searchResults.length === 0 ? (
                <div className="admin-search-modal__empty">没有匹配的结果</div>
              ) : (
                searchResults.map((item) => (
                  <button
                    key={item.path}
                    className={`admin-search-modal__item ${location.pathname === item.path ? "admin-search-modal__item--active" : ""}`}
                    onClick={() => handleSearchSelect(item)}
                  >
                    <span className="admin-search-modal__item-icon">{item.icon}</span>
                    <div className="admin-search-modal__item-text">
                      <strong>{item.label}</strong>
                      <small>{item.group} · {item.path}</small>
                    </div>
                    <span className="admin-search-modal__item-arrow">→</span>
                  </button>
                ))
              )}
            </div>
            <div className="admin-search-modal__footer">
              <span>↑↓ 导航</span>
              <span>↵ 打开</span>
              <span>ESC 关闭</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
