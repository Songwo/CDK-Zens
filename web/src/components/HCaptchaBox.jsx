import { forwardRef, useImperativeHandle, useRef, useState } from "react";
import HCaptcha from "@hcaptcha/react-hcaptcha";

const TEST_SITE_KEY = "10000000-ffff-ffff-ffff-000000000001";

const HCaptchaBox = forwardRef(function HCaptchaBox({ onVerify, onExpire, onError }, ref) {
  const captchaRef = useRef(null);
  const [state, setState] = useState("idle");
  const siteKey = import.meta.env.VITE_HCAPTCHA_SITE_KEY || TEST_SITE_KEY;

  useImperativeHandle(ref, () => ({
    resetCaptcha() {
      captchaRef.current?.resetCaptcha();
      setState("idle");
    },
  }));

  return (
    <div className={`hcaptcha-box hcaptcha-box--${state}`}>
      <div>
        <strong>{state === "verified" ? "验证完成，可以领取" : state === "expired" ? "验证已过期，请重新验证" : state === "error" ? "验证码加载失败，请刷新页面或稍后再试" : "完成验证后即可领取 CDK"}</strong>
      </div>
      <HCaptcha
        ref={captchaRef}
        sitekey={siteKey}
        languageOverride="zh-CN"
        onVerify={(token) => {
          setState("verified");
          onVerify?.(token);
        }}
        onExpire={() => {
          setState("expired");
          onExpire?.();
        }}
        onError={(err) => {
          setState("error");
          onError?.(err);
        }}
      />
    </div>
  );
});

export default HCaptchaBox;
