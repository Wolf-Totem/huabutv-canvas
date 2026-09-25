import { Button } from "antd";
import { Plus, Settings2, UserRoundCog } from "lucide-react";
import { lazy, useState } from "react";

import { useAdminContext } from "./admin-context";
import { AdminPageFrame } from "./components/admin-shell";
import { readAnnouncementPendingReview } from "./components/admin-announcement-safety";

const AnalyticsPanel = lazy(() => import("./components/analytics-panel"));
const AdminAnnouncementsPanel = lazy(() => import("./components/admin-announcements-panel"));
const CreditOperationsPanel = lazy(() => import("./components/credit-operations-panel"));
const AccessSettingsPanel = lazy(() => import("./components/access-settings-panel"));
const EmailSettingsPanel = lazy(() => import("./components/email-settings-panel"));
const SmsSettingsPanel = lazy(() => import("./components/sms-settings-panel"));
const FeatureAvailabilityPanel = lazy(() => import("./components/feature-availability-panel"));
const StorageResourcesPanel = lazy(() => import("./components/storage-resources-panel"));
const AgentLessonsPanel = lazy(() => import("./components/agent-lessons-panel"));
const StreamersPanel = lazy(() => import("./components/streamers-panel"));
const PayoutsPanel = lazy(() => import("./components/payouts-panel"));
const RolesPanel = lazy(() => import("./components/roles-panel"));

export function AnalyticsPage() {
    const { references } = useAdminContext();
    return (
        <AdminPageFrame title="数据概览" description="用户、任务、质量与成本健康度" scroll>
            <AnalyticsPanel users={references.users} channels={references.channels} />
        </AdminPageFrame>
    );
}

export function AnnouncementsPage() {
    const [publishOpen, setPublishOpen] = useState(false);
    const [publishBlocked, setPublishBlocked] = useState(() => Boolean(readAnnouncementPendingReview()));
    const [publishReturnFocus, setPublishReturnFocus] = useState<HTMLElement | null>(null);
    return (
        <AdminPageFrame
            title="系统公告"
            description="面向全体用户的通知发布与状态管理"
            actions={
                <Button
                    id="admin-announcement-publish-trigger"
                    type="primary"
                    disabled={publishBlocked}
                    title={publishBlocked ? "请先核对上一次结果不确定的发布请求" : undefined}
                    icon={<Plus className="size-4" />}
                    onClick={(event) => {
                        setPublishReturnFocus(event.currentTarget);
                        setPublishOpen(true);
                    }}
                >
                    发布公告
                </Button>
            }
        >
            <AdminAnnouncementsPanel publishOpen={publishOpen} publishBlocked={publishBlocked} publishReturnFocus={publishReturnFocus} onPublishOpenChange={setPublishOpen} onPublishBlockedChange={setPublishBlocked} />
        </AdminPageFrame>
    );
}

export function CreditOperationsPage() {
    const { references } = useAdminContext();
    const [activeOperation, setActiveOperation] = useState<"policy" | "adjustment" | null>(null);
    return (
        <AdminPageFrame
            title="积分运营"
            description="异常计费核对、积分策略与人工调账"
            actions={
                <>
                    <Button icon={<Settings2 className="size-4" />} onClick={() => setActiveOperation("policy")}>
                        积分策略
                    </Button>
                    <Button type="primary" icon={<UserRoundCog className="size-4" />} onClick={() => setActiveOperation("adjustment")}>
                        人工调账
                    </Button>
                </>
            }
        >
            <CreditOperationsPanel users={references.users} activeOperation={activeOperation} onOperationChange={setActiveOperation} />
        </AdminPageFrame>
    );
}

export function AccessSettingsPage() {
    return (
        <AdminPageFrame title="登录与注册" description="先控制账号创建，再配置第三方登录入口" scroll>
            <AccessSettingsPanel />
        </AdminPageFrame>
    );
}

export function EmailSettingsPage() {
    return (
        <AdminPageFrame title="邮件服务" description="先决定是否发送注册验证码，再配置 SMTP" scroll>
            <EmailSettingsPanel />
        </AdminPageFrame>
    );
}

export function SmsSettingsPage() {
    return (
        <AdminPageFrame title="短信服务" description="用阿里云短信发送注册验证码，签名和模板需先审核" scroll>
            <SmsSettingsPanel />
        </AdminPageFrame>
    );
}

export function FeatureAvailabilityPage() {
    return (
        <AdminPageFrame title="功能开放" description="按用户使用路径控制工作台、插件与模型能力" scroll>
            <FeatureAvailabilityPanel />
        </AdminPageFrame>
    );
}

export function StorageResourcesPage() {
    return (
        <AdminPageFrame title="存储资源" description="只读查看资源记录、容量分布与文件预览" scroll>
            <StorageResourcesPanel />
        </AdminPageFrame>
    );
}

export function StreamersPage() {
    return (
        <AdminPageFrame title="主播代理" description="创建主播子域、邀请码、分类返利与档案；专属首页与官网相同，只换背景视频。" scroll>
            <StreamersPanel />
        </AdminPageFrame>
    );
}

export function PayoutsPage() {
    return (
        <AdminPageFrame title="提现审核" description="同意只扣钱包记账，需要管理员线下转到代理支付宝。" scroll>
            <PayoutsPanel />
        </AdminPageFrame>
    );
}

export function RolesPage() {
    return (
        <AdminPageFrame title="角色管理" description="配置管理员、代理和普通用户分别能访问哪些页面" scroll>
            <RolesPanel />
        </AdminPageFrame>
    );
}

export function AgentLessonsPage() {
    return (
        <AdminPageFrame title="Agent 记忆" description="按用户查看个人记忆；批准仍由用户自己处理" scroll>
            <AgentLessonsPanel />
        </AdminPageFrame>
    );
}
