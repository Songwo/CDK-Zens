const FINGERPRINT_KEY = "cdk-airdrop:fingerprint";
const RECEIPT_KEY = "cdk-airdrop:last-receipt";
const ACCOUNT_KEY = "cdk-airdrop:last-account";

export function getFingerprint() {
  const existing = localStorage.getItem(FINGERPRINT_KEY);
  if (existing) {
    return existing;
  }

  const generated =
    globalThis.crypto?.randomUUID?.() ||
    `fp-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;

  localStorage.setItem(FINGERPRINT_KEY, generated);
  return generated;
}

export function loadClaimReceipt() {
  const raw = localStorage.getItem(RECEIPT_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw);
  } catch {
    localStorage.removeItem(RECEIPT_KEY);
    return null;
  }
}

export function saveClaimReceipt(receipt) {
  localStorage.setItem(RECEIPT_KEY, JSON.stringify(receipt));
}

export function loadSavedAccount() {
  return localStorage.getItem(ACCOUNT_KEY) || "";
}

export function saveAccount(account) {
  const normalized = account.trim();
  if (!normalized) {
    localStorage.removeItem(ACCOUNT_KEY);
    return;
  }
  localStorage.setItem(ACCOUNT_KEY, normalized);
}
