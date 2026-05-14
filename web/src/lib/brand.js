export const DEFAULT_BRAND = {
  systemName: "Zens-CDK",
  brandName: "Zens-CDK",
  brandEnglishName: "Zens CDK Airdrop Hub",
  logoText: "ZC",
};

export const SETTINGS_UPDATED_EVENT = "zens:settings-updated";

export function normalizeBrand(settings) {
  return {
    ...DEFAULT_BRAND,
    ...(settings || {}),
  };
}

export function setAppTitle(settings) {
  document.title = normalizeBrand(settings).systemName || DEFAULT_BRAND.systemName;
}

export function notifySettingsUpdated(settings) {
  window.dispatchEvent(new CustomEvent(SETTINGS_UPDATED_EVENT, { detail: settings }));
}
