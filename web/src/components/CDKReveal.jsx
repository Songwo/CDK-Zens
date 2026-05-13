import { useEffect, useState } from "react";

const CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$*&!";

export default function CDKReveal({ code, claimedAt }) {
  const [displayedChars, setDisplayedChars] = useState([]);
  const [flash, setFlash] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    setDisplayedChars([]);
    setFlash(false);
    setCopied(false);

    if (!code) {
      return undefined;
    }

    const length = code.length;
    const initialChars = Array.from({ length }, () => ({
      char: CHARS[Math.floor(Math.random() * CHARS.length)],
      resolved: false
    }));
    
    setDisplayedChars(initialChars);

    let resolveIndex = 0;
    const timer = window.setInterval(() => {
      setDisplayedChars(current => {
        const next = [...current];
        for (let i = resolveIndex; i < length; i++) {
          if (code[i] === ' ' || code[i] === '/' || code[i] === ':' || code[i] === '.') {
            next[i] = { char: code[i], resolved: false };
          } else {
            next[i] = { char: CHARS[Math.floor(Math.random() * CHARS.length)], resolved: false };
          }
        }
        if (resolveIndex < length) {
           next[resolveIndex] = { char: code[resolveIndex], resolved: true };
        }
        return next;
      });

      resolveIndex++;
      
      if (resolveIndex > length) {
        window.clearInterval(timer);
        setFlash(true);
        window.setTimeout(() => setFlash(false), 800);
      }
    }, 45);

    return () => window.clearInterval(timer);
  }, [code]);

  async function handleCopy() {
    if (!code) {
      return;
    }

    await navigator.clipboard.writeText(code);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  return (
    <div className="reveal-panel">
      <div className="reveal-panel__bar">
        <span>兑换信息面板</span>
        <button
          type="button"
          disabled={!code}
          onClick={handleCopy}
          className={`app-button app-button--text ${copied ? "is-success" : ""}`}
        >
          {copied ? "✓ 已复制" : "复制内容"}
        </button>
      </div>
      <div className={`reveal-panel__line ${flash ? "is-flash" : ""}`}>
        {!code && <span className="reveal-prefix">&gt;&gt;</span>}
        {!code && <span style={{ marginLeft: '10px' }}>等待获取兑换链接/口令...</span>}
        {code && (
           <span>
             {displayedChars.map((item, idx) => (
                <span key={idx} className={`decode-char ${item.resolved ? 'is-resolved' : ''}`}>
                  {item.char}
                </span>
             ))}
           </span>
        )}
        {!code && <span className="reveal-cursor" />}
      </div>
      <p className="reveal-panel__hint">
        {claimedAt ? `锁定时间 ${new Date(claimedAt).toLocaleString("zh-CN")}` : "成功获取后自动解密显示。"}
      </p>
    </div>
  );
}
