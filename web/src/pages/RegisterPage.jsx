import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { setAuthToken, publicApi } from "../lib/api";
import { DEFAULT_BRAND, normalizeBrand, setAppTitle } from "../lib/brand";

export default function RegisterPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [brand, setBrand] = useState(DEFAULT_BRAND);

  useEffect(() => {
    setAppTitle(DEFAULT_BRAND);
    publicApi.brand().then((data) => {
      if (data?.systemName) {
        const next = normalizeBrand(data);
        setBrand(next);
        setAppTitle(next);
      }
    }).catch(() => {});
  }, []);

  async function handleRegister(e) {
    e.preventDefault();
    if (!username || !password) {
      setError("请输入账号和密码");
      return;
    }
    setLoading(true);
    try {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.message || "注册失败");
      setAuthToken(data.data?.token || data.token);
      navigate("/admin");
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-layout">
      <div className="login-card anim-fade-up">
        <div className="login-card__brand">
          <div className="login-card__logo">{brand.logoText}</div>
          <h1>{brand.systemName}</h1>
          <p>{brand.brandEnglishName} 终端注册</p>
        </div>

        {error && <div style={{background: '#fef2f2', color: '#dc2626', padding: '12px', borderRadius: '6px', marginBottom: '24px', fontSize: '14px', textAlign: 'center'}}>{error}</div>}

        <form onSubmit={handleRegister} style={{display: 'flex', flexDirection: 'column', gap: '16px'}}>
          <input 
            className="styled-input" 
            placeholder="自定义用户名" 
            value={username} 
            onChange={e => setUsername(e.target.value)} 
          />
          <input 
            type="password" 
            className="styled-input" 
            placeholder="安全访问凭证 (密码)" 
            value={password} 
            onChange={e => setPassword(e.target.value)} 
          />
          <button type="submit" className="btn btn--primary" style={{padding: '12px', fontSize: '15px'}} disabled={loading}>
            {loading ? "登记中..." : "确立身份标识"}
          </button>
        </form>

        <div style={{marginTop: '32px', textAlign: 'center', fontSize: '13px'}}>
          <span style={{color: 'var(--cp-muted)'}}>已拥有身份？</span>
          <a href="/login" style={{color: 'var(--cp-brand)', fontWeight: 600, marginLeft: '8px'}}>返回接入</a>
        </div>
      </div>
    </div>
  );
}
