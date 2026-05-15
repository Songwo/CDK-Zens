import { useEffect, useRef, useState, useCallback, useMemo } from "react";
import { useParams } from "react-router-dom";
import { claimApi, isLoggedIn, publicApi, userApi, clearAuthToken } from "../lib/api";
import { DEFAULT_BRAND, normalizeBrand, setAppTitle } from "../lib/brand";
import { getFingerprint } from "../lib/storage";
import ParticleBurst from "../components/ParticleBurst";
import HCaptchaBox from "../components/HCaptchaBox";

const CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$*&!";

function formatTime(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

/* 内联 CDK 解密动画组件 */
function InlineCDKReveal({ code, claimedAt }) {
  const [chars, setChars] = useState([]);
  const [done, setDone] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!code) return;
    const len = code.length;
    setChars(Array.from({ length: len }, () => ({ c: CHARS[Math.floor(Math.random() * CHARS.length)], ok: false })));
    let idx = 0;
    const t = setInterval(() => {
      setChars(prev => {
        const next = [...prev];
        for (let i = idx; i < len; i++) {
          next[i] = { c: /[\s/:.\-]/.test(code[i]) ? code[i] : CHARS[Math.floor(Math.random() * CHARS.length)], ok: false };
        }
        if (idx < len) next[idx] = { c: code[idx], ok: true };
        return next;
      });
      idx++;
      if (idx > len) { clearInterval(t); setDone(true); }
    }, 40);
    return () => clearInterval(t);
  }, [code]);

  async function copy() {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <div className="cdk-reveal">
      <div className="cdk-reveal__header">
        <span>兑换信息</span>
        <button onClick={copy} disabled={!code}>{copied ? "✓ 已复制" : "复制内容"}</button>
      </div>
      <div className={`cdk-reveal__code ${done ? "cdk-reveal__code--done" : ""}`}>
        {!code && <span className="cdk-reveal__waiting">&gt;&gt; 等待获取兑换链接 / 口令...</span>}
        {code && chars.map((item, i) => (
          <span key={i} className={item.ok ? "cdk-char--resolved" : "cdk-char--scramble"}>{item.c}</span>
        ))}
      </div>
      {claimedAt && <p className="cdk-reveal__hint">锁定时间 {new Date(claimedAt).toLocaleString("zh-CN")}</p>}
    </div>
  );
}

/* 当前登录账号徽章 */
function CurrentUserBadge({ user, loggedIn }) {
  if (!loggedIn) {
    return (
      <a
        href={`/login?returnUrl=${encodeURIComponent(window.location.pathname)}`}
        className="claim-userbadge claim-userbadge--guest"
        title="点击前往登录"
      >
        <span className="claim-userbadge__avatar claim-userbadge__avatar--guest">?</span>
        <div className="claim-userbadge__meta">
          <span className="claim-userbadge__label">未登录</span>
          <span className="claim-userbadge__hint">点击使用社区账号登录</span>
        </div>
      </a>
    );
  }
  if (!user) {
    return (
      <div className="claim-userbadge claim-userbadge--loading">
        <span className="claim-userbadge__avatar claim-userbadge__avatar--guest">…</span>
        <div className="claim-userbadge__meta">
          <span className="claim-userbadge__label">正在校验身份…</span>
        </div>
      </div>
    );
  }
  const display = user.nickname || user.username || user.email || "已登录用户";
  const sub = user.username && user.username !== display ? `@${user.username}` : (user.email || (user.role === 'admin' ? '管理员' : '社区账号'));
  const initial = (display || "?").trim().charAt(0).toUpperCase();
  return (
    <div className="claim-userbadge" title={`已登录：${display}`}>
      {user.avatar ? (
        <img src={user.avatar} alt={display} className="claim-userbadge__avatar" />
      ) : (
        <span className="claim-userbadge__avatar">{initial}</span>
      )}
      <div className="claim-userbadge__meta">
        <span className="claim-userbadge__label">{display}</span>
        <span className="claim-userbadge__hint">{sub}</span>
      </div>
    </div>
  );
}

/* 倒计时组件 */
function Countdown({ targetTime, onComplete }) {
  const [timeLeft, setTimeLeft] = useState(0);

  useEffect(() => {
    const end = new Date(targetTime).getTime();
    const update = () => {
      const now = Date.now();
      const diff = end - now;
      if (diff <= 0) {
        setTimeLeft(0);
        if (onComplete) onComplete();
      } else {
        setTimeLeft(diff);
      }
    };
    update();
    const t = setInterval(update, 1000);
    return () => clearInterval(t);
  }, [targetTime, onComplete]);

  if (timeLeft <= 0) return <span style={{color: 'var(--cp-brand)', fontWeight: 800}}>通道已开启</span>;

  const d = Math.floor(timeLeft / (1000 * 60 * 60 * 24));
  const h = Math.floor((timeLeft / (1000 * 60 * 60)) % 24);
  const m = Math.floor((timeLeft / 1000 / 60) % 60);
  const s = Math.floor((timeLeft / 1000) % 60);

  const pad = (n) => String(n).padStart(2, '0');

  return (
    <div className="countdown-box">
      <div className="countdown-item"><span className="val">{d}</span><span className="lbl">天</span></div>
      <div className="countdown-sep">:</div>
      <div className="countdown-item"><span className="val">{pad(h)}</span><span className="lbl">时</span></div>
      <div className="countdown-sep">:</div>
      <div className="countdown-item"><span className="val">{pad(m)}</span><span className="lbl">分</span></div>
      <div className="countdown-sep">:</div>
      <div className="countdown-item"><span className="val">{pad(s)}</span><span className="lbl">秒</span></div>
    </div>
  );
}

export default function ClaimPage() {
  const { projectCode } = useParams();
  const [project, setProject] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [claiming, setClaiming] = useState(false);
  const [claimResult, setClaimResult] = useState(null);
  const [burstTrigger, setBurstTrigger] = useState("");
  const [hcaptchaToken, setHcaptchaToken] = useState("");
  const [toast, setToast] = useState("");
  const [brand, setBrand] = useState(DEFAULT_BRAND);
  const [currentUser, setCurrentUser] = useState(null);
  const hcaptchaRef = useRef(null);
  const fingerprint = getFingerprint();
  const userScope = currentUser?.userId || currentUser?.id || "guest";
  const storageKey = useMemo(() => `claim-result:${userScope}:${projectCode}`, [userScope, projectCode]);

  useEffect(() => {
    setAppTitle(DEFAULT_BRAND);
    publicApi.brand().then((d) => {
      if (d?.systemName) {
        const next = normalizeBrand(d);
        setBrand(next);
        setAppTitle(next);
      }
    }).catch(() => {});
  }, []);

  // 拉取当前登录用户信息（用于在领取页展示当前账号）
  useEffect(() => {
    if (!isLoggedIn()) {
      setCurrentUser(null);
      return;
    }
    let cancelled = false;
    userApi.getMe()
      .then((data) => {
        if (cancelled) return;
        if (data?.loggedIn === false) {
          setCurrentUser(null);
          clearAuthToken();
          return;
        }
        setCurrentUser(data?.user || data || null);
      })
      .catch(() => { if (!cancelled) setCurrentUser(null); });
    return () => { cancelled = true; };
  }, []);

  // 切换账号或登出后，清理上一账号遗留在本地的领取缓存（避免串号）
  useEffect(() => {
    Object.keys(localStorage).forEach((key) => {
      if (key.startsWith("claim-result:") && !key.startsWith(`claim-result:${userScope}:`)) {
        // 历史的 `claim-result:<projectCode>` 旧格式也清掉
        const parts = key.split(":");
        if (parts.length === 2 || (parts.length === 3 && parts[1] !== userScope)) {
          localStorage.removeItem(key);
        }
      }
    });
  }, [userScope]);

  const fetchProject = useCallback(async () => {
    try {
      const data = await claimApi.getNode(projectCode, fingerprint).catch(() => claimApi.getProject(projectCode, fingerprint));
      setProject(data);
      // 后端在登录态下会返回 userInfo，作为兜底的当前用户信息源
      if (data?.userInfo && !currentUser) {
        setCurrentUser(data.userInfo);
      }
      // 后端按 userID 严格判定是否已领取；只要 userClaimed=true，刷新页面也只会显示既有兑换码
      if (data.userClaimed) {
        const normalized = {
          rewardContent: data.userRewardContent,
          code: data.userRewardContent,
          claimedAt: data.userClaimedAt,
        };
        setClaimResult(normalized);
        localStorage.setItem(storageKey, JSON.stringify(normalized));
        return;
      }
      // 未领取：清掉本地任何陈旧缓存，避免显示其他账号留下的兑换码
      setClaimResult(null);
      localStorage.removeItem(storageKey);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [projectCode, fingerprint, storageKey, currentUser]);

  useEffect(() => { fetchProject(); }, [fetchProject]);

  async function handleClaim() {
    setClaiming(true);
    setError("");
    try {
      const payload = {
        fingerprint,
        captchaProvider: project?.requireCaptcha ? "hcaptcha" : "",
        hcaptchaToken,
      };
      const result = await claimApi.submitNode(projectCode, payload).catch(() => claimApi.submit(projectCode, { fingerprint }));
      const normalized = normalizeClaimResult(result);
      setClaimResult(normalized);
      localStorage.setItem(storageKey, JSON.stringify(normalized));
      setBurstTrigger(normalized.code || normalized.rewardContent || "success");
      await fetchProject();
    } catch (err) {
      setError(err.message);
      if (["CAPTCHA_INVALID", "HCAPTCHA_INVALID", "CAPTCHA_EXPIRED"].includes(err.code)) {
        setHcaptchaToken("");
        hcaptchaRef.current?.resetCaptcha?.();
      }
    } finally {
      setClaiming(false);
    }
  }

  function getClaimState() {
    if (!project) return { disabled: true, text: "加载中..." };
    if (project.userClaimed || claimResult) return { disabled: true, text: "已领取" };
    if (!project.enabled || project.status === "disabled" || project.status === "paused" || project.status === "archived") return { disabled: true, text: "通道已关闭" };
    if (project.status === "upcoming" || project.status === "draft") return { disabled: true, text: "尚未激活" };
    if (project.status === "ended") return { disabled: true, text: "已过生命周期" };
    if (project.status === "soldout" || project.status === "exhausted" || project.remaining <= 0) return { disabled: true, text: "资源已耗尽" };
    if (!isLoggedIn()) return { disabled: true, text: "需要身份验证" };
    if (project.requireCaptcha && !hcaptchaToken) return { disabled: true, text: "请先完成人机验证" };
    if (claiming) return { disabled: true, text: "正在领取..." };
    return { disabled: false, text: "接入并领取" };
  }

  const claimState = getClaimState();

  /* ── 加载状态 ── */
  if (loading) {
    return (
      <div className="claim-shell">
        <div className="claim-shell__center">
          <div className="claim-loader"><div className="claim-loader__spinner" /></div>
          <p style={{color: 'var(--cp-muted)', marginTop: '16px'}}>正在连接分发节点...</p>
        </div>
      </div>
    );
  }

  /* ── 错误 / 404 ── */
  if (error && !project) {
    return (
      <div className="claim-shell">
        <div className="claim-shell__center">
          <h2 style={{fontSize: '28px', fontWeight: 800, color: 'var(--cp-ink)', marginBottom: '12px'}}>节点不可达</h2>
          <p style={{color: 'var(--cp-muted)'}}>{error}</p>
        </div>
      </div>
    );
  }

  // 严格拦截
  if (project.status === "disabled" || project.status === "paused" || project.status === "archived") {
    return (
      <div className="claim-shell">
        <div className="claim-shell__center">
          <h2 style={{fontSize: '28px', fontWeight: 800, color: 'var(--cp-ink)', marginBottom: '12px'}}>通道已关停</h2>
          <p style={{color: 'var(--cp-muted)'}}>该资源的释放已被管理员中断，无法继续访问。</p>
        </div>
      </div>
    );
  }

  const progress = project.totalStock > 0 ? Math.round((project.claimedCount / project.totalStock) * 100) : 0;
  const statusLabels = {
    active: "正在分发", upcoming: "等待激活", draft: "等待激活", ended: "周期结束", soldout: "额度耗尽", exhausted: "额度耗尽", disabled: "已关停", paused: "已暂停",
  };

  return (
    <div className="claim-shell">
      <ParticleBurst trigger={burstTrigger} />

      {/* 顶部品牌标识 */}
      <header className="claim-brand anim-fade-up" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '16px', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
          <img src="/logo.png" alt="Logo" width="48" height="48" className="claim-brand__logo" style={{ objectFit: 'cover', background: 'transparent', boxShadow: 'none' }} />
          <div>
            <h1>{brand.systemName}</h1>
            <span>{brand.brandEnglishName}</span>
          </div>
        </div>
        <CurrentUserBadge user={currentUser} loggedIn={isLoggedIn()} />
      </header>

      {/* 主内容区 — 全幅无卡片 */}
      <div className="claim-content anim-fade-up stagger-1">

        {/* 状态行 */}
        <div className="claim-status-line">
          <div style={{display: 'flex', alignItems: 'center', gap: '16px'}}>
            <span className={`status-dot status--${project.status}`}>{statusLabels[project.status] || project.status}</span>
            <span className="claim-time-range">{formatTime(project.startTime)} — {formatTime(project.endTime)}</span>
          </div>
          <div style={{fontSize: '12px', color: 'var(--cp-faint)', fontWeight: 600}}>PER USER LIMIT: {project.perUserLimit}X</div>
        </div>

        {/* 倒计时 (如果即将开启) — 移动到标题上方更显眼 */}
        {(project.status === "upcoming" || project.status === "draft") && project.startTime && (
          <div style={{display: 'flex', flexDirection: 'column', alignItems: 'center', marginBottom: '40px', background: 'var(--cp-surface-hover)', padding: '32px', borderRadius: '16px', border: '1px solid var(--cp-brand-glow)'}}>
            <h3 style={{fontSize: '13px', color: 'var(--cp-brand)', fontWeight: 800, textTransform: 'uppercase', letterSpacing: '1.5px', marginBottom: '20px'}}>距离分发网络开启还剩</h3>
            <Countdown targetTime={project.startTime} onComplete={() => fetchProject()} />
          </div>
        )}

        {/* 项目标题 */}
        <h2 className="claim-title">{project.name}</h2>

        {/* 主体区域 */}
        <div style={{flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0}}>
          <div style={{display: 'grid', gridTemplateColumns: project.rules ? '1.5fr 1fr' : '1fr', gap: '32px', flex: 1, minHeight: 0}}>
            <div>
              {project.description && (
                <p className="claim-desc">{project.description}</p>
              )}
              
              {/* 数据带 */}
              <div className="claim-data-band anim-fade-up stagger-2">
                <div className="claim-data-band__item" style={{paddingLeft: 0}}>
                  <span className="claim-data-band__label">已释放负载</span>
                  <span className="claim-data-band__value">{project.claimedCount}</span>
                </div>
                <div className="claim-data-band__item">
                  <span className="claim-data-band__label">当前保留量</span>
                  <span className="claim-data-band__value">{project.remaining}</span>
                </div>
                <div className="claim-data-band__item">
                  <span className="claim-data-band__label">节点总容量</span>
                  <span className="claim-data-band__value">{project.totalStock}</span>
                </div>
              </div>

              {/* 进度条 */}
              <div className="claim-track anim-fade-up stagger-2">
                <div className="track-bar" style={{height: '10px', borderRadius: '5px'}}>
                  <div className="track-bar__fill" style={{width: `${progress}%`}} />
                </div>
                <div className="track-meta">
                  <span style={{fontWeight: 600}}>负载释放率 {progress}%</span>
                {project.needLogin && <span style={{color: 'var(--cp-brand)', fontWeight: 600}}>需身份认证</span>}
                </div>
              </div>
            </div>

            {project.rules && (
              <div className="claim-rules anim-fade-up stagger-3">
                <h4>使用条款与协议</h4>
                <p>{project.rules}</p>
              </div>
            )}
          </div>
        </div>

        {/* 底部领取区域 */}
        <div className="claim-action anim-fade-up stagger-3" style={{padding: '24px 0 0', borderTop: '1px solid var(--cp-divider)', marginTop: '24px', flexShrink: 0}}>
          {claimResult ? (
            <div className="claim-result" style={{padding: 0, border: 'none', textAlign: 'center'}}>
              <div className="claim-result__label" style={{fontSize: '16px', color: 'var(--cp-brand)', marginBottom: '12px'}}>领取成功 · CDK 已解锁</div>
              {currentUser && (
                <div style={{fontSize: '13px', color: 'var(--cp-muted)', marginBottom: '20px'}}>
                  本兑换码已绑定至账号 <b style={{color: 'var(--cp-ink)'}}>{currentUser.nickname || currentUser.username}</b>，刷新页面不会重新发放新码。
                </div>
              )}
              <InlineCDKReveal code={claimResult.code || claimResult.rewardContent} claimedAt={claimResult.claimedAt} />
            </div>
          ) : (
            <div style={{display: 'flex', flexDirection: 'column', alignItems: 'center'}}>
              {project.requireCaptcha && (
                <HCaptchaBox
                  ref={hcaptchaRef}
                  onVerify={setHcaptchaToken}
                  onExpire={() => setHcaptchaToken("")}
                  onError={() => setHcaptchaToken("")}
                />
              )}
              <button
                className={`btn ${claimState.disabled ? "btn--secondary" : "btn--primary"} claim-action__btn`}
                disabled={claimState.disabled}
                onClick={handleClaim}
                style={{height: '48px', maxWidth: '320px', fontSize: '16px', width: '100%'}}
              >
                {project.buttonText && !claimState.disabled && !claiming ? project.buttonText : claimState.text}
              </button>
              {error && <p className="claim-error" style={{textAlign: 'center'}}>{error}</p>}
            </div>
          )}
        </div>

        {/* 登录提示 - 所有用户必须登录 */}
        {!isLoggedIn() && !claimResult && (
          <div className="claim-login-hint" style={{marginTop: '16px', padding: '16px 0 0', textAlign: 'center', borderTop: '1px solid var(--cp-divider)', flexShrink: 0}}>
            <p style={{margin: 0, display: 'inline-block', color: 'var(--cp-muted)'}}>此通道需要经过身份验证才能接入</p>
            <a href={`/login?returnUrl=${encodeURIComponent(window.location.pathname)}`} className="btn btn--text" style={{padding: '0 12px', color: 'var(--cp-brand)', fontWeight: 700}}>
              前往验证身份 →
            </a>
          </div>
        )}
      </div>

      {/* 底部 */}
      <footer className="claim-footer-bar">
        Powered by {brand.systemName} · {brand.brandEnglishName}
      </footer>
      {toast && <div className="toast" onAnimationEnd={() => {}}>{toast}</div>}
    </div>
  );
}

function normalizeClaimResult(result) {
  return {
    claimId: result.claimId,
    claimToken: result.claimToken,
    campaignId: result.campaignId,
    nodeId: result.nodeId,
    code: result.code || result.rewardContent,
    rewardContent: result.rewardContent || result.code,
    claimedAt: result.claimedAt,
  };
}
