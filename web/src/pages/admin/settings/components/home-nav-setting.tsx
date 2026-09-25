import { Button, Input } from "antd";
import { Plus, Trash2 } from "lucide-react";
import { Switch } from "@/pages/admin/ui/controls";

import type { HomeNavItem } from "@/lib/home-navigation";

export function HomeNavSetting({
    items,
    ctaLabel,
    ctaHref,
    disabled,
    onChangeItems,
    onChangeCtaLabel,
    onChangeCtaHref,
}: {
    items: HomeNavItem[];
    ctaLabel: string;
    ctaHref: string;
    disabled?: boolean;
    onChangeItems: (items: HomeNavItem[]) => void;
    onChangeCtaLabel: (value: string) => void;
    onChangeCtaHref: (value: string) => void;
}) {
    const update = (index: number, patch: Partial<HomeNavItem>) => {
        onChangeItems(items.map((item, current) => (current === index ? { ...item, ...patch } : item)));
    };
    return (
        <div className="space-y-6">
            <div className="space-y-3">
                <div className="flex items-center justify-between gap-3">
                    <div>
                        <strong className="text-sm">导航菜单</strong>
                        <p className="mt-1 text-xs text-foreground/55">显示在欢迎页顶栏。链接可以是页内锚点（#workbench）、站内路径（/create）或完整网址。</p>
                    </div>
                    <Button size="small" icon={<Plus className="size-3.5" />} disabled={disabled || items.length >= 8} onClick={() => onChangeItems([...items, { label: "", href: "/", openInNewTab: false }])}>
                        添加菜单
                    </Button>
                </div>
                {items.length ? (
                    <div className="grid gap-3">
                        {items.map((item, index) => (
                            <div key={index} className="grid gap-2 rounded-lg border border-border p-3 sm:grid-cols-[1fr_1.4fr_auto_auto]">
                                <Input value={item.label} maxLength={24} placeholder="菜单名称" disabled={disabled} onChange={(event) => update(index, { label: event.target.value })} />
                                <Input value={item.href} maxLength={300} placeholder="#workbench 或 /create 或 https://" disabled={disabled} onChange={(event) => update(index, { href: event.target.value })} />
                                <label className="flex items-center gap-2 text-xs text-foreground/70">
                                    <Switch checked={Boolean(item.openInNewTab)} disabled={disabled} onChange={(openInNewTab) => update(index, { openInNewTab })} />
                                    新窗口
                                </label>
                                <Button type="text" danger disabled={disabled} icon={<Trash2 className="size-3.5" />} aria-label={`删除菜单 ${item.label || index + 1}`} onClick={() => onChangeItems(items.filter((_, current) => current !== index))} />
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="text-xs text-foreground/45">还没有菜单项。可以只保留右侧按钮。</p>
                )}
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
                <label className="space-y-1 text-sm">
                    <span>首页按钮文案</span>
                    <Input value={ctaLabel} maxLength={24} disabled={disabled} placeholder="开始创作" onChange={(event) => onChangeCtaLabel(event.target.value)} />
                </label>
                <label className="space-y-1 text-sm">
                    <span>首页按钮链接</span>
                    <Input value={ctaHref} maxLength={300} disabled={disabled} placeholder="/create" onChange={(event) => onChangeCtaHref(event.target.value)} />
                </label>
            </div>
        </div>
    );
}
