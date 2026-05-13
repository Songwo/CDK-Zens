import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { api } from "../lib/api";

const CODE_TYPE_LABELS = {
  single_use: "一次性激活码",
  generic: "通用兑换码",
  batch_unique: "批量独立码",
};

const REWARD_TYPE_LABELS = {
  text: "文本内容",
  benefit_package: "权益包",
  points: "积分",
  membership_days: "会员天数",
  api_key: "API Key",
  redeem_link: "兑换链接",
  mixed: "组合奖励",
};

function toLocalInputValue(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const pad = (part) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function toISOValue(value) {
  return value ? new Date(value).toISOString() : "";
}

function createProjectForm(activity = {}) {
  return {
    projectName: activity.title || "",
    tagsText: (activity.tags || []).join(", "),
    startTime: toLocalInputValue(activity.startTime),
    endTime: toLocalInputValue(activity.endTime),
    requiredLevel: activity.limit?.requiredLevel ?? 0,
    requiredScore: activity.limit?.requiredScore ?? 0,
    ipLimit: activity.limit?.limitPerIP ?? true,
    costPoints: activity.limit?.costPoints ?? 0,
    description: activity.description || "",
    codeType: activity.codeType || "batch_unique",
    codeCount: 100,
    userLimit: activity.limit?.userLimit ?? 1,
    totalStock: activity.maxCount ?? 100,
    dailyLimit: activity.limit?.dailyLimit ?? 0,
    rewardType: activity.rewardType || "text",
    rewardContent: activity.rewardContent || "CDK 激活码",
    successMessage: activity.successMessage || "领取成功，请尽快复制并完成兑换。",
    failureMessage: activity.failureMessage || "当前不满足领取条件，请检查活动规则。",
  };
}

function splitTags(value) {
  return value
    .split(/[,，\s]+/)
    .map((tag) => tag.trim())
    .filter(Boolean)
    .slice(0, 10);
}

function formatTime(value) {
  if (!value) return "--";
  return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function formatDateTimeLocal(date) {
  const pad = (part) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function buildClaimUrl(link) {
  return `${window.location.origin}${link.link || `/?claim=${link.token}`}`;
}

function MetricCard({ label, value, desc, trend, tone = "good" }) {
  return (
    <article className="dash-metric-card">
      <div className="dash-metric-card__top">
        <span>{label}</span>
        <b className={`dash-trend dash-trend--${tone}`}>{trend}</b>
      </div>
      <strong>{value}</strong>
      <p>{desc}</p>
    </article>
  );
}

function LineChart({ mode }) {
  const points = mode === "claims" ? [18, 34, 28, 48, 56, 72, 84] : [12, 18, 24, 31, 40, 46, 58];
  const max = Math.max(...points);
  const coords = points.map((value, index) => {
    const x = 30 + index * 88;
    const y = 184 - (value / max) * 136;
    return [x, y];
  });
  const path = coords.map(([x, y], index) => `${index === 0 ? "M" : "L"} ${x} ${y}`).join(" ");
  const area = `${path} L ${coords.at(-1)[0]} 204 L ${coords[0][0]} 204 Z`;

  return (
    <svg className="dash-line-chart" viewBox="0 0 620 220" role="img" aria-label="趋势图">
      <defs>
        <linearGradient id="lineFill" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor="rgba(17,24,39,.18)" />
          <stop offset="100%" stopColor="rgba(17,24,39,0)" />
        </linearGradient>
      </defs>
      {[48, 88, 128, 168, 204].map((y) => (
        <line key={y} x1="28" x2="590" y1={y} y2={y} stroke="#eef2f7" />
      ))}
      <path d={area} fill="url(#lineFill)" />
      <path d={path} fill="none" stroke="#111827" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" />
      {coords.map(([x, y]) => (
        <circle key={`${x}-${y}`} cx={x} cy={y} r="5" fill="#111827" stroke="#fff" strokeWidth="3" />
      ))}
      {["一", "二", "三", "四", "五", "六", "日"].map((day, index) => (
        <text key={day} x={30 + index * 88} y="216" textAnchor="middle" fill="#94a3b8" fontSize="12">
          周{day}
        </text>
      ))}
    </svg>
  );
}

function DonutChart({ stats }) {
  const claimed = Number(stats?.claimedTotal ?? 0);
  const remaining = Number(stats?.remainingTotal ?? 0);
  const total = Math.max(1, claimed + remaining);
  const claimedPercent = Math.round((claimed / total) * 100);

  return (
    <div className="dash-donut-wrap">
      <div className="dash-donut" style={{ "--claimed": `${claimedPercent}%` }}>
        <div>
          <strong>{claimedPercent}%</strong>
          <span>已领取</span>
        </div>
      </div>
      <div className="dash-dist-list">
        <p><span className="dot dot--dark" />进行中 <strong>{stats ? 1 : 0}</strong></p>
        <p><span className="dot dot--gold" />待领取 <strong>{remaining}</strong></p>
        <p><span className="dot dot--muted" />已领取 <strong>{claimed}</strong></p>
      </div>
    </div>
  );
}

function RankingCard({ title, items }) {
  return (
    <article className="dash-panel dash-rank-card">
      <div className="dash-panel__head dash-panel__head--compact">
        <div>
          <h3>{title}</h3>
          <p>{items.length} 条记录</p>
        </div>
      </div>
      <div className="dash-rank-list">
        {items.map((item, index) => (
          <div className="dash-rank-item" key={`${title}-${item.name}`}>
            <span>{String(index + 1).padStart(2, "0")}</span>
            <div>
              <strong>{item.name}</strong>
              <p>{item.desc}</p>
            </div>
            <b>{item.value}</b>
          </div>
        ))}
      </div>
    </article>
  );
}

function ProjectModal({ visible, state, loading, onClose, onSubmit, pushToast }) {
  const [tab, setTab] = useState("basic");
  const [step, setStep] = useState(0);
  const [form, setForm] = useState(() => createProjectForm());
  const [errors, setErrors] = useState({});

  useEffect(() => {
    if (visible) {
      setTab("basic");
      setStep(0);
      setErrors({});
      setForm(createProjectForm(state?.activity));
    }
  }, [visible, state?.activity]);

  if (!visible) return null;

  const highRisk = !form.ipLimit || Number(form.userLimit) === 0 || form.codeType === "generic";

  function patch(key, value) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function applyQuickTime(type) {
    const start = new Date();
    const end = new Date();
    if (type === "today") {
      end.setHours(23, 59, 0, 0);
    } else if (type === "7d") {
      end.setDate(end.getDate() + 7);
    } else if (type === "30d") {
      end.setDate(end.getDate() + 30);
    } else {
      end.setFullYear(end.getFullYear() + 20);
    }
    patch("startTime", formatDateTimeLocal(start));
    patch("endTime", formatDateTimeLocal(end));
  }

  function validate(targetStep = step) {
    const nextErrors = {};
    if (targetStep <= 0) {
      if (!form.projectName.trim()) nextErrors.projectName = "项目名称必填";
      if (form.projectName.trim().length > 32) nextErrors.projectName = "项目名称最多 32 字";
      if (!form.startTime) nextErrors.startTime = "请选择开始时间";
      if (!form.endTime) nextErrors.endTime = "请选择结束时间";
      if (form.startTime && form.endTime && new Date(form.endTime) <= new Date(form.startTime)) {
        nextErrors.endTime = "结束时间必须晚于开始时间";
      }
    }
    if (targetStep <= 1) {
      if (Number(form.totalStock) < 1) nextErrors.totalStock = "总库存必须大于 0";
      if (Number(form.codeCount) < 1) nextErrors.codeCount = "CDK 数量必须大于 0";
    }
    setErrors(nextErrors);
    return Object.keys(nextErrors).length === 0;
  }

  function next() {
    if (!validate(step)) return;
    if (step === 1 && highRisk && !window.confirm("当前存在高风险配置：不限 IP、不限制领取次数或通用码。确认继续预览吗？")) {
      return;
    }
    setStep((current) => Math.min(2, current + 1));
    setTab(step + 1 === 1 ? "distribution" : "preview");
  }

  async function create() {
    if (!validate(1)) return;
    if (highRisk && !window.confirm("请再次确认：当前配置可能被刷领，是否仍然创建？")) return;
    const firstLink = await onSubmit(form);
    if (firstLink) {
      await navigator.clipboard.writeText(firstLink);
      pushToast({ tone: "success", title: "项目创建成功", message: "首个领取链接已复制到剪贴板。" });
    }
  }

  return createPortal(
    <div className="dash-modal-layer" role="presentation">
      <div className="dash-modal" role="dialog" aria-modal="true" aria-label="新建项目">
        <div className="dash-modal__head">
          <div>
            <p>New Project</p>
            <h2>新建 CDK 项目</h2>
          </div>
          <button type="button" className="dash-icon-btn" onClick={onClose}>×</button>
        </div>

        <div className="dash-steps">
          {["基本设置", "分发内容", "规则预览"].map((label, index) => (
            <button
              key={label}
              type="button"
              className={step === index ? "is-active" : ""}
              onClick={() => {
                if (index <= step || validate(step)) {
                  setStep(index);
                  setTab(index === 0 ? "basic" : index === 1 ? "distribution" : "preview");
                }
              }}
            >
              <span>{index + 1}</span>{label}
            </button>
          ))}
        </div>

        <div className="dash-modal__body">
          {tab === "basic" ? (
            <div className="dash-form-grid">
              <label className="dash-field dash-field--full">
                <span>项目名称 <b>*</b></span>
                <input value={form.projectName} maxLength={32} onChange={(e) => patch("projectName", e.target.value)} placeholder="例如：春节限定礼包" />
                {errors.projectName ? <em>{errors.projectName}</em> : <small>最多 32 字。</small>}
              </label>

              <label className="dash-field dash-field--full">
                <span>项目标签</span>
                <input value={form.tagsText} onChange={(e) => patch("tagsText", e.target.value)} placeholder="最多 10 个，例如：限时活动, 会员专属" />
                <small>用空格或逗号分隔，最多保存 10 个。</small>
              </label>

              <label className="dash-field">
                <span>开始时间 <b>*</b></span>
                <input type="datetime-local" value={form.startTime} onChange={(e) => patch("startTime", e.target.value)} />
                {errors.startTime ? <em>{errors.startTime}</em> : null}
              </label>

              <label className="dash-field">
                <span>结束时间 <b>*</b></span>
                <input type="datetime-local" value={form.endTime} onChange={(e) => patch("endTime", e.target.value)} />
                {errors.endTime ? <em>{errors.endTime}</em> : null}
              </label>

              <div className="dash-quick-row dash-field--full">
                {[
                  ["today", "今天"],
                  ["7d", "7 天"],
                  ["30d", "30 天"],
                  ["forever", "永久有效"],
                ].map(([key, label]) => (
                  <button type="button" key={key} onClick={() => applyQuickTime(key)}>{label}</button>
                ))}
              </div>

              <label className="dash-field">
                <span>最低社区等级</span>
                <input type="number" min="0" max="100" value={form.requiredLevel} onChange={(e) => patch("requiredLevel", e.target.value)} />
                <small>单位：等级，0 表示不限制。</small>
              </label>

              <label className="dash-field">
                <span>最低社区分数</span>
                <input type="number" min="0" max="100000" value={form.requiredScore} onChange={(e) => patch("requiredScore", e.target.value)} />
                <small>单位：分，0 表示不限制。</small>
              </label>

              <label className="dash-field">
                <span>领取消耗积分</span>
                <input type="number" min="0" max="999999" value={form.costPoints} onChange={(e) => patch("costPoints", e.target.value)} />
                <small>单位：积分，0 表示免费领取。</small>
              </label>

              <label className="dash-switch">
                <span>是否限制相同 IP</span>
                <input type="checkbox" checked={form.ipLimit} onChange={(e) => patch("ipLimit", e.target.checked)} />
              </label>

              <label className="dash-field dash-field--full">
                <span>项目描述</span>
                <textarea rows="4" value={form.description} onChange={(e) => patch("description", e.target.value)} placeholder="填写项目说明、领取规则或兑换说明" />
              </label>
            </div>
          ) : null}

          {tab === "distribution" ? (
            <div className="dash-form-grid">
              <label className="dash-field">
                <span>CDK 类型 <b>*</b></span>
                <select value={form.codeType} onChange={(e) => patch("codeType", e.target.value)}>
                  <option value="single_use">一次性激活码</option>
                  <option value="generic">通用兑换码</option>
                  <option value="batch_unique">批量独立码</option>
                </select>
              </label>

              <label className="dash-field">
                <span>CDK 数量</span>
                <input type="number" min="1" max="100000" value={form.codeCount} onChange={(e) => patch("codeCount", e.target.value)} />
                {errors.codeCount ? <em>{errors.codeCount}</em> : <small>范围：1 - 100000 个。</small>}
              </label>

              <label className="dash-field">
                <span>单用户领取次数</span>
                <input type="number" min="0" max="999" value={form.userLimit} onChange={(e) => patch("userLimit", e.target.value)} />
                <small>0 表示不限制，公开项目建议为 1。</small>
              </label>

              <label className="dash-field">
                <span>总库存</span>
                <input type="number" min="1" max="100000" value={form.totalStock} onChange={(e) => patch("totalStock", e.target.value)} />
                {errors.totalStock ? <em>{errors.totalStock}</em> : <small>单位：份，不能小于 1。</small>}
              </label>

              <label className="dash-field">
                <span>每日领取限制</span>
                <input type="number" min="0" max="100000" value={form.dailyLimit} onChange={(e) => patch("dailyLimit", e.target.value)} />
                <small>0 表示不限制每日领取总量。</small>
              </label>

              <label className="dash-field">
                <span>发放内容类型</span>
                <select value={form.rewardType} onChange={(e) => patch("rewardType", e.target.value)}>
                  <option value="text">文本内容</option>
                  <option value="benefit_package">权益包</option>
                  <option value="points">积分</option>
                  <option value="membership_days">会员天数</option>
                  <option value="api_key">API Key</option>
                  <option value="redeem_link">兑换链接</option>
                </select>
              </label>

              <label className="dash-field dash-field--full">
                <span>发放内容配置</span>
                <textarea rows="3" value={form.rewardContent} onChange={(e) => patch("rewardContent", e.target.value)} placeholder="例如：积分 100、会员 7 天、权益包 ID、兑换链接等" />
              </label>

              <label className="dash-field">
                <span>领取成功提示文案</span>
                <input value={form.successMessage} onChange={(e) => patch("successMessage", e.target.value)} />
              </label>

              <label className="dash-field">
                <span>领取失败提示文案</span>
                <input value={form.failureMessage} onChange={(e) => patch("failureMessage", e.target.value)} />
              </label>
            </div>
          ) : null}

          {tab === "preview" ? (
            <div className="dash-preview">
              <h3>{form.projectName || "未命名项目"}</h3>
              <div className="dash-badge-row">
                {splitTags(form.tagsText).map((tag) => <span key={tag}>{tag}</span>)}
                <span>{CODE_TYPE_LABELS[form.codeType]}</span>
                <span>{REWARD_TYPE_LABELS[form.rewardType]}</span>
                <span className={form.ipLimit ? "" : "is-danger"}>{form.ipLimit ? "限制 IP" : "不限 IP"}</span>
              </div>
              <dl>
                <div><dt>有效期</dt><dd>{form.startTime || "--"} 至 {form.endTime || "--"}</dd></div>
                <div><dt>领取门槛</dt><dd>等级 ≥ {form.requiredLevel || 0}，分数 ≥ {form.requiredScore || 0}</dd></div>
                <div><dt>库存策略</dt><dd>CDK {form.codeCount} 个，总库存 {form.totalStock}，每日 {Number(form.dailyLimit) || "不限"}</dd></div>
                <div><dt>单用户限制</dt><dd>{Number(form.userLimit) || "不限"} 次</dd></div>
                <div><dt>消耗积分</dt><dd>{form.costPoints || 0} 积分</dd></div>
              </dl>
              {highRisk ? <p className="dash-risk">高风险设置：当前存在不限 IP、不限制领取次数或通用码，公开项目可能被刷领。</p> : null}
            </div>
          ) : null}
        </div>

        <div className="dash-modal__foot">
          <button type="button" className="dash-ghost-btn" onClick={onClose}>取消</button>
          {step > 0 ? <button type="button" className="dash-ghost-btn" onClick={() => {
            const prev = step - 1;
            setStep(prev);
            setTab(prev === 0 ? "basic" : "distribution");
          }}>上一步</button> : null}
          {step < 2 ? (
            <button type="button" className="dash-primary-btn" onClick={next}>下一步</button>
          ) : (
            <button type="button" className="dash-primary-btn" disabled={loading} onClick={create}>
              {loading ? "创建中..." : "创建项目"}
            </button>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
}

export default function AdminPanel({ pushToast }) {
  const [state, setState] = useState(null);
  const [range, setRange] = useState("7天");
  const [chartMode, setChartMode] = useState("claims");
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState("");

  useEffect(() => {
    let alive = true;

    async function loadState() {
      try {
        const data = await api.getAdminState();
        if (alive) {
          setState(data);
          setLastSyncedAt(new Date().toLocaleTimeString("zh-CN"));
        }
      } catch (error) {
        if (alive) {
          pushToast({ tone: "error", title: "同步失败", message: error.message });
        }
      }
    }

    void loadState();
    const timer = window.setInterval(loadState, 3000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [pushToast]);

  const stats = state?.stats;
  const linkPreview = state?.linkPreview || [];
  const recentClaims = state?.recentClaims || [];
  const metrics = useMemo(() => ([
    { label: "总用户数", value: "12,846", desc: `${range}新增 328 用户`, trend: "+12.5%", tone: "good" },
    { label: "总项目数", value: "146", desc: "当前活动批次 1 个", trend: "+8", tone: "good" },
    { label: "总领取数", value: stats?.claimedTotal ?? "--", desc: `库存总量 ${stats?.importedTotal ?? "--"}`, trend: "+18.2%", tone: "good" },
    { label: "最近领取数", value: recentClaims.length, desc: `最后同步 ${lastSyncedAt || "--"}`, trend: "+246", tone: "warn" },
  ]), [range, stats, recentClaims.length, lastSyncedAt]);

  const hotProjects = [
    { name: state?.activity?.title || "当前 CDK 项目", value: stats?.claimedTotal ?? 0, desc: `剩余 ${stats?.remainingTotal ?? "--"} / ${stats?.distributableTotal ?? "--"}` },
    { name: "批量独立码池", value: linkPreview.length, desc: "可复制专属链接" },
    { name: "CodeHub 内测权益", value: "Mock", desc: "示例项目" },
  ];
  const activeCreators = [
    { name: "Admin", value: "32", desc: "创建项目" },
    { name: "运营组", value: "18", desc: "维护中" },
    { name: "增长组", value: "11", desc: "近 7 天活跃" },
  ];
  const activeClaimers = recentClaims.length
    ? recentClaims.map((record) => ({ name: record.account || "匿名领取", value: "1", desc: formatTime(record.time) }))
    : [{ name: "暂无领取者", value: "0", desc: "等待首条领取记录" }];

  async function handleCreateProject(form) {
    setSaving(true);
    try {
      const payload = {
        id: "__new__",
        stationName: "CodeHub 兑换码平台",
        title: form.projectName.trim(),
        tags: splitTags(form.tagsText),
        codeType: form.codeType,
        rewardType: form.rewardType,
        rewardContent: form.rewardContent.trim(),
        successMessage: form.successMessage.trim(),
        failureMessage: form.failureMessage.trim(),
        description: form.description,
        startTime: toISOValue(form.startTime),
        endTime: toISOValue(form.endTime),
        maxCount: Number(form.totalStock),
        limit: {
          limitPerIP: form.ipLimit,
          limitPerAccount: Number(form.userLimit) === 1,
          limitPerDevice: true,
          loginRequired: false,
          bindAccount: false,
          allowReclaimAfterExpired: false,
          whitelistEnabled: false,
          blacklistEnabled: true,
          captchaRequired: false,
          userLimit: Number(form.userLimit),
          dailyLimit: Number(form.dailyLimit),
          requiredLevel: Number(form.requiredLevel),
          requiredScore: Number(form.requiredScore),
          costPoints: Number(form.costPoints),
        },
      };
      let nextState = await api.updateActivity(payload);
      if (Number(form.codeCount) > 0) {
        const generated = await api.generateCodes({
          prefix: "CH",
          count: Math.min(Number(form.codeCount), 500),
          segments: 3,
          segmentLength: 4,
        });
        nextState = generated.state;
      }
      setState(nextState);
      setModalOpen(false);
      const firstLink = nextState?.linkPreview?.[0] ? buildClaimUrl(nextState.linkPreview[0]) : "";
      pushToast({ tone: "success", title: "项目创建成功", message: "项目已创建，CDK 专属链接已生成。" });
      return firstLink;
    } catch (error) {
      pushToast({ tone: "error", title: "创建失败", message: error.message });
      return "";
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="codehub-dashboard">
      <header className="dash-header">
        <div className="dash-brand">
          <span>CH</span>
          <div>
            <h1>CodeHub 兑换码平台</h1>
            <p>CDK 项目管理与领取数据看板</p>
          </div>
        </div>

        <div className="dash-actions">
          <div className="dash-segment">
            {["7天", "15天", "30天"].map((item) => (
              <button key={item} type="button" className={range === item ? "is-active" : ""} onClick={() => setRange(item)}>{item}</button>
            ))}
          </div>
          <button type="button" className="dash-primary-btn" onClick={() => setModalOpen(true)}>新建项目</button>
          <div className="dash-user">
            <span>A</span>
            <strong>Admin</strong>
          </div>
        </div>
      </header>

      <section className="dash-metric-grid">
        {metrics.map((metric) => <MetricCard key={metric.label} {...metric} />)}
      </section>

      <section className="dash-chart-grid">
        <article className="dash-panel dash-panel--trend">
          <div className="dash-panel__head">
            <div>
              <h2>核心趋势</h2>
              <p>{range}数据变化</p>
            </div>
            <div className="dash-segment dash-segment--small">
              <button type="button" className={chartMode === "claims" ? "is-active" : ""} onClick={() => setChartMode("claims")}>领取趋势</button>
              <button type="button" className={chartMode === "users" ? "is-active" : ""} onClick={() => setChartMode("users")}>用户增长</button>
            </div>
          </div>
          <LineChart mode={chartMode} />
        </article>

        <article className="dash-panel dash-panel--dist">
          <div className="dash-panel__head">
            <div>
              <h2>项目状态分布</h2>
              <p>标签与库存概览</p>
            </div>
            <span className="dash-badge">实时</span>
          </div>
          <DonutChart stats={stats} />
        </article>
      </section>

      <section className="dash-bottom-grid">
        <RankingCard title="热门项目" items={hotProjects} />
        <RankingCard title="活跃创建者" items={activeCreators} />
        <RankingCard title="活跃领取者" items={activeClaimers} />
      </section>

      <ProjectModal
        visible={modalOpen}
        state={state}
        loading={saving}
        onClose={() => setModalOpen(false)}
        onSubmit={handleCreateProject}
        pushToast={pushToast}
      />
    </div>
  );
}
