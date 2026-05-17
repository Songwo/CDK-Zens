import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { setAuthToken, communityApi } from "../lib/api";

export default function LoginCallbackPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [status, setStatus] = useState("processing"); // processing | success | error
  const [error, setError] = useState("");
  const [returnTarget, setReturnTarget] = useState("/admin");

  useEffect(() => {
    const ssoToken = searchParams.get("sso_token");
    const errorParam = searchParams.get("error");

    // 优先从 query 里取 returnUrl（如果 SSO 透传），否则回退到登录前保存的 sessionStorage。
    let target = searchParams.get("returnUrl") || "";
    if (!target) {
      try { target = sessionStorage.getItem("sso_return_url") || ""; } catch { target = ""; }
    }
    // 只允许相对路径，避免开放重定向风险。
    const safeTarget = target && target.startsWith("/") && !target.startsWith("//") ? target : "/admin";
    setReturnTarget(safeTarget);

    if (errorParam) {
      setStatus("error");
      setError(errorParam === "access_denied" ? "用户拒绝了授权" : `授权失败: ${errorParam}`);
      return;
    }

    if (!ssoToken) {
      setStatus("error");
      setError("缺少 SSO Token 参数");
      return;
    }

    // 使用 SSO Token 登录
    communityApi.login(ssoToken)
      .then((data) => {
        const token = data.data?.token || data.token;
        if (!token) throw new Error("未返回有效的登录凭证");
        setAuthToken(token);
        try { sessionStorage.removeItem("sso_return_url"); } catch { /* ignore */ }
        setStatus("success");
        // 短暂展示成功状态后跳转
        setTimeout(() => {
          navigate(safeTarget, { replace: true });
        }, 800);
      })
      .catch((err) => {
        setStatus("error");
        setError(err.message || "SSO 登录失败");
      });
  }, [searchParams, navigate]);

  const isClaimReturn = returnTarget.startsWith("/claim/");

  return (
    <div className="login-layout">
      <div className="login-card anim-fade-up" style={{ textAlign: "center", padding: "48px 36px" }}>
        {status === "processing" && (
          <>
            <div className="sso-callback-spinner" />
            <h2 style={{ margin: "24px 0 8px", fontSize: "18px", color: "var(--cp-text)" }}>
              正在验证社区身份...
            </h2>
            <p style={{ color: "var(--cp-muted)", fontSize: "14px", margin: 0 }}>
              请稍候，正在同步您的社区账号信息
            </p>
          </>
        )}

        {status === "success" && (
          <>
            <div className="sso-callback-success">✓</div>
            <h2 style={{ margin: "24px 0 8px", fontSize: "18px", color: "var(--cp-text)" }}>
              登录成功！
            </h2>
            <p style={{ color: "var(--cp-muted)", fontSize: "14px", margin: 0 }}>
              {isClaimReturn ? "即将返回领取页…" : "即将进入管理后台..."}
            </p>
          </>
        )}

        {status === "error" && (
          <>
            <div className="sso-callback-error">✕</div>
            <h2 style={{ margin: "24px 0 8px", fontSize: "18px", color: "var(--cp-text)" }}>
              登录失败
            </h2>
            <p style={{ color: "#dc2626", fontSize: "14px", margin: "0 0 24px" }}>
              {error}
            </p>
            <button
              className="btn btn--primary"
              style={{ padding: "10px 32px" }}
              onClick={() => navigate("/login", { replace: true })}
            >
              返回登录页
            </button>
          </>
        )}
      </div>

      <style>{`
        .sso-callback-spinner {
          width: 48px;
          height: 48px;
          border: 3px solid var(--cp-divider, #e5e7eb);
          border-top: 3px solid var(--cp-brand, #f5c542);
          border-radius: 50%;
          margin: 0 auto;
          animation: sso-spin 0.8s linear infinite;
        }
        @keyframes sso-spin {
          to { transform: rotate(360deg); }
        }
        .sso-callback-success {
          width: 56px;
          height: 56px;
          border-radius: 50%;
          background: linear-gradient(135deg, #10b981, #059669);
          color: #fff;
          font-size: 28px;
          font-weight: 700;
          display: flex;
          align-items: center;
          justify-content: center;
          margin: 0 auto;
          animation: sso-pop 0.3s ease-out;
        }
        .sso-callback-error {
          width: 56px;
          height: 56px;
          border-radius: 50%;
          background: #fef2f2;
          color: #ef4444;
          font-size: 28px;
          font-weight: 700;
          display: flex;
          align-items: center;
          justify-content: center;
          margin: 0 auto;
        }
        @keyframes sso-pop {
          0% { transform: scale(0.5); opacity: 0; }
          100% { transform: scale(1); opacity: 1; }
        }
      `}</style>
    </div>
  );
}
