import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { setAuthToken, publicApi, communityApi } from "../lib/api";

export default function LoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const returnUrl = searchParams.get("returnUrl") || "/admin";

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [ssoLoading, setSsoLoading] = useState(false);
  const [brand, setBrand] = useState({ systemName: "缪盒空投台", brandEnglishName: "MiuBox Airdrop Hub", logoText: "MB" });

  useEffect(() => {
    publicApi.brand().then((data) => {
      if (data?.systemName) setBrand(data);
    }).catch(() => {});
  }, []);

  async function handleLogin(e) {
    e.preventDefault();
    if (!username || !password) {
      setError("请输入账号和密码");
      return;
    }
    setLoading(true);
    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.message || "登录失败");
      setAuthToken(data.data?.token || data.token);
      navigate(returnUrl);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  async function handleCommunityLogin() {
    setSsoLoading(true);
    setError("");
    try {
      // 获取社区 SSO 配置
      const config = await communityApi.config();
      const communityUrl = config.communityUrl;
      const clientId = config.clientId;

      if (!communityUrl || !clientId) {
        throw new Error("社区 SSO 配置未就绪，请联系管理员");
      }

      // 构建 CDK 回调地址
      const callbackUrl = `${window.location.origin}/login/callback`;

      // 跳转到社区 SSO 授权页
      const ssoUrl = `${communityUrl}/sso/authorize?client_id=${encodeURIComponent(clientId)}&redirect_uri=${encodeURIComponent(callbackUrl)}`;
      window.location.href = ssoUrl;
    } catch (err) {
      setError(err.message || "无法跳转到社区登录");
      setSsoLoading(false);
    }
  }

  return (
    <div className="login-layout">
      <div className="login-card anim-fade-up">
        <div className="login-card__brand">
          <div className="login-card__logo">{brand.logoText}</div>
          <h1>{brand.systemName}</h1>
          <p>{brand.brandEnglishName} 身份验证</p>
        </div>

        {error && <div style={{background: '#fef2f2', color: '#dc2626', padding: '12px', borderRadius: '6px', marginBottom: '24px', fontSize: '14px', textAlign: 'center'}}>{error}</div>}

        <form onSubmit={handleLogin} style={{display: 'flex', flexDirection: 'column', gap: '16px'}}>
          <input 
            className="styled-input" 
            placeholder="管理员或终端用户名" 
            value={username} 
            onChange={e => setUsername(e.target.value)} 
          />
          <input 
            type="password" 
            className="styled-input" 
            placeholder="访问凭证 (密码)" 
            value={password} 
            onChange={e => setPassword(e.target.value)} 
          />
          <button type="submit" className="btn btn--primary" style={{padding: '12px', fontSize: '15px'}} disabled={loading || ssoLoading}>
            {loading ? "验证中..." : "接入系统"}
          </button>
        </form>

        <div style={{marginTop: '24px', position: 'relative', textAlign: 'center'}}>
          <div style={{position: 'absolute', top: '50%', left: 0, right: 0, borderTop: '1px solid var(--cp-divider)', zIndex: 0}} />
          <span style={{background: 'var(--cp-bg)', padding: '0 12px', position: 'relative', zIndex: 1, fontSize: '12px', color: 'var(--cp-faint)'}}>或者</span>
        </div>

        <button 
          type="button" 
          className="btn btn--secondary" 
          style={{width: '100%', marginTop: '24px', padding: '12px', color: 'var(--cp-brand)', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px'}} 
          onClick={handleCommunityLogin}
          disabled={loading || ssoLoading}
        >
          {ssoLoading ? (
            <>
              <span className="sso-spinner" />
              正在跳转到社区...
            </>
          ) : (
            <>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
                <polyline points="10 17 15 12 10 7" />
                <line x1="15" y1="12" x2="3" y2="12" />
              </svg>
              使用 Zens 社区账号登录
            </>
          )}
        </button>

        <div style={{marginTop: '32px', textAlign: 'center', fontSize: '13px'}}>
          <span style={{color: 'var(--cp-muted)'}}>尚未拥有身份标识？</span>
          <a href="/register" style={{color: 'var(--cp-brand)', fontWeight: 600, marginLeft: '8px'}}>注册终端</a>
        </div>
      </div>
    </div>
  );
}
