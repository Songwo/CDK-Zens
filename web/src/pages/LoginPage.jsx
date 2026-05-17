import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { isLoggedIn, publicApi, communityApi } from "../lib/api";
import { DEFAULT_BRAND, normalizeBrand, setAppTitle } from "../lib/brand";

export default function LoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const returnUrl = searchParams.get("returnUrl") || "/admin";

  const [error, setError] = useState("");
  const [ssoLoading, setSsoLoading] = useState(false);
  const [brand, setBrand] = useState(DEFAULT_BRAND);

  useEffect(() => {
    setAppTitle(DEFAULT_BRAND);
    // 如果已登录，直接跳转
    if (isLoggedIn()) {
      navigate(returnUrl, { replace: true });
      return;
    }
    publicApi.brand().then((data) => {
      if (data?.systemName) {
        const next = normalizeBrand(data);
        setBrand(next);
        setAppTitle(next);
      }
    }).catch(() => {});
  }, []);

  async function handleCommunityLogin() {
    setSsoLoading(true);
    setError("");
    try {
      const config = await communityApi.config();
      const communityUrl = config.communityUrl;
      const clientId = config.clientId;

      if (!communityUrl || !clientId) {
        throw new Error("社区 SSO 配置未就绪，请联系管理员");
      }

      // 把原始 returnUrl 写入 sessionStorage，让 SSO 回调页登录成功后跳回原页（例如领取页）。
      try {
        if (returnUrl && returnUrl !== "/admin") {
          sessionStorage.setItem("sso_return_url", returnUrl);
        } else {
          sessionStorage.removeItem("sso_return_url");
        }
      } catch { /* sessionStorage 可能被禁用，忽略 */ }

      const callbackUrl = `${window.location.origin}/login/callback`;
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
          <img src="/logo.png" alt="Logo" width="56" height="56" className="login-card__logo" style={{ objectFit: 'cover', background: 'transparent', boxShadow: 'none' }} />
          <h1>{brand.systemName}</h1>
          <p>{brand.brandEnglishName}</p>
        </div>

        <p style={{ textAlign: 'center', color: 'var(--cp-muted)', fontSize: '14px', margin: '0 0 24px', lineHeight: 1.6 }}>
          本站仅支持通过 Zens 社区账号进行身份验证
        </p>

        {error && <div style={{background: '#fef2f2', color: '#dc2626', padding: '12px', borderRadius: '6px', marginBottom: '24px', fontSize: '14px', textAlign: 'center'}}>{error}</div>}

        <button
          type="button"
          className="btn btn--primary"
          style={{ width: '100%', padding: '14px', fontSize: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '10px' }}
          onClick={handleCommunityLogin}
          disabled={ssoLoading}
        >
          {ssoLoading ? (
            <>
              <span className="sso-spinner" />
              正在跳转到社区...
            </>
          ) : (
            <>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
                <polyline points="10 17 15 12 10 7" />
                <line x1="15" y1="12" x2="3" y2="12" />
              </svg>
              使用 Zens 社区账号一键登录
            </>
          )}
        </button>

        <div style={{ marginTop: '32px', textAlign: 'center', fontSize: '13px', color: 'var(--cp-faint)' }}>
          <p style={{ margin: 0 }}>没有社区账号？请先在 Zens 社区完成注册</p>
        </div>
      </div>
    </div>
  );
}
