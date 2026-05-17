import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { adminApi } from "../../lib/api";
import CdkImportModal from "../../components/CdkImportModal";
import { notifyOnboardingRefresh } from "../../components/DistributionGuide";
import { ConfirmDialog, CopyButton, DataTable, EmptyState, FilterSelect, Modal, PageHeader, SearchInput, StatCard, StatusBadge, Toast } from "../../components/ui";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";

function items(res) {
  return res?.items || res || [];
}

function fmt(n) {
  return new Intl.NumberFormat("zh-CN").format(Number(n || 0));
}

function time(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN");
}

function useToast() {
  const [toast, setToast] = useState(null);
  return [toast, (message, type = "success") => setToast(message ? { message, type } : null)];
}

export function FeaturePage({ feature }) {
  return <AnalyticsPage feature={feature} />;
}

export function CDKInventoryPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [rows, setRows] = useState([]);
  const [projects, setProjects] = useState([]);
  const [campaigns, setCampaigns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("all");
  const [projectId, setProjectId] = useState("");
  const [campaignId, setCampaignId] = useState("");
  const [selected, setSelected] = useState([]);
  const [importOpen, setImportOpen] = useState(false);
  const [confirm, setConfirm] = useState(null);
  const [toast, showToast] = useToast();
  const [tasks, setTasks] = useState([]);
  const mode = location.pathname.endsWith("/import") ? "import" : location.pathname.endsWith("/status") ? "status" : location.pathname.endsWith("/export") ? "export" : "overview";

  async function reload() {
    setLoading(true);
    try {
      const [cdks, projectData, camps, exportTasks] = await Promise.all([
        adminApi.listCDKs({ keyword, status, projectId, campaignId, pageSize: 500 }),
        adminApi.listProjects({ pageSize: 500 }),
        adminApi.listCampaigns({ pageSize: 500 }),
        adminApi.exportTasks({ type: "cdks", pageSize: 50 }).catch(() => ({ items: [] })),
      ]);
      setRows(items(cdks));
      setProjects(items(projectData));
      setCampaigns(items(camps));
      setTasks(items(exportTasks));
    } catch (err) {
      showToast(err.message, "error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { reload(); }, [keyword, status, projectId, campaignId]);
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    if (location.pathname.endsWith("/import")) {
      setCampaignId(params.get("campaignId") || campaignId);
      setProjectId(params.get("projectId") || projectId);
      setImportOpen(true);
    }
    if (location.pathname.endsWith("/status")) setStatus("unused");
  }, [location.pathname, location.search]);

  const stats = useMemo(() => ({
    total: rows.length,
    unused: rows.filter((r) => r.status === "unused").length,
    claimed: rows.filter((r) => r.status === "claimed").length,
    frozen: rows.filter((r) => r.status === "frozen").length,
    invalid: rows.filter((r) => r.status === "invalid").length,
  }), [rows]);
  const visibleCampaigns = useMemo(() => projectId ? campaigns.filter((c) => c.projectId === projectId) : campaigns, [campaigns, projectId]);

  async function run(action, row) {
    try {
      if (action === "freeze") await adminApi.freezeCDK(row.id);
      if (action === "unfreeze") await adminApi.unfreezeCDK(row.id);
      if (action === "invalidate") await adminApi.invalidateCDK(row.id);
      if (action === "delete") await adminApi.deleteCDK(row.id);
      if (action === "batchFreeze") await adminApi.batchFreezeCDKs(selected);
      if (action === "batchInvalid") await adminApi.batchInvalidateCDKs(selected);
      if (action === "export") {
        const task = await adminApi.exportCDKs({ keyword, status, campaignId });
        window.open(task.filePath, "_blank");
      }
      showToast("操作成功");
      setSelected([]);
      reload();
    } catch (err) {
      showToast(err.message, "error");
    } finally {
      setConfirm(null);
    }
  }

  return (
    <div className="admin-page">
      {toast && <Toast {...toast} onClose={() => showToast(null)} />}
      <PageHeader eyebrow="Inventory" title={mode === "import" ? "批量导入" : mode === "status" ? "状态管理" : mode === "export" ? "导出任务" : "库存总览"} description="CDK 按「一活动一池」隔离，所有导入和管理操作必须先选定具体活动。" actions={<div className="btn-group"><button className="btn btn--primary" onClick={() => navigate("/admin/campaigns")}>前往活动管理</button><button className="btn btn--secondary" onClick={() => run("export")} disabled={!campaignId}>导出 CSV</button></div>} />
      <section className="stats-grid">
        <StatCard label="总量" value={fmt(stats.total)} />
        <StatCard label="未领取" value={fmt(stats.unused)} tone="success" />
        <StatCard label="已领取" value={fmt(stats.claimed)} tone="gold" />
        <StatCard label="冻结 / 失效" value={`${fmt(stats.frozen)} / ${fmt(stats.invalid)}`} tone="warning" />
      </section>
      <div className="toolbar panel-toolbar">
        <FilterSelect value={projectId} onChange={(v) => { setProjectId(v); setCampaignId(""); }}><option value="">全部项目</option>{projects.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</FilterSelect>
        <FilterSelect value={campaignId} onChange={setCampaignId}><option value="">请选择活动</option>{visibleCampaigns.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</FilterSelect>
        <FilterSelect value={status} onChange={setStatus}><option value="all">全部状态</option><option value="unused">未领取</option><option value="claimed">已领取</option><option value="frozen">已冻结</option><option value="invalid">已失效</option></FilterSelect>
        <SearchInput value={keyword} onChange={setKeyword} placeholder="搜索 CDK" />
        <button className="btn btn--secondary" disabled={!selected.length} onClick={() => setConfirm({ action: "batchFreeze" })}>批量冻结</button>
        <button className="btn btn--secondary" disabled={!selected.length} onClick={() => setConfirm({ action: "batchInvalid" })}>批量作废</button>
      </div>
      {!campaignId ? (
        <EmptyState
          title="请先选择一个活动查看其 CDK 库"
          description="按「一活动一池」策略，CDK 严格归属于某个活动。请从上方筛选器选择活动，或前往活动管理页打开导入入口。"
          action={<button className="btn btn--primary" onClick={() => navigate("/admin/campaigns")}>前往活动列表</button>}
        />
      ) : (
        <section className="panel">
          <DataTable loading={loading} rows={rows} pageSize={12} columns={[
            { key: "select", title: "选择", render: (r) => <input type="checkbox" checked={selected.includes(r.id)} onChange={(e) => setSelected((prev) => e.target.checked ? [...prev, r.id] : prev.filter((id) => id !== r.id))} /> },
            { key: "projectName", title: "项目", render: (r) => r.projectName || "--" },
            { key: "campaignName", title: "活动" },
            { key: "code", title: "CDK", render: (r) => <code>{r.code}</code> },
            { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> },
            { key: "claimedAt", title: "领取时间", render: (r) => time(r.claimedAt) },
            { key: "actions", title: "操作", render: (r) => <div className="row-actions"><CopyButton value={r.code}>复制</CopyButton>{r.status === "frozen" ? <button className="btn btn--text" onClick={() => run("unfreeze", r)}>解冻</button> : <button className="btn btn--text" onClick={() => setConfirm({ action: "freeze", row: r })}>冻结</button>}<button className="btn btn--text" onClick={() => setConfirm({ action: "invalidate", row: r })}>作废</button><button className="btn btn--text danger" onClick={() => setConfirm({ action: "delete", row: r })}>删除</button></div> },
          ]} />
        </section>
      )}
      {mode === "export" && (
        <section className="panel">
          <div className="panel__header"><div><h2>CDK 导出任务</h2><p>所有 CSV 导出会写入后端 ExportTask，并可再次下载。</p></div></div>
          <DataTable rows={tasks} columns={[
            { key: "filename", title: "文件名" },
            { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> },
            { key: "createdAt", title: "创建时间", render: (r) => time(r.createdAt) },
            { key: "download", title: "下载", render: (r) => r.filePath ? <a className="btn btn--text" href={r.filePath} target="_blank">下载</a> : "--" },
          ]} />
        </section>
      )}
      <CdkImportModal
        open={importOpen}
        projects={projects}
        campaigns={campaigns}
        initialProjectId={projectId}
        initialCampaignId={campaignId}
        lockSelection={!!campaignId}
        onCancel={() => setImportOpen(false)}
        onCreateProject={() => navigate(`/admin/projects?open=create&returnTo=${encodeURIComponent("/admin/cdks/import")}`)}
        onSuccess={(res, campaign) => {
          const imported = res.imported || res.successCount || 0;
          const usedElsewhere = res.usedElsewhere || 0;
          const dupes = res.duplicates || 0;
          let msg = `成功导入 ${imported} 个 CDK 到活动【${campaign?.name || "未知活动"}】`;
          if (usedElsewhere > 0 || dupes > 0) {
            const parts = [];
            if (usedElsewhere > 0) parts.push(`${usedElsewhere} 个被其他活动占用`);
            if (dupes > 0) parts.push(`${dupes} 个本批重复`);
            msg += `，跳过 ${parts.join("、")}`;
          }
          showToast(msg, usedElsewhere > 0 ? "error" : "success");
          setImportOpen(false);
          notifyOnboardingRefresh();
          const returnTo = new URLSearchParams(location.search).get("returnTo");
          if (returnTo) {
            navigate(returnTo, { replace: true });
            return;
          }
          reload();
        }}
        onError={(message) => showToast(message, "error")}
      />
      <ConfirmDialog open={!!confirm} danger title="确认危险操作" description="该操作会修改 CDK 状态或删除未使用 CDK，确认继续？" onCancel={() => setConfirm(null)} onConfirm={() => run(confirm.action, confirm.row)} />
    </div>
  );
}

export function ClaimRecordsPage() {
  const location = useLocation();
  const [rows, setRows] = useState([]);
  const [campaigns, setCampaigns] = useState([]);
  const [nodes, setNodes] = useState([]);
  const [filters, setFilters] = useState({ keyword: "", status: "all", campaignId: "", nodeId: "", ip: "", fingerprint: "" });
  const [loading, setLoading] = useState(true);
  const [detail, setDetail] = useState(null);
  const [tasks, setTasks] = useState([]);
  const [toast, showToast] = useToast();
  const mode = location.pathname.endsWith("/failures") ? "failures" : location.pathname.endsWith("/devices") ? "devices" : location.pathname.endsWith("/exports") ? "exports" : "stream";
  useEffect(() => {
    setLoading(true);
    Promise.all([adminApi.listClaims({ ...filters, pageSize: 500 }), adminApi.listCampaigns({ pageSize: 200 }), adminApi.listNodes({ pageSize: 200 }), adminApi.exportTasks({ type: "claims", pageSize: 50 }).catch(() => ({ items: [] }))])
      .then(([claims, camps, nodeRes, exportTasks]) => { setRows(items(claims)); setCampaigns(items(camps)); setNodes(items(nodeRes)); setTasks(items(exportTasks)); })
      .catch((err) => showToast(err.message, "error"))
      .finally(() => setLoading(false));
  }, [filters]);
  useEffect(() => {
    if (location.pathname.endsWith("/failures")) setFilters((prev) => ({ ...prev, status: "failed" }));
    if (location.pathname.endsWith("/devices")) setFilters((prev) => ({ ...prev, keyword: "", status: "all" }));
  }, [location.pathname]);
  async function markRisk(row) { try { await adminApi.markClaimRisk(row.id); showToast("已标记异常"); } catch (err) { showToast(err.message, "error"); } }
  async function exportCSV() { try { const task = await adminApi.exportClaims(filters); window.open(task.filePath, "_blank"); } catch (err) { showToast(err.message, "error"); } }
  return (
    <div className="admin-page">
      {toast && <Toast {...toast} onClose={() => showToast(null)} />}
      <PageHeader eyebrow="Claim Records" title={mode === "failures" ? "失败记录" : mode === "devices" ? "设备与 IP" : mode === "exports" ? "导出任务" : "领取流水"} description="查看成功、失败、风控拦截流水，支持 IP、设备、活动、节点筛选和导出。" actions={<button className="btn btn--secondary" onClick={exportCSV}>导出 CSV</button>} />
      <div className="toolbar panel-toolbar">
        <FilterSelect value={filters.campaignId} onChange={(v) => setFilters({ ...filters, campaignId: v })}><option value="">全部活动</option>{campaigns.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</FilterSelect>
        <FilterSelect value={filters.nodeId} onChange={(v) => setFilters({ ...filters, nodeId: v })}><option value="">全部节点</option>{nodes.map((n) => <option key={n.id} value={n.id}>{n.name}</option>)}</FilterSelect>
        <FilterSelect value={filters.status} onChange={(v) => setFilters({ ...filters, status: v })}><option value="all">全部状态</option><option value="success">成功</option><option value="failed">失败</option><option value="blocked">拦截</option></FilterSelect>
        <SearchInput value={filters.keyword} onChange={(v) => setFilters({ ...filters, keyword: v })} placeholder="搜索 CDK / IP / 指纹" />
        <input className="search-input" value={filters.ip} onChange={(e) => setFilters({ ...filters, ip: e.target.value })} placeholder="IP" />
        <input className="search-input" value={filters.fingerprint} onChange={(e) => setFilters({ ...filters, fingerprint: e.target.value })} placeholder="设备指纹" />
      </div>
      <section className="panel">
        <DataTable loading={loading} rows={rows} pageSize={12} columns={[
          { key: "campaignName", title: "活动 / 节点", render: (r) => <><strong>{r.campaignName || "--"}</strong><small>{r.nodeName || "--"}</small></> },
          { key: "code", title: "CDK", render: (r) => r.code ? <CopyButton value={r.code}>{r.code}</CopyButton> : "--" },
          { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> },
          { key: "reason", title: "原因", render: (r) => r.reason || "--" },
          { key: "ip", title: "IP / 设备", render: (r) => <><span>{r.ip || "--"}</span><small>{r.fingerprint || "--"}</small></> },
          { key: "createdAt", title: "时间", render: (r) => time(r.createdAt) },
          { key: "actions", title: "操作", render: (r) => <div className="row-actions"><button className="btn btn--text" onClick={() => setDetail(r)}>详情</button><button className="btn btn--text" onClick={() => markRisk(r)}>标记异常</button></div> },
        ]} />
      </section>
      {mode === "devices" && (
        <section className="panel">
          <div className="panel__header"><div><h2>设备与 IP 聚合</h2><p>基于真实领取流水聚合同 IP、同设备行为。</p></div></div>
          <DataTable rows={Object.values(rows.reduce((acc, r) => { const key = `${r.ip || "--"}|${r.fingerprint || "--"}`; acc[key] ||= { id: key, ip: r.ip, fingerprint: r.fingerprint, count: 0, blocked: 0 }; acc[key].count++; if (r.status === "blocked") acc[key].blocked++; return acc; }, {}))} columns={[{ key: "ip", title: "IP" }, { key: "fingerprint", title: "设备指纹" }, { key: "count", title: "请求数" }, { key: "blocked", title: "拦截数" }]} />
        </section>
      )}
      {mode === "exports" && (
        <section className="panel">
          <div className="panel__header"><div><h2>领取记录导出任务</h2><p>导出任务来自后端 ExportTask。</p></div></div>
          <DataTable rows={tasks} columns={[{ key: "filename", title: "文件名" }, { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> }, { key: "createdAt", title: "创建时间", render: (r) => time(r.createdAt) }, { key: "download", title: "下载", render: (r) => r.filePath ? <a className="btn btn--text" href={r.filePath} target="_blank">下载</a> : "--" }]} />
        </section>
      )}
      <Modal open={!!detail} title="领取详情" onCancel={() => setDetail(null)}><pre className="modal-body">{JSON.stringify(detail, null, 2)}</pre></Modal>
    </div>
  );
}

export function AnalyticsPage() {
  const [range, setRange] = useState("7d");
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => { setLoading(true); adminApi.analyticsOverview({ range }).then(setData).finally(() => setLoading(false)); }, [range]);
  const overview = data?.overview || {};
  const claimsTrend = data?.claimsTrend || [];
  const campaignRanking = data?.campaignRanking || [];
  const nodeRanking = data?.nodeRanking || [];
  const failureReasons = data?.failureReasons || [];

  return (
    <div className="admin-page">
      <PageHeader eyebrow="Analytics" title="数据分析" description="基于真实节点访问、领取记录和失败原因聚合。" actions={<FilterSelect value={range} onChange={setRange}><option value="today">今日</option><option value="7d">最近 7 天</option><option value="30d">最近 30 天</option></FilterSelect>} />
      <section className="stats-grid"><StatCard label="总领取请求" value={fmt(overview.total)} /><StatCard label="成功" value={fmt(overview.success)} tone="success" /><StatCard label="失败 / 拦截" value={`${fmt(overview.failed)} / ${fmt(overview.blocked)}`} tone="warning" /><StatCard label="成功率" value={`${Number(overview.successRate || 0).toFixed(1)}%`} /></section>

      <section className="panel">
        <div className="panel__header"><div><h2>领取趋势</h2><p>按天聚合成功、失败、拦截数量。</p></div></div>
        {loading ? <EmptyState title="加载中" /> : claimsTrend.length === 0 ? <EmptyState title="暂无趋势数据" description="当有领取记录后将自动生成趋势图。" /> : (
          <AnalyticsChart data={claimsTrend} />
        )}
      </section>

      <section className="dashboard-main-grid">
        <div className="panel">
          <div className="panel__header"><h2>活动排行</h2></div>
          {campaignRanking.length === 0 ? <EmptyState title="暂无活动数据" /> : (
            <DataTable rows={campaignRanking} columns={[{ key: "name", title: "活动" }, { key: "claimedCount", title: "已领取" }, { key: "remaining", title: "剩余" }, { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> }]} />
          )}
        </div>
        <div className="panel">
          <div className="panel__header"><h2>失败原因</h2></div>
          {failureReasons.length === 0 ? <EmptyState title="暂无失败记录" description="所有领取均成功。" /> : (
            <div className="failure-reasons-list">
              {failureReasons.map((r, i) => (
                <div className="failure-reason-item" key={i}>
                  <div className="failure-reason-item__bar">
                    <div className="failure-reason-item__fill" style={{ width: `${Math.min(100, (r.count / Math.max(1, failureReasons[0]?.count)) * 100)}%` }} />
                    <span className="failure-reason-item__label">{r.reason}</span>
                  </div>
                  <strong className="failure-reason-item__count">{r.count}</strong>
                </div>
              ))}
            </div>
          )}
        </div>
      </section>

      <section className="panel">
        <div className="panel__header"><h2>节点转化排行</h2></div>
        {nodeRanking.length === 0 ? <EmptyState title="暂无节点数据" /> : (
          <DataTable rows={nodeRanking} columns={[{ key: "name", title: "节点" }, { key: "visits", title: "访问" }, { key: "claims", title: "领取" }, { key: "conversion", title: "转化率", render: (r) => `${r.conversion || 0}%` }]} />
        )}
      </section>
    </div>
  );
}

function AnalyticsChart({ data }) {
  return (
    <div style={{ width: "100%", height: 300 }}>
      <ResponsiveContainer>
        <AreaChart data={data} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
          <XAxis dataKey="date" tick={{ fontSize: 12, fill: "#737373" }} tickFormatter={(v) => v?.slice(5) || v} />
          <YAxis tick={{ fontSize: 12, fill: "#737373" }} />
          <Tooltip contentStyle={{ borderRadius: 10, border: "1px solid #e2e8f0", fontSize: 13 }} />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <Area type="monotone" dataKey="success" name="成功" stroke="#10b981" fill="rgba(16,185,129,0.15)" strokeWidth={2} />
          <Area type="monotone" dataKey="failed" name="失败" stroke="#ef4444" fill="rgba(239,68,68,0.1)" strokeWidth={2} />
          <Area type="monotone" dataKey="blocked" name="拦截" stroke="#f59e0b" fill="rgba(245,158,11,0.1)" strokeWidth={2} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

export function LogsPage() {
  const location = useLocation();
  const [rows, setRows] = useState([]);
  const [filters, setFilters] = useState({ keyword: "", type: "all", level: "all" });
  const [loading, setLoading] = useState(true);
  const [detail, setDetail] = useState(null);
  const [confirm, setConfirm] = useState(null);
  const [toast, showToast] = useToast();
  const mode = location.pathname.endsWith("/operations") ? "操作日志" : location.pathname.endsWith("/claims") ? "领取日志" : location.pathname.endsWith("/errors") ? "错误日志" : "日志总览";
  async function reload() { setLoading(true); try { setRows(items(await adminApi.logs({ ...filters, pageSize: 500 }))); } catch (err) { showToast(err.message, "error"); } finally { setLoading(false); } }
  useEffect(() => { reload(); }, [filters]);
  useEffect(() => {
    if (location.pathname.endsWith("/operations")) setFilters((prev) => ({ ...prev, type: "operation" }));
    if (location.pathname.endsWith("/claims")) setFilters((prev) => ({ ...prev, type: "claim" }));
    if (location.pathname.endsWith("/errors")) setFilters((prev) => ({ ...prev, type: "error" }));
  }, [location.pathname]);
  async function cleanup() { try { await adminApi.cleanupLogs(30); showToast("清理完成"); reload(); } catch (err) { showToast(err.message, "error"); } finally { setConfirm(null); } }
  async function exportCSV() { const task = await adminApi.exportLogs(filters); window.open(task.filePath, "_blank"); }
  return <div className="admin-page">{toast && <Toast {...toast} onClose={() => showToast(null)} />}<PageHeader eyebrow="System Logs" title={mode} description="查看操作、领取、错误和安全日志。" actions={<div className="btn-group"><button className="btn btn--secondary" onClick={exportCSV}>导出</button><button className="btn btn--secondary" onClick={() => setConfirm(true)}>清理旧日志</button></div>} /><div className="toolbar panel-toolbar"><FilterSelect value={filters.type} onChange={(v) => setFilters({ ...filters, type: v })}><option value="all">全部类型</option><option value="operation">操作</option><option value="claim">领取</option><option value="error">错误</option><option value="security">安全</option></FilterSelect><FilterSelect value={filters.level} onChange={(v) => setFilters({ ...filters, level: v })}><option value="all">全部等级</option><option value="info">info</option><option value="warn">warn</option><option value="error">error</option></FilterSelect><SearchInput value={filters.keyword} onChange={(v) => setFilters({ ...filters, keyword: v })} placeholder="关键词" /></div><section className="panel"><DataTable loading={loading} rows={rows} pageSize={14} columns={[{ key: "type", title: "类型", render: (r) => <StatusBadge status={r.type} /> }, { key: "level", title: "等级", render: (r) => <StatusBadge status={r.level} /> }, { key: "title", title: "标题" }, { key: "message", title: "消息" }, { key: "actor", title: "操作者" }, { key: "createdAt", title: "时间", render: (r) => time(r.createdAt) }, { key: "actions", title: "操作", render: (r) => <button className="btn btn--text" onClick={() => setDetail(r)}>详情</button> }]} /></section><Modal open={!!detail} title="日志详情" onCancel={() => setDetail(null)}><pre className="modal-body">{JSON.stringify(detail, null, 2)}</pre></Modal><ConfirmDialog open={!!confirm} danger title="清理旧日志" description="将清理 30 天前日志，确认继续？" onCancel={() => setConfirm(null)} onConfirm={cleanup} /></div>;
}

export function RiskCenterPage() {
  const location = useLocation();
  const [overview, setOverview] = useState({});
  const [rules, setRules] = useState([]);
  const [blacklist, setBlacklist] = useState([]);
  const [hits, setHits] = useState([]);
  const [ruleForm, setRuleForm] = useState(null);
  const [blackForm, setBlackForm] = useState(null);
  const [toast, showToast] = useToast();
  const mode = location.pathname.endsWith("/rules") ? "限制规则" : location.pathname.endsWith("/blacklist") ? "黑名单" : location.pathname.endsWith("/frequency") ? "频控策略" : "规则总览";
  async function reload() { const [o, r, b, h] = await Promise.all([adminApi.riskOverview(), adminApi.riskRules(), adminApi.blacklist(), adminApi.riskHits()]); setOverview(o); setRules(r); setBlacklist(b); setHits(h); }
  useEffect(() => { reload().catch((err) => showToast(err.message, "error")); }, []);
  useEffect(() => {
    if (location.pathname.endsWith("/frequency")) setRuleForm({ name: "单 IP 每分钟限制", type: "rate_limit", enabled: true, action: "block", configText: "{\"limit\":10,\"windowSeconds\":60}" });
  }, [location.pathname]);
  async function saveRule(e) { e.preventDefault(); try { const config = JSON.parse(ruleForm.configText || "{}"); await adminApi.saveRiskRule({ ...ruleForm, config }, ruleForm.id); setRuleForm(null); showToast("规则已保存"); reload(); } catch (err) { showToast(err.message, "error"); } }
  async function saveBlack(e) { e.preventDefault(); try { await adminApi.saveBlacklist(blackForm, blackForm.id); setBlackForm(null); showToast("黑名单已保存"); reload(); } catch (err) { showToast(err.message, "error"); } }
  return <div className="admin-page">{toast && <Toast {...toast} onClose={() => showToast(null)} />}<PageHeader eyebrow="Risk Control" title={mode} description="规则和黑名单会在公开领取 POST 流程中真实生效。" actions={<div className="btn-group"><button className="btn btn--primary" onClick={() => setRuleForm({ name: "", type: "rate_limit", enabled: true, action: "block", configText: "{\"limit\":10,\"windowSeconds\":60}" })}>创建规则</button><button className="btn btn--secondary" onClick={() => setBlackForm({ type: "ip", value: "", reason: "", enabled: true })}>添加黑名单</button></div>} /><section className="stats-grid"><StatCard label="规则" value={fmt(overview.rules)} /><StatCard label="启用规则" value={fmt(overview.enabledRules)} tone="success" /><StatCard label="黑名单" value={fmt(overview.blacklist)} /><StatCard label="命中记录" value={fmt(overview.hits)} tone="warning" /></section><section className="dashboard-main-grid"><div className="panel"><div className="panel__header"><h2>限制规则</h2></div><DataTable rows={rules} columns={[{ key: "name", title: "名称" }, { key: "type", title: "类型" }, { key: "enabled", title: "状态", render: (r) => <StatusBadge status={r.enabled ? "active" : "paused"} /> }, { key: "action", title: "动作" }, { key: "actions", title: "操作", render: (r) => <div className="row-actions"><button className="btn btn--text" onClick={() => setRuleForm({ ...r, configText: JSON.stringify(r.config || {}, null, 2) })}>编辑</button><button className="btn btn--text" onClick={() => r.enabled ? adminApi.disableRiskRule(r.id).then(reload) : adminApi.enableRiskRule(r.id).then(reload)}>{r.enabled ? "禁用" : "启用"}</button><button className="btn btn--text danger" onClick={() => adminApi.deleteRiskRule(r.id).then(reload)}>删除</button></div> }]} /></div><div className="panel"><div className="panel__header"><h2>黑名单</h2></div><DataTable rows={blacklist} columns={[{ key: "type", title: "类型" }, { key: "value", title: "值" }, { key: "reason", title: "原因" }, { key: "enabled", title: "状态", render: (r) => <StatusBadge status={r.enabled ? "active" : "paused"} /> }, { key: "actions", title: "操作", render: (r) => <div className="row-actions"><button className="btn btn--text" onClick={() => setBlackForm(r)}>编辑</button><button className="btn btn--text" onClick={() => r.enabled ? adminApi.disableBlacklist(r.id).then(reload) : adminApi.enableBlacklist(r.id).then(reload)}>{r.enabled ? "禁用" : "启用"}</button><button className="btn btn--text danger" onClick={() => adminApi.deleteBlacklist(r.id).then(reload)}>删除</button></div> }]} /></div></section><section className="panel"><div className="panel__header"><h2>风控命中记录</h2></div><DataTable rows={hits} columns={[{ key: "campaignName", title: "活动" }, { key: "ip", title: "IP" }, { key: "fingerprint", title: "设备" }, { key: "reason", title: "原因" }, { key: "createdAt", title: "时间", render: (r) => time(r.createdAt) }]} /></section><Modal open={!!ruleForm} title="风控规则" onCancel={() => setRuleForm(null)}><form className="modal-form" onSubmit={saveRule}><input className="styled-input" placeholder="规则名称" value={ruleForm?.name || ""} onChange={(e) => setRuleForm({ ...ruleForm, name: e.target.value })} /><FilterSelect value={ruleForm?.type || "rate_limit"} onChange={(v) => setRuleForm({ ...ruleForm, type: v })}><option value="rate_limit">单 IP 每分钟限制</option><option value="device_limit">单设备同活动限制</option><option value="user_agent">User-Agent 规则</option><option value="ip_limit">IP 限制</option></FilterSelect><textarea className="styled-textarea mono" rows="5" value={ruleForm?.configText || ""} onChange={(e) => setRuleForm({ ...ruleForm, configText: e.target.value })} /><label className="switch-label"><input type="checkbox" checked={!!ruleForm?.enabled} onChange={(e) => setRuleForm({ ...ruleForm, enabled: e.target.checked })} />启用</label><div className="modal-footer"><button className="modal-btn modal-btn--primary">保存</button></div></form></Modal><Modal open={!!blackForm} title="黑名单" onCancel={() => setBlackForm(null)}><form className="modal-form" onSubmit={saveBlack}><FilterSelect value={blackForm?.type || "ip"} onChange={(v) => setBlackForm({ ...blackForm, type: v })}><option value="ip">IP</option><option value="fingerprint">设备指纹</option><option value="user_agent">User-Agent</option></FilterSelect><input className="styled-input" value={blackForm?.value || ""} onChange={(e) => setBlackForm({ ...blackForm, value: e.target.value })} placeholder="匹配值" /><input className="styled-input" value={blackForm?.reason || ""} onChange={(e) => setBlackForm({ ...blackForm, reason: e.target.value })} placeholder="原因" /><label className="switch-label"><input type="checkbox" checked={!!blackForm?.enabled} onChange={(e) => setBlackForm({ ...blackForm, enabled: e.target.checked })} />启用</label><div className="modal-footer"><button className="modal-btn modal-btn--primary">保存</button></div></form></Modal></div>;
}

export function CaptchaConfigPage() {
  const location = useLocation();
  const [config, setConfig] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [selected, setSelected] = useState([]);
  const [toast, showToast] = useToast();
  const mode = location.pathname.endsWith("/hcaptcha") ? "hCaptcha" : location.pathname.endsWith("/nodes") ? "节点验证码" : location.pathname.endsWith("/testing") ? "连接测试" : "配置总览";
  async function reload() { const [c, n] = await Promise.all([adminApi.captchaConfig(), adminApi.captchaNodes({ pageSize: 200 })]); setConfig(c); setNodes(items(n)); }
  useEffect(() => { reload().catch((err) => showToast(err.message, "error")); }, []);
  async function test() { try { setConfig(await adminApi.testCaptcha()); showToast("测试完成"); } catch (err) { showToast(err.message, "error"); } }
  async function batch(enabled) { try { await (enabled ? adminApi.batchEnableCaptcha(selected) : adminApi.batchDisableCaptcha(selected)); showToast("操作成功"); setSelected([]); reload(); } catch (err) { showToast(err.message, "error"); } }
  async function save() { try { setConfig(await adminApi.updateCaptchaConfig(config)); showToast("配置已保存"); } catch (err) { showToast(err.message, "error"); } }
  return <div className="admin-page">{toast && <Toast {...toast} onClose={() => showToast(null)} />}<PageHeader eyebrow="Captcha" title={mode} description="hCaptcha Secret 只在后端环境变量读取，前端只显示是否配置。" actions={<div className="btn-group"><button className="btn btn--primary" onClick={save}>保存配置</button><button className="btn btn--secondary" onClick={test}>连接测试</button></div>} /><section className="stats-grid"><StatCard label="Provider" value={config?.provider || "--"} /><StatCard label="全局开关" value={config?.enabled ? "开启" : "关闭"} tone={config?.enabled ? "success" : "warning"} /><StatCard label="Site Key" value={config?.hCaptchaSiteKeyConfigured ? "已配置" : "未配置"} /><StatCard label="Secret" value={config?.hCaptchaSecretConfigured ? "已配置" : "未配置"} /></section><section className="panel modal-form"><div className="panel__header"><div><h2>{mode === "testing" ? "连接测试" : "hCaptcha 配置"}</h2><p>Secret 不会展示，也不会写入 state.json。</p></div></div><label className="switch-label"><input type="checkbox" checked={!!config?.enabled} onChange={(e) => setConfig({ ...config, enabled: e.target.checked })} />开启全局验证码</label><FilterSelect value={config?.provider || "hcaptcha"} onChange={(v) => setConfig({ ...config, provider: v })}><option value="none">none</option><option value="hcaptcha">hCaptcha</option><option value="builtin">builtin</option></FilterSelect><div className="empty-panel">最近测试：{config?.lastTestStatus || "--"} · {config?.lastTestMessage || "--"} · {time(config?.lastTestAt)}</div></section><section className="panel"><div className="panel__header"><div><h2>节点验证码</h2><p>可批量开启或关闭节点验证码。</p></div><div className="btn-group"><button className="btn btn--secondary" disabled={!selected.length} onClick={() => batch(true)}>批量开启</button><button className="btn btn--secondary" disabled={!selected.length} onClick={() => batch(false)}>批量关闭</button></div></div><DataTable rows={nodes} columns={[{ key: "select", title: "选择", render: (r) => <input type="checkbox" checked={selected.includes(r.id)} onChange={(e) => setSelected((prev) => e.target.checked ? [...prev, r.id] : prev.filter((id) => id !== r.id))} /> }, { key: "name", title: "节点" }, { key: "campaignName", title: "活动" }, { key: "requireCaptcha", title: "验证码", render: (r) => <StatusBadge status={r.requireCaptcha ? "active" : "paused"} /> }]} /></section></div>;
}

export function SettingsPageFallback() {
  return null;
}
