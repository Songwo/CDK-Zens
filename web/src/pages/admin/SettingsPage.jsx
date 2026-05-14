import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { adminApi } from "../../lib/api";
import { DEFAULT_BRAND, normalizeBrand, notifySettingsUpdated, setAppTitle } from "../../lib/brand";
import { ConfirmDialog, DataTable, FilterSelect, Modal, PageHeader, StatCard, StatusBadge, Toast } from "../../components/ui";

function time(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN");
}

export default function SettingsPage() {
  const location = useLocation();
  const [settings, setSettings] = useState(null);
  const [health, setHealth] = useState(null);
  const [admins, setAdmins] = useState([]);
  const [adminForm, setAdminForm] = useState(null);
  const [restoreName, setRestoreName] = useState("");
  const [confirmRestore, setConfirmRestore] = useState(false);
  const [toast, setToast] = useState(null);
  const [savingSettings, setSavingSettings] = useState(false);
  const mode = location.pathname.endsWith("/brand") ? "品牌配置" : location.pathname.endsWith("/storage") ? "存储与队列" : location.pathname.endsWith("/admins") ? "管理员" : "基础设置";

  async function reload() {
    const [s, h, a] = await Promise.all([adminApi.settings(), adminApi.health(), adminApi.admins()]);
    const next = normalizeBrand(s);
    setSettings(next);
    setAppTitle(next);
    notifySettingsUpdated(next);
    setHealth(h);
    setAdmins(a || []);
  }

  useEffect(() => { reload().catch((err) => setToast({ type: "error", message: err.message })); }, []);

  async function saveSettings(e) {
    e.preventDefault();
    if (savingSettings) return;
    setSavingSettings(true);
    try {
      const next = normalizeBrand(await adminApi.updateSettings(settings));
      setSettings(next);
      notifySettingsUpdated(next);
      setAppTitle(next);
      setToast({ message: "系统设置已保存" });
    } catch (err) {
      setToast({ type: "error", message: err.message });
    } finally {
      setSavingSettings(false);
    }
  }

  async function saveAdmin(e) {
    e.preventDefault();
    try {
      if (adminForm.id) await adminApi.updateAdmin(adminForm.id, adminForm);
      else await adminApi.createAdmin(adminForm);
      setAdminForm(null);
      setToast({ message: "管理员已保存" });
      reload();
    } catch (err) {
      setToast({ type: "error", message: err.message });
    }
  }

  async function backup() {
    try {
      const res = await adminApi.backup();
      setToast({ message: `备份已生成：${res.filename}` });
      if (res.path) window.open(res.path, "_blank");
    } catch (err) {
      setToast({ type: "error", message: err.message });
    }
  }

  async function restore() {
    try {
      await adminApi.restore({ filename: restoreName });
      setToast({ message: "恢复成功，已重新加载状态" });
      setConfirmRestore(false);
      reload();
    } catch (err) {
      setToast({ type: "error", message: err.message });
    }
  }

  return (
    <div className="admin-page">
      {toast && <Toast {...toast} onClose={() => setToast(null)} />}
      <PageHeader eyebrow="System" title={mode} description="管理品牌、公网地址、存储状态、管理员和数据备份恢复。" actions={<div className="btn-group"><button className="btn btn--primary" onClick={saveSettings} disabled={savingSettings}>{savingSettings ? "保存中..." : "保存设置"}</button><button className="btn btn--secondary" onClick={backup}>数据备份</button></div>} />
      <section className="stats-grid">
        <StatCard label="系统名称" value={settings?.systemName || "--"} hint={settings?.brandEnglishName} tone="gold" />
        <StatCard label="Logo" value={settings?.logoText || DEFAULT_BRAND.logoText} />
        <StatCard label="JSON 存储" value={health?.jsonStore || "--"} tone="success" />
        <StatCard label="Redis / RabbitMQ" value={`${health?.redis || "--"} / ${health?.rabbitmq || "--"}`} />
      </section>
      <form className="panel modal-form" onSubmit={saveSettings}>
        <div className="panel__header"><div><h2>基础与品牌配置</h2><p>公开领取页和后台品牌信息会读取这些配置。</p></div></div>
        <input className="styled-input" value={settings?.systemName || ""} onChange={(e) => setSettings({ ...settings, systemName: e.target.value })} placeholder="系统名称" />
        <input className="styled-input" value={settings?.brandName || ""} onChange={(e) => setSettings({ ...settings, brandName: e.target.value })} placeholder="中文品牌名" />
        <input className="styled-input" value={settings?.brandEnglishName || ""} onChange={(e) => setSettings({ ...settings, brandEnglishName: e.target.value })} placeholder="英文品牌名" />
        <input className="styled-input" value={settings?.logoText || ""} onChange={(e) => setSettings({ ...settings, logoText: e.target.value })} placeholder="Logo 文案" />
        <input className="styled-input" value={settings?.publicBaseURL || ""} onChange={(e) => setSettings({ ...settings, publicBaseURL: e.target.value })} placeholder="publicBaseURL" />
        <FilterSelect value={settings?.storageMode || "json"} onChange={(v) => setSettings({ ...settings, storageMode: v })}><option value="json">JSON 本地存储</option><option value="redis">Redis 增强</option></FilterSelect>
      </form>
      <section className="panel">
        <div className="panel__header"><div><h2>系统健康</h2><p>中间件为可选能力，默认 JSON 模式完整可用。</p></div></div>
        <div className="detail-lines">
          <p><b>状态文件</b><span>{health?.stateFile || "--"}</span></p>
          <p><b>存储模式</b><span>{health?.storageMode || "json"}</span></p>
          <p><b>Redis 状态</b><span><StatusBadge status={health?.redis === "enabled" ? "active" : "paused"} /></span></p>
          <p><b>RabbitMQ 状态</b><span><StatusBadge status={health?.rabbitmq === "enabled" ? "active" : "paused"} /></span></p>
        </div>
      </section>
      <section className="panel">
        <div className="panel__header"><div><h2>管理员</h2><p>维护后台登录账号。</p></div><button className="btn btn--primary" onClick={() => setAdminForm({ username: "", password: "", role: "admin" })}>新增管理员</button></div>
        <DataTable rows={admins} columns={[
          { key: "username", title: "用户名" },
          { key: "role", title: "角色" },
          { key: "createdAt", title: "创建时间", render: (r) => time(r.createdAt) },
          { key: "actions", title: "操作", render: (r) => <div className="row-actions"><button className="btn btn--text" onClick={() => setAdminForm({ ...r, password: "" })}>编辑</button><button className="btn btn--text danger" onClick={() => adminApi.deleteAdmin(r.id).then(reload)}>删除</button></div> },
        ]} />
      </section>
      <section className="panel modal-form">
        <div className="panel__header"><div><h2>数据恢复</h2><p>填写已有备份文件名，恢复前后端会自动先备份当前数据。</p></div></div>
        <input className="styled-input" value={restoreName} onChange={(e) => setRestoreName(e.target.value)} placeholder="state-YYYYMMDD-HHMMSS.json" />
        <button className="btn btn--secondary" disabled={!restoreName} onClick={() => setConfirmRestore(true)}>恢复备份</button>
      </section>
      <Modal open={!!adminForm} title="管理员" onCancel={() => setAdminForm(null)}>
        <form className="modal-form" onSubmit={saveAdmin}>
          <input className="styled-input" value={adminForm?.username || ""} onChange={(e) => setAdminForm({ ...adminForm, username: e.target.value })} placeholder="用户名" />
          <input className="styled-input" type="password" value={adminForm?.password || ""} onChange={(e) => setAdminForm({ ...adminForm, password: e.target.value })} placeholder={adminForm?.id ? "留空则不修改密码" : "密码"} />
          <input className="styled-input" value={adminForm?.role || "admin"} onChange={(e) => setAdminForm({ ...adminForm, role: e.target.value })} placeholder="角色" />
          <div className="modal-footer"><button className="modal-btn modal-btn--primary">保存</button></div>
        </form>
      </Modal>
      <ConfirmDialog open={confirmRestore} danger title="二次确认恢复数据" description="恢复会覆盖当前 state.json，系统会先自动备份当前数据。确认继续？" onCancel={() => setConfirmRestore(false)} onConfirm={restore} />
    </div>
  );
}
