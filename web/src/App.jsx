import { Routes, Route, Navigate } from "react-router-dom";
import AdminLayout from "./layouts/AdminLayout";
import ClaimPage from "./pages/ClaimPage";
import LoginPage from "./pages/LoginPage";
import LoginCallbackPage from "./pages/LoginCallbackPage";

import DashboardPage from "./pages/admin/DashboardPage";
import ProjectListPage from "./pages/admin/ProjectListPage";
import ProjectCreatePage from "./pages/admin/ProjectCreatePage";
import ProjectDetailPage from "./pages/admin/ProjectDetailPage";
import CampaignListPage from "./pages/admin/CampaignListPage";
import NodeListPage from "./pages/admin/NodeListPage";
import NodeDetailPage from "./pages/admin/NodeDetailPage";
import SettingsPage from "./pages/admin/SettingsPage";
import { AnalyticsPage, CaptchaConfigPage, CDKInventoryPage, ClaimRecordsPage, LogsPage, RiskCenterPage } from "./pages/admin/PlaceholderAdminPage";

export default function App() {
  return (
    <Routes>
      {/* 登录页 */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/login/callback" element={<LoginCallbackPage />} />


      {/* 管理后台 */}
      <Route path="/admin" element={<AdminLayout />}>
        <Route index element={<Navigate to="dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="campaigns" element={<CampaignListPage />} />
        <Route path="campaigns/create" element={<CampaignListPage />} />
        <Route path="campaigns/rules" element={<CampaignListPage />} />
        <Route path="campaigns/archive" element={<CampaignListPage />} />
        <Route path="cdks" element={<CDKInventoryPage />} />
        <Route path="cdks/import" element={<CDKInventoryPage />} />
        <Route path="cdks/status" element={<CDKInventoryPage />} />
        <Route path="cdks/export" element={<CDKInventoryPage />} />
        <Route path="nodes" element={<NodeListPage />} />
        <Route path="nodes/create" element={<NodeListPage />} />
        <Route path="nodes/links" element={<NodeListPage />} />
        <Route path="nodes/performance" element={<NodeListPage />} />
        <Route path="nodes/:id" element={<NodeDetailPage />} />
        <Route path="claims" element={<ClaimRecordsPage />} />
        <Route path="claims/failures" element={<ClaimRecordsPage />} />
        <Route path="claims/devices" element={<ClaimRecordsPage />} />
        <Route path="claims/exports" element={<ClaimRecordsPage />} />
        <Route path="risk" element={<RiskCenterPage />} />
        <Route path="risk/rules" element={<RiskCenterPage />} />
        <Route path="risk/blacklist" element={<RiskCenterPage />} />
        <Route path="risk/frequency" element={<RiskCenterPage />} />
        <Route path="captcha" element={<CaptchaConfigPage />} />
        <Route path="captcha/hcaptcha" element={<CaptchaConfigPage />} />
        <Route path="captcha/nodes" element={<CaptchaConfigPage />} />
        <Route path="captcha/testing" element={<CaptchaConfigPage />} />
        <Route path="analytics" element={<AnalyticsPage />} />
        <Route path="analytics/traffic" element={<AnalyticsPage />} />
        <Route path="analytics/claims" element={<AnalyticsPage />} />
        <Route path="analytics/ranking" element={<AnalyticsPage />} />
        <Route path="logs" element={<LogsPage />} />
        <Route path="logs/operations" element={<LogsPage />} />
        <Route path="logs/claims" element={<LogsPage />} />
        <Route path="logs/errors" element={<LogsPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="settings/brand" element={<SettingsPage />} />
        <Route path="settings/storage" element={<SettingsPage />} />
        <Route path="settings/admins" element={<SettingsPage />} />
        <Route path="projects" element={<ProjectListPage />} />
        <Route path="projects/create" element={<ProjectCreatePage />} />
        <Route path="projects/bindings" element={<ProjectListPage />} />
        <Route path="projects/:id" element={<ProjectDetailPage />} />
      </Route>

      {/* 独立领取页 — 普通用户唯一入口 */}
      <Route path="/claim/:projectCode" element={<ClaimPage />} />

      {/* 默认重定向到管理后台 */}
      <Route path="*" element={<Navigate to="/admin" replace />} />
    </Routes>
  );
}
