import { lazy, Suspense, type ComponentType, type ReactNode } from "react";
import { createBrowserRouter, Navigate, Outlet, useLocation } from "react-router";

import { RequireAuth } from "@/components/auth/require-auth";
import { RequireFeature } from "@/components/auth/require-feature";
import { FullScreenLoader, WorkspaceRouteLoader } from "@/components/ui/aceternity/full-screen-loader";
import { importWithChunkRecovery } from "@/lib/chunk-load";
import { loadAssetsPage, loadCanvasPage, loadCanvasProjectPage, loadCreatePage, loadProjectDetailPage, loadProjectsPage } from "@/lib/workspace-route-modules";
import { CanvasRefreshShell } from "@/pages/canvas/canvas-refresh-shell";
import { AuthScene } from "@/pages/auth/auth-scene";
import { AuthDialogHost } from "@/components/auth/auth-dialog";
import RouteErrorPage from "@/pages/route-error";

const lazyRoute = (loader: () => Promise<{ default: ComponentType<any> }>) => lazy(() => importWithChunkRecovery(loader));

const AdminPage = lazyRoute(() => import("@/pages/admin"));
const AnalyticsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.AnalyticsPage })));
const AnnouncementsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.AnnouncementsPage })));
const StorageResourcesPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.StorageResourcesPage })));
const CreditOperationsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.CreditOperationsPage })));
const AccessSettingsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.AccessSettingsPage })));
const EmailSettingsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.EmailSettingsPage })));
const SmsSettingsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.SmsSettingsPage })));
const FeatureAvailabilityPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.FeatureAvailabilityPage })));
const AgentLessonsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.AgentLessonsPage })));
const StreamersPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.StreamersPage })));
const PayoutsPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.PayoutsPage })));
const RolesPage = lazyRoute(() => import("@/pages/admin/admin-route-pages").then((module) => ({ default: module.RolesPage })));
const AgentConsolePage = lazyRoute(() => import("@/pages/agent/agent-console"));
const ChannelsPage = lazyRoute(() => import("@/pages/admin/channels/channels-page"));
const LogicalModelsPage = lazyRoute(() => import("@/pages/admin/logical-models/logical-models-page"));
const AdminPluginsPage = lazyRoute(() => import("@/pages/admin/plugins/plugins-page"));
const AdminPaymentsPage = lazyRoute(() => import("@/pages/admin/payments/payments-page"));
const LogsPage = lazyRoute(() => import("@/pages/admin/logs/logs-page"));
const RedemptionCodesPage = lazyRoute(() => import("@/pages/admin/redemption-codes/redemption-codes-page"));
const RuntimePolicySettingsPage = lazyRoute(() => import("@/pages/admin/settings/runtime-policy-settings-page"));
const AppearanceSettingsPage = lazyRoute(() => import("@/pages/admin/settings/appearance-settings-page"));
const DrawingEngineSettingsPage = lazyRoute(() => import("@/pages/admin/settings/drawing-engine-settings-page"));
const StorageSettingsPage = lazyRoute(() => import("@/pages/admin/settings/storage-settings-page"));
const ArkPrivateAssetsSettingsPage = lazyRoute(() => import("@/pages/admin/settings/ark-private-assets-settings-page"));
const ResponseInterceptionSettingsPage = lazyRoute(() => import("@/pages/admin/settings/response-interception-settings-page"));
const ThirdPartySettingsPage = lazyRoute(() => import("@/pages/admin/settings/libtv-settings-page"));
const SystemUpdatePage = lazyRoute(() => import("@/pages/admin/settings/system-update-page"));
const SystemPerformancePage = lazyRoute(() => import("@/pages/admin/settings/system-performance-page"));
const StoryboardPromptsPage = lazyRoute(() => import("@/pages/admin/storyboard-prompts/storyboard-prompts-page"));
const UsersPage = lazyRoute(() => import("@/pages/admin/users/users-page"));
const AssetsPage = lazy(() => importWithChunkRecovery(loadAssetsPage));
const ForgotPasswordPage = lazyRoute(() => import("@/pages/auth/forgot-password"));
const CanvasPage = lazy(() => importWithChunkRecovery(loadCanvasPage));
const CanvasProjectPage = lazy(() => importWithChunkRecovery(loadCanvasProjectPage));
const SharedCanvasPage = lazyRoute(() => import("@/pages/canvas/shared"));
const PlazaIndexPage = lazyRoute(() => import("@/pages/plaza/index"));
const PlazaProfilePage = lazyRoute(() => import("@/pages/plaza/profile"));
const PlazaWorkPage = lazyRoute(() => import("@/pages/plaza/work"));
const PlazaTourPage = lazyRoute(() => import("@/pages/plaza/tour"));
const PlazaApplicationsPage = lazyRoute(() => import("@/pages/admin/plaza-applications"));
const PlazaWorksAdminPage = lazyRoute(() => import("@/pages/admin/plaza-works"));
const PlazaApplicationPreviewPage = lazyRoute(() => import("@/pages/admin/plaza-application-preview"));
const CreatePage = lazy(() => importWithChunkRecovery(loadCreatePage));
const NotFound = lazyRoute(() => import("@/pages/not-found"));
const SkillsPage = lazyRoute(() => import("@/pages/skills"));
const PluginsPage = lazyRoute(() => import("@/pages/plugins"));
const EagleLibraryPage = lazyRoute(() => import("@/pages/plugins/eagle"));
const TasksPage = lazyRoute(() => import("@/pages/tasks"));
const ProjectsPage = lazy(() => importWithChunkRecovery(loadProjectsPage));
const ProjectDetailPage = lazy(() => importWithChunkRecovery(loadProjectDetailPage));
const SettingsPage = lazyRoute(() => import("@/pages/settings"));
const TestVoiceRecording = lazyRoute(() => import("@/pages/test-voice-recording"));
const UserLayout = lazyRoute(() => import("@/layouts/user-layout"));
const RootHome = lazyRoute(() => import("@/pages/public-home/root-home"));
const WelcomeHardLoad = lazyRoute(() => import("@/pages/public-home/root-home").then((module) => ({ default: module.WelcomeHardLoad })));

function deferred(element: ReactNode) {
    return <Suspense fallback={<WorkspaceRouteLoader />}>{element}</Suspense>;
}

function fullScreenDeferred(element: ReactNode) {
    return <Suspense fallback={<FullScreenLoader label="正在打开页面" detail="准备当前页面" />}>{element}</Suspense>;
}

function WorkspaceLayout() {
    const { pathname } = useLocation();
    const isCanvasProjectRoute = pathname.startsWith("/canvas/");
    const fallback = isCanvasProjectRoute ? <CanvasRefreshShell /> : <FullScreenLoader label="正在打开页面" detail="准备当前页面" />;
    return <Suspense fallback={fallback}><UserLayout><Outlet /></UserLayout></Suspense>;
}

function AuthenticatedWorkspaceLayout() {
    return <RequireAuth><WorkspaceLayout /></RequireAuth>;
}

/**
 * DEV 专用实验室路由。
 *
 * lazy(() => import(...)) 写在函数体内，而不是模块顶层常量：
 * 生产构建时 import.meta.env.DEV 被替换为 false，本函数随之不可达，
 * 摇树会连同其中的动态 import 一起删除，实验室代码不进入生产依赖图。
 * 若把 lazy 提到模块顶层，动态 import 会被静态分析成真实 chunk 并打进 dist。
 */
function devRoutes() {
    const FolderPreviewLab = lazyRoute(() => import("@/pages/dev/folder-preview-lab"));
    const DirectorReproLab = lazyRoute(() => import("@/pages/dev/director-repro-lab"));
    return [
        { path: "/dev/folders", element: fullScreenDeferred(<FolderPreviewLab />), errorElement: <RouteErrorPage /> },
        { path: "/dev/director-repro", element: fullScreenDeferred(<DirectorReproLab />), errorElement: <RouteErrorPage /> },
    ];
}

function AppFrame() {
    return (
        <>
            <AuthDialogHost />
            <Outlet />
        </>
    );
}

export const router = createBrowserRouter([
    {
        element: <AppFrame />,
        errorElement: <RouteErrorPage />,
        children: [
    {
        element: <AuthScene />,
        errorElement: <RouteErrorPage />,
        children: [
            { path: "/login", element: <Navigate to="/?auth=login" replace /> },
            { path: "/register", element: <Navigate to="/?auth=register" replace /> },
            { path: "/forgot-password", element: fullScreenDeferred(<ForgotPasswordPage />) },
        ],
    },
    { path: "/share/canvas/:token", element: fullScreenDeferred(<SharedCanvasPage />), errorElement: <RouteErrorPage /> },
    { path: "/plaza", element: fullScreenDeferred(<PlazaIndexPage />), errorElement: <RouteErrorPage /> },
    { path: "/plaza/:slug", element: fullScreenDeferred(<PlazaWorkPage />), errorElement: <RouteErrorPage /> },
    { path: "/plaza/:slug/tour", element: fullScreenDeferred(<PlazaTourPage />), errorElement: <RouteErrorPage /> },
    { path: "/u/:userId", element: fullScreenDeferred(<PlazaProfilePage />), errorElement: <RouteErrorPage /> },
    { path: "/admin/plaza/applications/:id/preview", element: <RequireAuth>{fullScreenDeferred(<PlazaApplicationPreviewPage />)}</RequireAuth>, errorElement: <RouteErrorPage /> },
    { path: "/", element: <Suspense fallback={null}><RootHome /></Suspense>, errorElement: <RouteErrorPage /> },
    { path: "/agent", element: <RequireAuth>{fullScreenDeferred(<AgentConsolePage />)}</RequireAuth>, errorElement: <RouteErrorPage /> },
    { path: "/welcome", element: fullScreenDeferred(<WelcomeHardLoad />), errorElement: <RouteErrorPage /> },
    ...(import.meta.env.DEV ? devRoutes() : []),
    {
        element: <WorkspaceLayout />,
        errorElement: <RouteErrorPage />,
        children: [
            { path: "/create", element: deferred(<CreatePage />) },
            {
                path: "/tasks",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="taskCenterEnabled">{deferred(<TasksPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            { path: "/assets", element: <RequireAuth>{deferred(<AssetsPage />)}</RequireAuth> },
            { path: "/skills", element: <RequireAuth>{deferred(<SkillsPage />)}</RequireAuth> },
            {
                path: "/plugins",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="pluginCenterEnabled">{deferred(<PluginsPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            {
                path: "/plugins/eagle",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="pluginCenterEnabled">{deferred(<EagleLibraryPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            {
                path: "/wallet",
                element: <RequireAuth>{null}</RequireAuth>,
            },
            { path: "/settings", element: <RequireAuth>{deferred(<SettingsPage />)}</RequireAuth> },
            { path: "/test-voice-recording", element: <RequireAuth>{deferred(<TestVoiceRecording />)}</RequireAuth> },
            {
                path: "/projects",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="shortDramaEnabled">{deferred(<ProjectsPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            {
                path: "/projects/:projectId",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="shortDramaEnabled">{deferred(<ProjectDetailPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            {
                path: "/projects/:projectId/:view",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="shortDramaEnabled">{deferred(<ProjectDetailPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            {
                path: "/projects/:projectId/chapters/:chapterId",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="shortDramaEnabled">{deferred(<ProjectDetailPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            {
                path: "/projects/:projectId/workflow/:unitId/:stage",
                element: (
                    <RequireAuth>
                        <RequireFeature feature="shortDramaEnabled">{deferred(<ProjectDetailPage />)}</RequireFeature>
                    </RequireAuth>
                ),
            },
            { path: "/canvas", element: <RequireAuth>{deferred(<CanvasPage />)}</RequireAuth> },
            { path: "/canvas/:id", element: <RequireAuth><CanvasProjectPage /></RequireAuth> },
            {
                path: "/admin",
                element: <RequireAuth>{deferred(<AdminPage />)}</RequireAuth>,
                children: [
                    { index: true, element: <AnalyticsPage /> },
                    { path: "users", element: <UsersPage /> },
                    { path: "roles", element: <RolesPage /> },
                    { path: "streamers", element: <StreamersPage /> },
                    { path: "payouts", element: <PayoutsPage /> },
                    { path: "channels", element: <ChannelsPage /> },
                    { path: "models", element: <RequireFeature feature="frontendModelsEnabled"><LogicalModelsPage /></RequireFeature> },
                    { path: "plugins", element: <AdminPluginsPage /> },
                    { path: "payments", element: <AdminPaymentsPage /> },
                    { path: "prompt-templates", element: <StoryboardPromptsPage /> },
                    { path: "storyboard-prompts", element: <Navigate to="/admin/prompt-templates" replace /> },
                    { path: "announcements", element: <AnnouncementsPage /> },
                    { path: "plaza/applications", element: <PlazaApplicationsPage /> },
                    { path: "plaza/works", element: <PlazaWorksAdminPage /> },
                    { path: "agent-lessons", element: <AgentLessonsPage /> },
                    { path: "resources", element: <StorageResourcesPage /> },
                    { path: "credit-operations", element: <CreditOperationsPage /> },
                    { path: "redemption-codes", element: <RedemptionCodesPage /> },
                    { path: "logs", element: <LogsPage /> },
                    { path: "settings", element: <Navigate to="runtime-policy" replace /> },
                    { path: "settings/appearance", element: <AppearanceSettingsPage /> },
                    { path: "settings/drawing-engine", element: <DrawingEngineSettingsPage /> },
                    { path: "settings/concurrency", element: <Navigate to="/admin/settings/runtime-policy" replace /> },
                    { path: "settings/runtime-policy", element: <RuntimePolicySettingsPage /> },
                    { path: "settings/features", element: <FeatureAvailabilityPage /> },
                    { path: "settings/access", element: <AccessSettingsPage /> },
                    { path: "settings/email", element: <EmailSettingsPage /> },
                    { path: "settings/sms", element: <SmsSettingsPage /> },
                    { path: "settings/storage", element: <StorageSettingsPage /> },
                    { path: "settings/ark-private-assets", element: <ArkPrivateAssetsSettingsPage /> },
                    { path: "settings/response-interception", element: <ResponseInterceptionSettingsPage /> },
                    { path: "settings/third-party", element: <ThirdPartySettingsPage /> },
                    { path: "settings/system-update", element: <SystemUpdatePage /> },
                    { path: "settings/system-performance", element: <SystemPerformancePage /> },
                    { path: "settings/libtv", element: <Navigate to="/admin/settings/third-party" replace /> },
                ],
            },
        ],
    },
    { path: "*", element: fullScreenDeferred(<NotFound />) },
        ],
    },
]);
