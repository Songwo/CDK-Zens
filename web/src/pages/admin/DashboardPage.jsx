import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "../../lib/api";
import {
  AlertPanel,
  CopyButton,
  DataTable,
  FilterSelect,
  MiniStatCard,
  PageHeader,
  ProgressBar,
  QuickActions,
  SearchInput,
  StatCard,
  StatusBadge,
  SystemHealthCard,
} from "../../components/ui";
import { DistributionGuideCard } from "../../components/DistributionGuide";

function fmt(n) {
  return new Intl.NumberFormat("zh-CN").format(Number(n || 0));
}

function formatTime(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function campaignStatusLabel(status) {
  if (status === "disabled") return "paused";
  return status;
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const [data, setData] = useState(null);
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [sort, setSort] = useState("recent");

  async function reload() {
    setLoading(true);
    try {
      const [dashboardData, healthData] = await Promise.all([
        adminApi.dashboard(),
        adminApi.health().catch(() => null),
      ]);
      setData(dashboardData);
      setHealth(healthData);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { reload(); }, []);

  const campaigns = useMemo(() => {
    const q = search.trim().toLowerCase();
    const rows = (data?.campaigns || []).filter((item) => {
      if (status !== "all" && item.status !== status) return false;
      if (!q) return true;
      return [item.name, item.id, item.projectCode, item.claimUrl].some((v) => String(v || "").toLowerCase().includes(q));
    });
    return [...rows].sort((a, b) => {
      if (sort === "stock") return Number(a.remaining || 0) - Number(b.remaining || 0);
      if (sort === "claimed") return Number(b.claimedCount || 0) - Number(a.claimedCount || 0);
      return new Date(b.createdAt || b.startTime || 0) - new Date(a.createdAt || a.startTime || 0);
    });
  }, [data, search, status, sort]);

  const stats = data?.stats || {};
  const nodes = data?.nodes || [];
  const captchaRequired = nodes.some((node) => node.requireCaptcha);
  const captchaConfigured = Boolean(import.meta.env.VITE_HCAPTCHA_SITE_KEY);

  const coreStats = [
    ["活跃活动", stats.activeCampaigns, `共 ${fmt(stats.totalCampaigns)} 个活动`, "success"],
    ["总 CDK 库存", stats.totalStock, "所有活动累计库存", "default"],
    ["已领取", stats.claimedCount, "累计成功发放", "gold"],
    ["剩余库存", stats.remainingCount, "可继续分发", Number(stats.remainingCount) === 0 ? "danger" : "default"],
  ];
  const miniStats = [
    ["总节点", stats.totalNodes, "分发入口数量", "default"],
    ["异常节点", stats.abnormalNodes, "暂停或异常节点", Number(stats.abnormalNodes) ? "danger" : "success"],
    ["今日领取", stats.todayClaims, "今日成功领取", "gold"],
    ["领取成功率", `${(stats.successRate || 0).toFixed(1)}%`, "已领取 / 总库存", "success"],
  ];

  const quickActions = [
    { label: "新建活动", icon: "▦", onClick: () => navigate("/admin/campaigns?open=create") },
    { label: "导入 CDK", icon: "⇪", onClick: () => navigate("/admin/cdks/import") },
    { label: "创建节点", icon: "⎈", onClick: () => navigate("/admin/nodes/create") },
    { label: "领取记录", icon: "◷", onClick: () => navigate("/admin/claims") },
    { label: "验证码", icon: "▣", onClick: () => navigate("/admin/captcha") },
    { label: "系统日志", icon: "☰", onClick: () => navigate("/admin/logs") },
  ];

  return (
    <div className="admin-page dashboard-page">
      <PageHeader
        eyebrow="Operations"
        title="运营总览"
        description="集中观察活动、分发节点、库存、领取记录和异常提醒。"
        actions={<div className="btn-group"><button className="btn btn--primary" onClick={() => navigate("/admin/campaigns?open=create")}>新建活动</button><button className="btn btn--secondary" onClick={() => navigate("/admin/nodes/create")}>创建节点</button></div>}
      />

      <section className="stats-grid stats-grid--core">
        {coreStats.map(([label, value, hint, tone]) => <StatCard key={label} label={label} value={fmt(value)} hint={hint} tone={tone} />)}
      </section>

      <DistributionGuideCard stats={stats} nodes={nodes} captchaConfigured={captchaConfigured} />

      <section className="stats-grid stats-grid--mini">
        {miniStats.map(([label, value, hint, tone]) => <MiniStatCard key={label} label={label} value={typeof value === "number" ? fmt(value) : value} hint={hint} tone={tone} />)}
      </section>

      <section className="dashboard-main-grid">
        <div className="dashboard-primary">
          <section className="panel">
            <div className="panel__header">
              <div><h2>分发活动总览</h2><p>默认展示最近 8 条，支持状态筛选、搜索和排序。</p></div>
              <button className="btn btn--secondary" onClick={() => navigate("/admin/campaigns")}>查看全部活动</button>
            </div>
            <div className="table-toolbar">
              <FilterSelect value={status} onChange={setStatus}>
                <option value="all">全部状态</option>
                <option value="active">进行中</option>
                <option value="disabled">已暂停</option>
                <option value="upcoming">未开始</option>
                <option value="ended">已结束</option>
                <option value="soldout">库存耗尽</option>
              </FilterSelect>
              <SearchInput value={search} onChange={setSearch} placeholder="搜索活动 / 活动 ID / 短链接" />
              <FilterSelect value={sort} onChange={setSort}>
                <option value="recent">最近创建</option>
                <option value="claimed">领取最多</option>
                <option value="stock">库存最少</option>
              </FilterSelect>
              <button className="btn btn--secondary" onClick={reload}>刷新</button>
            </div>
            <DataTable
              loading={loading}
              rows={campaigns}
              pageSize={8}
              emptyText="暂无匹配活动"
              columns={[
                { key: "name", title: "活动名称", render: (r) => <><strong>{r.name}</strong><small>{r.id || r.projectCode}</small></> },
                { key: "status", title: "状态", render: (r) => <StatusBadge status={campaignStatusLabel(r.status)} /> },
                { key: "progress", title: "领取进度", render: (r) => <ProgressBar claimed={r.claimedCount} total={r.totalStock} remaining={r.remaining} /> },
                { key: "time", title: "开始 / 结束", render: (r) => <span>{formatTime(r.startTime)}<br /><small>{formatTime(r.endTime)}</small></span> },
                { key: "nodes", title: "绑定节点", render: (r) => r.nodeCount || 0 },
                { key: "actions", title: "操作", render: (r) => <div className="row-actions"><button className="btn btn--text" onClick={() => navigate("/admin/campaigns")}>详情</button><button className="btn btn--text" onClick={() => navigate("/admin/cdks")}>CDK</button>{r.claimUrl && <CopyButton value={window.location.origin + r.claimUrl}>复制链接</CopyButton>}</div> },
              ]}
            />
          </section>

          <section className="panel">
            <div className="panel__header">
              <div><h2>节点总览</h2><p>观察分发入口的访问、领取、转化和库存。</p></div>
              <button className="btn btn--secondary" onClick={() => navigate("/admin/nodes")}>管理节点</button>
            </div>
            <DataTable
              loading={loading}
              rows={nodes}
              pageSize={6}
              emptyText="暂无分发节点"
              columns={[
                { key: "name", title: "节点", render: (r) => <><strong>{r.name}</strong><small>{r.slug}</small></> },
                { key: "campaignName", title: "绑定活动", render: (r) => <span>{r.campaignName || "--"}</span> },
                { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status === "active" ? (r.campaignStatus || "active") : r.status} /> },
                { key: "traffic", title: "访问 / 领取 / 转化", render: (r) => <span>{fmt(r.visits)} / {fmt(r.claims)} / {r.conversion || 0}%</span> },
                { key: "remaining", title: "剩余库存", render: (r) => fmt(r.remaining) },
                { key: "copy", title: "链接", render: (r) => <CopyButton value={window.location.origin + r.claimUrl}>复制</CopyButton> },
              ]}
            />
          </section>
        </div>

        <aside className="dashboard-side-stack">
          <section className="panel panel--tight">
            <div className="panel__header"><div><h2>异常提醒</h2><p>最多显示 4 条高优先级提醒。</p></div></div>
            <AlertPanel alerts={data?.alerts || []} onViewAll={() => navigate("/admin/logs")} />
          </section>
          <section className="panel panel--tight">
            <div className="panel__header"><div><h2>快捷操作</h2><p>常用运营入口。</p></div></div>
            <QuickActions actions={quickActions} />
          </section>
          <section className="panel panel--tight">
            <div className="panel__header"><div><h2>系统健康</h2><p>关键服务状态概览。</p></div></div>
            <SystemHealthCard health={health || {}} captchaConfigured={captchaConfigured} captchaRequired={captchaRequired} />
          </section>
          <section className="panel panel--tight">
            <div className="panel__header"><div><h2>最近领取</h2><p>最近 4 条成功或失败记录。</p></div></div>
            <div className="recent-claim-list">
              {(data?.recentRecords || []).slice(0, 4).map((record) => (
                <div className="recent-claim" key={record.id || record.claimToken || record.createdAt}>
                  <code>{record.rewardContent || record.code || "--"}</code>
                  <span>{record.ip || record.fingerprint || "匿名"} · {formatTime(record.createdAt)}</span>
                </div>
              ))}
              {!(data?.recentRecords || []).length && <div className="empty-panel empty-panel--compact"><strong>暂无领取记录</strong></div>}
            </div>
          </section>
        </aside>
      </section>
    </div>
  );
}
