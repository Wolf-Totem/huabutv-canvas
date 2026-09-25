import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";

import { BrandLogo } from "@/components/brand/brand-logo";
import { ossProcessedImage } from "@/lib/oss-image";
import { DEFAULT_HOME_CTA_HREF, DEFAULT_HOME_CTA_LABEL, DEFAULT_HOME_NAV_ITEMS, validHomeHref } from "@/lib/home-navigation";
import featuredTemplates from "@/lib/plaza-featured.json";
import { getPlazaCategories, getPlazaSettings, listPlazaWorks, type PlazaCategory, type PlazaWork } from "@/services/api/plaza";
import { useAppearanceStore } from "@/stores/use-appearance-store";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import { useUserStore } from "@/stores/use-user-store";
import "./portal-home.css";

type FeaturedCard = { id: string; slug: string; title: string; subtitle: string; coverUrl: string; href: string; author?: string };

const featuredCards: FeaturedCard[] = (featuredTemplates as Array<{ projectUuid: string; name: string; description: string; coverUrl: string }>).map((item) => ({
    id: item.projectUuid,
    slug: item.projectUuid,
    title: item.name,
    subtitle: item.description,
    coverUrl: item.coverUrl,
    href: "/create",
    author: "精选",
}));

export default function PortalHomePage() {
    const appearance = useAppearanceStore((state) => state.appearance);
    const user = useUserStore((state) => state.user);
    const openAuth = useAuthDialogStore((state) => state.openAuth);
    const navItems = (appearance.homeNavItems?.length ? appearance.homeNavItems : DEFAULT_HOME_NAV_ITEMS).filter((item) => validHomeHref(item.href));
    const ctaLabel = appearance.homeCtaLabel?.trim() || DEFAULT_HOME_CTA_LABEL;
    const ctaHref = appearance.homeCtaHref?.trim() || DEFAULT_HOME_CTA_HREF;
    const [category, setCategory] = useState("all");
    const [categories, setCategories] = useState<PlazaCategory[]>([]);
    const [works, setWorks] = useState<PlazaWork[]>([]);
    const [plazaOn, setPlazaOn] = useState(false);

    useEffect(() => {
        let active = true;
        getPlazaSettings()
            .then(({ settings }) => {
                if (!active) return;
                setPlazaOn(settings.enabled);
                if (!settings.enabled) return Promise.resolve();
                return Promise.all([getPlazaCategories(), listPlazaWorks({ category, sort: "hot", page: 1, pageSize: 32 })]).then(([cats, list]) => {
                    if (!active) return;
                    setCategories(cats.categories.filter((item) => item.slug !== "all"));
                    setWorks(list.works);
                });
            })
            .catch(() => undefined);
        return () => {
            active = false;
        };
    }, [category]);

    const cards = useMemo<FeaturedCard[]>(() => {
        if (works.length) {
            return works.map((work) => ({
                id: work.id,
                slug: work.slug,
                title: work.title,
                subtitle: work.subtitle || work.category?.name || "",
                coverUrl: work.coverUrl || "",
                href: `/plaza/${work.slug}`,
                author: work.author?.displayName,
            }));
        }
        return featuredCards;
    }, [works]);

    const tabs = [{ slug: "all", name: "全部" }, ...categories.map((item) => ({ slug: item.slug, name: item.name }))];

    return (
        <div className="portal-home">
            <header className="portal-header">
                <Link to="/" className="portal-brand">
                    <BrandLogo className="portal-brand-mark" alt="" fallback={<img src="/logo.png" alt="" className="portal-brand-mark" />} />
                    <strong>{appearance.brandName || "画布TV"}</strong>
                </Link>
                <nav className="portal-nav" aria-label="首页导航">
                    {navItems.map((item) => (
                        <a key={`${item.label}-${item.href}`} href={item.href} target={item.openInNewTab ? "_blank" : undefined} rel={item.openInNewTab ? "noreferrer" : undefined}>
                            {item.label}
                        </a>
                    ))}
                </nav>
                <div className="portal-header-actions">
                    {user ? (
                        <Link className="portal-ghost" to="/create">{user.displayName || user.username}</Link>
                    ) : (
                        <>
                            <button type="button" className="portal-ghost" onClick={() => openAuth({ tab: "login" })}>登录</button>
                            <button type="button" className="portal-pill is-primary" onClick={() => openAuth({ tab: "register" })}>注册</button>
                        </>
                    )}
                </div>
            </header>
            <main className="portal-main">
                <section className="portal-hero">
                    <Link className="portal-start" to={ctaHref.startsWith("/") ? ctaHref : "/create"}>{ctaLabel}</Link>
                    <p>从一句话开始做漫剧与短片。未登录也能进创作台，生成时再登录。</p>
                </section>
                <div className="portal-section-title">
                    <span>作品广场</span>
                    <Link to="/plaza" style={{ color: "rgba(244,241,234,.5)", fontSize: 13 }}>查看全部</Link>
                </div>
                <div className="portal-tabs">
                    {tabs.map((item) => (
                        <button key={item.slug} type="button" className={category === item.slug ? "is-active" : undefined} onClick={() => setCategory(item.slug)}>
                            {item.name}
                        </button>
                    ))}
                </div>
                {cards.length ? (
                    <div className="portal-grid">
                        {cards.map((card) => (
                            <Link key={card.id} className="portal-card" to={card.href}>
                                <div className="portal-card-cover">
                                    {card.coverUrl ? <img src={ossProcessedImage(card.coverUrl, 720)} alt="" loading="lazy" /> : null}
                                </div>
                                <div className="portal-card-title">{card.title}</div>
                                <div className="portal-card-meta">{card.author || card.subtitle}</div>
                            </Link>
                        ))}
                    </div>
                ) : (
                    <div className="portal-empty">{plazaOn ? "还没有上架作品。" : "广场即将开放，先从开始创作进入工作台。"}</div>
                )}
            </main>
        </div>
    );
}
