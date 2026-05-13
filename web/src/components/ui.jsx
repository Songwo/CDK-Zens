import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";

export function PageHeader({ eyebrow, title, description, actions }) {
  return (
    <header className="page-header">
      <div>
        {eyebrow && <span className="page-header__eyebrow">{eyebrow}</span>}
        <h1>{title}</h1>
        {description && <p>{description}</p>}
      </div>
      {actions && <div className="page-header__actions">{actions}</div>}
    </header>
  );
}

export function StatCard({ label, value, hint, tone = "default", compact = false }) {
  return (
    <div className={`stat-card stat-card--${tone} ${compact ? "stat-card--compact" : ""}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      {hint && <small>{hint}</small>}
    </div>
  );
}

export function MiniStatCard(props) {
  return <StatCard {...props} compact />;
}

const STATUS_LABELS = {
  active: "进行中",
  paused: "已暂停",
  disabled: "已暂停",
  upcoming: "未开始",
  ended: "已结束",
  soldout: "库存耗尽",
  unused: "未领取",
  claimed: "已领取",
  frozen: "已冻结",
  invalid: "已失效",
  success: "成功",
  failed: "失败",
  blocked: "已拦截",
  draft: "未开始",
  exhausted: "库存耗尽",
  archived: "已归档",
};

export function StatusBadge({ status }) {
  return <span className={`status-badge status-badge--${status}`}>{STATUS_LABELS[status] || status || "--"}</span>;
}

export function ProgressBar({ claimed = 0, total = 0, remaining = 0 }) {
  const pct = total > 0 ? Math.min(100, Math.round((claimed / total) * 100)) : 0;
  const low = total > 0 && remaining > 0 && remaining / total < 0.1;
  const empty = total > 0 && remaining <= 0;
  return (
    <div className="progress-wrap">
      <div className="progress-top"><span>{claimed} / {total}</span><span>{pct}%</span></div>
      <div className="progress-line"><div className={`progress-line__fill ${low ? "is-low" : ""} ${empty ? "is-empty" : ""}`} style={{ width: `${pct}%` }} /></div>
      <div className="progress-bottom">剩余 {remaining}</div>
    </div>
  );
}

export function EmptyState({ title = "暂无数据", description, action }) {
  return (
    <div className="empty-panel">
      <strong>{title}</strong>
      {description && <p>{description}</p>}
      {action}
    </div>
  );
}

export function SearchInput({ value, onChange, placeholder = "搜索" }) {
  return <input className="search-input" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} />;
}

export function FilterSelect({ value, onChange, children }) {
  return <select className="styled-select" value={value} onChange={(e) => onChange(e.target.value)}>{children}</select>;
}

export function AlertPanel({ alerts = [], onViewAll }) {
  const visible = alerts.slice(0, 4);
  return (
    <div className="alert-panel">
      {visible.length === 0 ? (
        <EmptyState title="暂无异常，系统运行良好" />
      ) : (
        <div className="alert-list compact">
          {visible.map((a, index) => (
            <div className={`alert-item alert-item--${a.level || "info"}`} key={`${a.type || "alert"}-${index}`}>
              <div>
                <strong>{a.title}</strong>
                <p>{a.message}</p>
              </div>
              <button className="btn btn--text" onClick={onViewAll}>处理</button>
            </div>
          ))}
        </div>
      )}
      {alerts.length > 4 && <button className="btn btn--text alert-panel__more" onClick={onViewAll}>查看全部异常</button>}
    </div>
  );
}

export function QuickActions({ actions = [] }) {
  return (
    <div className="quick-actions">
      {actions.map((action) => (
        <button key={action.label} className="quick-action" onClick={action.onClick}>
          <span>{action.icon}</span>
          <strong>{action.label}</strong>
        </button>
      ))}
    </div>
  );
}

export function SystemHealthCard({ health = {}, captchaConfigured = false, captchaRequired = false }) {
  const items = [
    ["应用服务", health.app || "ok", "success"],
    ["JSON 存储", health.jsonStore || "ok", "success"],
    ["Redis", health.redis === "enabled" ? "已启用" : health.redis === "error" ? "异常" : "未启用", health.redis === "error" ? "danger" : "info"],
    ["RabbitMQ", health.rabbitmq === "enabled" ? "已启用" : health.rabbitmq === "error" ? "异常" : "未启用", health.rabbitmq === "error" ? "danger" : "info"],
    ["hCaptcha", captchaConfigured ? "已配置" : captchaRequired ? "未配置" : "未配置", captchaConfigured ? "success" : captchaRequired ? "danger" : "info"],
  ];
  return (
    <div className="health-list">
      {items.map(([label, value, tone]) => (
        <div className="health-row" key={label}>
          <span className={`health-dot health-dot--${tone}`} />
          <span>{label}</span>
          <strong>{value}</strong>
        </div>
      ))}
    </div>
  );
}

export function Toast({ message, type = "success", onClose }) {
  useEffect(() => {
    const t = setTimeout(onClose, 2400);
    return () => clearTimeout(t);
  }, [onClose]);
  return <div className={`toast toast--${type}`}>{message}</div>;
}

export function ConfirmDialog({ open, title, description, confirmText = "确认", onConfirm, onCancel, danger }) {
  return (
    <Modal open={open} title={title} onCancel={onCancel} danger={danger}>
      <div className="modal-body">{description}</div>
      <div className="modal-footer">
        <button className="modal-btn modal-btn--cancel" onClick={onCancel}>取消</button>
        <button className={`modal-btn ${danger ? "modal-btn--danger" : "modal-btn--primary"}`} onClick={onConfirm}>{confirmText}</button>
      </div>
    </Modal>
  );
}

export function Modal({ open, title, children, onCancel, danger }) {
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKey = (e) => {
      if (e.key === "Escape") onCancel?.();
    };
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = prev;
      window.removeEventListener("keydown", onKey);
    };
  }, [open, onCancel]);
  if (!open) return null;
  return createPortal(
    <div className="modal-overlay">
      <div className="modal-backdrop" onClick={() => { if (!danger) onCancel?.(); }} />
      <div className="modal-container" role="dialog" aria-modal="true">
        <div className="modal-header"><h2 className="modal-title">{title}</h2></div>
        {children}
      </div>
    </div>,
    document.body
  );
}

export function CopyButton({ value, children = "复制" }) {
  const [copied, setCopied] = useState(false);
  async function copy() {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1600);
  }
  return <button className="btn btn--text" onClick={copy}>{copied ? "已复制" : children}</button>;
}

export function DataTable({ columns, rows, loading, emptyText = "暂无数据", pageSize = 10 }) {
  const [page, setPage] = useState(1);
  const totalPages = Math.max(1, Math.ceil((rows?.length || 0) / pageSize));
  const pageRows = useMemo(() => (rows || []).slice((page - 1) * pageSize, page * pageSize), [rows, page, pageSize]);
  useEffect(() => setPage(1), [rows]);

  if (loading) return <EmptyState title="加载中" description="正在同步数据..." />;
  if (!rows || rows.length === 0) return <EmptyState title={emptyText} />;

  return (
    <>
      <div className="table-scroll">
        <table className="data-table">
          <thead>
            <tr>{columns.map((c) => <th key={c.key}>{c.title}</th>)}</tr>
          </thead>
          <tbody>
            {pageRows.map((row) => (
              <tr key={row.id || row.key}>
                {columns.map((c) => <td key={c.key}>{c.render ? c.render(row) : row[c.key]}</td>)}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {totalPages > 1 && (
        <div className="pagination">
          <button className="btn btn--secondary" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>上一页</button>
          <span>{page} / {totalPages}</span>
          <button className="btn btn--secondary" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>下一页</button>
        </div>
      )}
    </>
  );
}
