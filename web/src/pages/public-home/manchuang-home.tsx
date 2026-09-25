import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { BrandLogo } from "@/components/brand/brand-logo";
import { LANDING_NAV, mergeManchuangLanding, type LandingRailCard } from "@/lib/manchuang-landing";
import { ossProcessedImage } from "@/lib/oss-image";
import { DEFAULT_HOME_CTA_HREF, DEFAULT_HOME_CTA_LABEL } from "@/lib/home-navigation";
import { useAppearanceStore } from "@/stores/use-appearance-store";
import { useAuthDialogStore } from "@/stores/use-auth-dialog-store";
import { useUserStore } from "@/stores/use-user-store";

import "@fontsource/zhi-mang-xing";
import "@fontsource/ma-shan-zheng";
import "@fontsource/liu-jian-mao-cao";
import "./manchuang-home.css";

const SECTION_IDS = ["product", "solutions", "enterprise", "resources", "pricing"] as const;

export default function ManchuangHomePage({ heroVideoUrl, heroPosterUrl }: { heroVideoUrl?: string; heroPosterUrl?: string } = {}) {
    const navigate = useNavigate();
    const appearance = useAppearanceStore((state) => state.appearance);
    const openAuth = useAuthDialogStore((state) => state.openAuth);
    const user = useUserStore((state) => state.user);
    const brand = appearance.brandName || "漫创";
    const ctaLabel = appearance.homeCtaLabel?.trim() || DEFAULT_HOME_CTA_LABEL;
    const ctaHref = appearance.homeCtaHref?.trim() || DEFAULT_HOME_CTA_HREF;
    const openWorkspace = (path: string) => {
        const target = path.startsWith("/") ? path : "/create";
        if (document.documentElement.dataset.publicShell === "1") {
            window.location.assign(target);
            return;
        }
        navigate(target);
    };
    const landing = useMemo(
        () =>
            mergeManchuangLanding({
                ...appearance.landing,
                heroVideoUrl: heroVideoUrl || appearance.landingVideoUrl || appearance.landing?.heroVideoUrl,
                heroPosterUrl: heroPosterUrl || appearance.landing?.heroPosterUrl,
            }),
        [appearance.landing, appearance.landingVideoUrl, heroVideoUrl, heroPosterUrl],
    );
    const gateRef = useRef<HTMLDivElement>(null);
    const [activeCap, setActiveCap] = useState(0);
    const [solidNav, setSolidNav] = useState(false);
    const [activeSection, setActiveSection] = useState("top");

    useLayoutEffect(() => {
        document.documentElement.classList.add("mc-public-home");
        document.body.classList.add("mc-public-home");
        return () => {
            document.documentElement.classList.remove("mc-public-home");
            document.body.classList.remove("mc-public-home");
        };
    }, []);

    useEffect(() => {
        document.title = `${brand} · 绘无限 造未来`;
        const root = gateRef.current;
        if (!root) return;

        const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
        const timeline = root.querySelector<HTMLElement>(".mc-flow-timeline");
        const nodes = timeline ? Array.from(timeline.querySelectorAll<HTMLElement>(".mc-flow-node")) : [];
        let ticking = false;
        let lastFill = -1;

        const glowCards = (event: PointerEvent) => {
            const card = (event.target as HTMLElement | null)?.closest<HTMLElement>(
                ".mc-enterprise-card, .mc-resource-card, .mc-capability-button, .mc-pricing-card, .mc-flow-step, .mc-canvas-stage",
            );
            if (!card) return;
            const box = card.getBoundingClientRect();
            card.style.setProperty("--mx", `${((event.clientX - box.left) / Math.max(box.width, 1)) * 100}%`);
            card.style.setProperty("--my", `${((event.clientY - box.top) / Math.max(box.height, 1)) * 100}%`);
        };

        const sync = () => {
            ticking = false;
            const top = root.scrollTop;
            setSolidNav(top > 24);

            const view = root.clientHeight || window.innerHeight;
            let current: string = "top";
            for (const id of SECTION_IDS) {
                const section = root.querySelector<HTMLElement>(`#${id}`);
                if (!section) continue;
                if (section.getBoundingClientRect().top - root.getBoundingClientRect().top < view * 0.42) current = id;
            }
            setActiveSection(current);

            if (!timeline || !nodes.length) return;
            if (reduced) {
                timeline.style.setProperty("--flow-fill", "1");
                nodes.forEach((node) => {
                    node.classList.add("is-lit");
                    node.closest(".mc-flow-step")?.classList.add("is-active");
                });
                return;
            }
            const box = timeline.getBoundingClientRect();
            const mid = (root.clientHeight || window.innerHeight || 1) * 0.5;
            const height = box.height || 1;
            let fill = (mid - box.top) / height;
            fill = fill < 0 ? 0 : fill > 1 ? 1 : fill;
            if (Math.abs(fill - lastFill) > 0.0012) {
                lastFill = fill;
                timeline.style.setProperty("--flow-fill", fill.toFixed(4));
            }
            const head = box.top + fill * height;
            nodes.forEach((node) => {
                const nodeBox = node.getBoundingClientRect();
                const lit = head >= nodeBox.top + nodeBox.height / 2 - 6;
                node.classList.toggle("is-lit", lit);
                node.closest(".mc-flow-step")?.classList.toggle("is-active", lit);
            });
        };

        const onScroll = () => {
            if (ticking) return;
            ticking = true;
            window.requestAnimationFrame(sync);
        };

        root.addEventListener("scroll", onScroll, { passive: true });
        root.addEventListener("pointermove", glowCards, { passive: true });
        window.addEventListener("resize", onScroll, { passive: true });
        timeline?.querySelectorAll<HTMLImageElement>(".mc-flow-media img").forEach((image) => {
            if (!image.complete) image.addEventListener("load", onScroll, { once: true });
        });
        sync();
        return () => {
            root.removeEventListener("scroll", onScroll);
            root.removeEventListener("pointermove", glowCards);
            window.removeEventListener("resize", onScroll);
            timeline?.style.removeProperty("--flow-fill");
            nodes.forEach((node) => {
                node.classList.remove("is-lit");
                node.closest(".mc-flow-step")?.classList.remove("is-active");
            });
        };
    }, [brand, landing.workflowSteps.length]);

    const go = (href: string) => {
        if (href.startsWith("#")) {
            const id = href.slice(1);
            const root = gateRef.current;
            const target = root?.querySelector<HTMLElement>(`#${id}`) ?? document.getElementById(id);
            if (root && target) {
                const next = target.getBoundingClientRect().top - root.getBoundingClientRect().top + root.scrollTop - 88;
                root.scrollTo({ top: Math.max(0, next), behavior: "smooth" });
                return;
            }
            target?.scrollIntoView({ behavior: "smooth", block: "start" });
            return;
        }
        openWorkspace(href);
    };

    return (
        <div className="mc-gate" ref={gateRef}>
            <header className={solidNav ? "mc-nav is-solid" : "mc-nav"}>
                <a className="mc-brand" href="#top" onClick={(event) => { event.preventDefault(); go("#top"); }}>
                    <BrandLogo theme="dark" className="mc-brand-mark" alt="" fallback={<img src="/manchuang/logo.png" alt="" className="mc-brand-mark" />} />
                    <span className="mc-brand-text">
                        <strong>{brand}</strong>
                        <em>CANVAS</em>
                    </span>
                </a>
                <nav className="mc-nav-links" aria-label="官网导航">
                    <button type="button" className="mc-btn mc-btn-primary mc-nav-start" onClick={() => openWorkspace(ctaHref.startsWith("/") ? ctaHref : "/create")}>{ctaLabel}</button>
                    <a href="/create#plaza" onClick={(event) => { event.preventDefault(); openWorkspace("/create#plaza"); }}>作品</a>
                    {LANDING_NAV.map((item) => (
                        <a
                            key={item.id}
                            href={`#${item.id}`}
                            className={activeSection === item.id ? "is-current" : undefined}
                            onClick={(event) => { event.preventDefault(); go(`#${item.id}`); }}
                        >
                            {item.label}
                        </a>
                    ))}
                </nav>
                <div className="mc-nav-actions">
                    {user ? (
                        <button type="button" className="mc-btn mc-btn-primary" onClick={() => openWorkspace("/create")}>进入工作台</button>
                    ) : (
                        <>
                            <button type="button" className="mc-btn mc-btn-ghost" onClick={(event) => { event.preventDefault(); event.stopPropagation(); openAuth({ tab: "login" }); }}>登录</button>
                            <button type="button" className="mc-btn mc-btn-primary" onClick={(event) => { event.preventDefault(); event.stopPropagation(); openAuth({ tab: "register" }); }}>注册</button>
                        </>
                    )}
                </div>
            </header>

            <section className="mc-hero" id="top">
                <div className="mc-hero-media" aria-hidden="true">
                    <video key={landing.heroVideoUrl} className="mc-hero-video" autoPlay muted loop playsInline preload="metadata" poster={landing.heroPosterUrl || undefined}>
                        <source src={landing.heroVideoUrl} type="video/mp4" />
                    </video>
                    <span className="mc-hero-media-scrim" />
                </div>
                <div className="mc-hero-inner">
                    <div className="mc-hero-copy">
                        <p className="mc-kicker"><span />{landing.heroKicker}</p>
                        <h1 className="mc-hero-title">
                            <span className="mc-sr-only">绘无限 造未来</span>
                            <img src="/manchuang/hero-title.webp" alt="" aria-hidden="true" />
                            <span className="mc-hero-title-ink" aria-hidden="true" />
                        </h1>
                        <p className="mc-hero-lead">{landing.heroLead}</p>
                    </div>
                </div>
                <div className="mc-hero-cta-pin">
                    <button type="button" className="mc-btn mc-btn-primary mc-hero-cta" onClick={() => openWorkspace(ctaHref.startsWith("/") ? ctaHref : "/create")}>{ctaLabel}</button>
                </div>
                <HeroRail items={landing.rail} onOpen={(id) => openWorkspace(`/plaza/${encodeURIComponent(id)}`)} />
            </section>

            <section className="mc-section mc-canvas" id="product">
                <div className="mc-canvas-layout">
                    <div className="mc-canvas-copy">
                        <h2 className="mc-canvas-title">
                            <span className="mc-sr-only">{landing.productTitleTop} 连接{landing.productTitleAccent}</span>
                            <span className="mc-canvas-title-text" aria-hidden="true">
                                <span className="mc-canvas-title-line mc-canvas-title-line-top">{landing.productTitleTop}</span>
                                <span className="mc-canvas-title-line mc-canvas-title-line-bottom">
                                    <span className="mc-canvas-title-warm">连接</span>
                                    <span className="mc-canvas-title-purple">{landing.productTitleAccent}</span>
                                </span>
                            </span>
                        </h2>
                        <p>{landing.productLead}</p>
                        <a href="#enterprise" onClick={(event) => { event.preventDefault(); go("#enterprise"); }}>{landing.productMoreLabel} <span aria-hidden>→</span></a>
                    </div>
                    <div className="mc-canvas-stage">
                        <img src={landing.productStageUrl} alt={`${brand}无限画布产品界面`} />
                    </div>
                    <div className="mc-capability-list">
                        {landing.capabilities.map((item, index) => (
                            <button key={item.title} type="button" className={index === activeCap ? "mc-capability-button is-active" : "mc-capability-button"} onClick={() => setActiveCap(index)}>
                                <span className="mc-capability-icon" aria-hidden="true">{["✦", "◎", "⌁", "◇"][index] || "✦"}</span>
                                <span>
                                    <strong>{item.title}</strong>
                                    <em>{item.text}</em>
                                </span>
                            </button>
                        ))}
                    </div>
                </div>
            </section>

            <section className="mc-section mc-flow" id="solutions">
                <div className="mc-section-head">
                    <p className="mc-section-kicker">{landing.workflowKicker}</p>
                    <h2>{landing.workflowTitle}</h2>
                    <p>{landing.workflowLead}</p>
                </div>
                <div className="mc-flow-timeline">
                    <span className="mc-flow-spine" aria-hidden="true" />
                    {landing.workflowSteps.map((step, index) => (
                        <article key={step.step} className={index % 2 === 0 ? "mc-flow-step mc-flow-step-left" : "mc-flow-step mc-flow-step-right"}>
                            <div className="mc-flow-text">
                                <span className="mc-flow-phase"><span className="mc-flow-phase-step">{step.step}</span>{step.phase}</span>
                                <h3>{step.title}</h3>
                                <p className="mc-flow-tagline">{step.tagline}</p>
                                <p className="mc-flow-desc">{step.description}</p>
                                <ul className="mc-flow-tags">{step.tags.map((tag) => <li key={tag}>{tag}</li>)}</ul>
                            </div>
                            <div className="mc-flow-node" aria-hidden="true"><span>{step.step}</span></div>
                            <div className="mc-flow-media"><img src={step.imageUrl} alt="" /></div>
                        </article>
                    ))}
                </div>
            </section>

            <section className="mc-section mc-enterprise" id="enterprise">
                <div className="mc-section-head">
                    <p className="mc-section-kicker">Enterprise</p>
                    <h2>{landing.enterpriseTitle}</h2>
                    <p>{landing.enterpriseLead}</p>
                </div>
                <div className="mc-enterprise-grid">
                    {landing.enterpriseCards.map((card) => (
                        <article key={card.title} className="mc-enterprise-card">
                            <h3>{card.title}</h3>
                            <p>{card.text}</p>
                        </article>
                    ))}
                </div>
            </section>

            <section className="mc-section mc-resources" id="resources">
                <div className="mc-section-head">
                    <p className="mc-section-kicker">Resources</p>
                    <h2>{landing.resourcesTitle}</h2>
                    <p>{landing.resourcesLead}</p>
                </div>
                <div className="mc-resource-grid">
                    {landing.resourceCards.map((card) => (
                        <article key={card.title} className="mc-resource-card">
                            <h3>{card.title}</h3>
                            <p>{card.text}</p>
                        </article>
                    ))}
                </div>
            </section>

            <section className="mc-section mc-pricing" id="pricing">
                <div className="mc-section-head">
                    <p className="mc-section-kicker">Pricing</p>
                    <h2>{landing.pricingTitle}</h2>
                    <p>{landing.pricingLead}</p>
                </div>
                <div className="mc-pricing-grid">
                    {landing.pricingTiers.map((tier) => (
                        <article key={tier.id} className={tier.featured ? "mc-pricing-card is-featured" : "mc-pricing-card"}>
                            {tier.badge ? <span className="mc-pricing-badge">{tier.badge}</span> : null}
                            <h3>{tier.name}</h3>
                            <p className="mc-pricing-tagline">{tier.tagline}</p>
                            <p className="mc-pricing-price"><strong>{tier.price}</strong><span>{tier.unit}</span></p>
                            <p className="mc-pricing-note">{tier.note}</p>
                            <ul>{tier.features.map((feature) => <li key={feature}>{feature}</li>)}</ul>
                            <button type="button" className="mc-btn mc-btn-primary" onClick={() => openAuth({ tab: "register" })}>{tier.cta}</button>
                        </article>
                    ))}
                </div>
            </section>
        </div>
    );
}

function neighborLift(index: number, hover: number) {
    if (hover < 0) return 0;
    const dist = Math.abs(index - hover);
    if (dist === 0) return 1;
    if (dist === 1) return 0.62;
    if (dist === 2) return 0.26;
    return 0;
}

function HeroRail({ items, onOpen }: { items: LandingRailCard[]; onOpen?: (id: string) => void }) {
    const track = useRef<HTMLDivElement>(null);
    const looped = useMemo(() => [0, 1, 2].flatMap((copy) => items.map((item) => ({ ...item, key: `${copy}-${item.id}` }))), [items]);

    useEffect(() => {
        const root = track.current;
        if (!root) return;
        const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
        const cards = Array.from(root.querySelectorAll<HTMLElement>(".mc-hero-card"));
        if (!cards.length) return;
        let width = cards[0].offsetWidth || 196;
        let step = width + 18;
        let total = cards.length * step;
        const measure = () => {
            width = cards[0].offsetWidth || 196;
            step = width + 18;
            total = cards.length * step;
        };
        const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(measure);
        observer?.observe(root);
        let offset = 0;
        let hover = -1;
        const lifts = Array(cards.length).fill(0);
        const enter: Array<() => void> = [];
        const leave: Array<() => void> = [];
        cards.forEach((card, index) => {
            const onEnter = () => {
                hover = index;
                const url = card.getAttribute("data-preview") || "";
                if (!url) return;
                let video = card.querySelector("video");
                if (!video) {
                    video = document.createElement("video");
                    video.className = "mc-hero-preview";
                    video.muted = true;
                    video.loop = true;
                    video.playsInline = true;
                    video.setAttribute("playsinline", "");
                    video.preload = "auto";
                    video.src = url;
                    card.appendChild(video);
                }
                void video.play().catch(() => undefined);
            };
            const onLeave = () => {
                if (hover === index) hover = -1;
                const video = card.querySelector("video");
                if (video) {
                    video.pause();
                    video.removeAttribute("src");
                    video.load();
                    video.remove();
                }
            };
            card.addEventListener("pointerenter", onEnter);
            card.addEventListener("pointerleave", onLeave);
            enter.push(onEnter);
            leave.push(onLeave);
        });
        let frame = 0;
        const tick = () => {
            const view = root.clientWidth || window.innerWidth;
            const center = view / 2;
            if (!reduced && hover < 0) {
                offset += 0.45;
                if (offset >= total) offset -= total;
            }
            cards.forEach((card, index) => {
                let x = (index * step + offset) % total;
                if (x < 0) x += total;
                x -= step;
                const mid = x + width / 2;
                const p = Math.max(-1.5, Math.min(1.5, (mid - center) / (view * 0.6)));
                const arc = 132 * (1 - p * p);
                const target = neighborLift(index, hover);
                lifts[index] += (target - lifts[index]) * 0.18;
                const lift = lifts[index];
                const y = -arc - 52 * lift;
                const scale = 1 + 0.16 * lift;
                const rot = p * 11 * (1 - lift);
                card.style.transform = `translate(${x.toFixed(1)}px, ${y.toFixed(1)}px) rotate(${rot.toFixed(2)}deg) scale(${scale.toFixed(3)})`;
                card.style.zIndex = String(Math.round(40 + lift * 60 - Math.abs(p) * 18));
                const dist = hover < 0 ? 99 : Math.abs(index - hover);
                card.classList.toggle("is-up", dist === 0 && hover >= 0);
                card.classList.toggle("is-near", dist === 1 && hover >= 0);
                card.classList.toggle("is-soft", dist === 2 && hover >= 0);
            });
            frame = window.requestAnimationFrame(tick);
        };
        tick();
        return () => {
            window.cancelAnimationFrame(frame);
            observer?.disconnect();
            cards.forEach((card, index) => {
                card.removeEventListener("pointerenter", enter[index]);
                card.removeEventListener("pointerleave", leave[index]);
            });
        };
    }, [looped]);

    return (
        <div className="mc-hero-rail" aria-hidden="true">
            <div className="mc-hero-rail-track" ref={track}>
                {looped.map((item) => (
                    <div key={item.key} className="mc-hero-card" data-preview={item.previewUrl || undefined} title={item.label || undefined} role="button" tabIndex={0} onClick={() => onOpen?.(item.id)}>
                        <img src={ossProcessedImage(item.imageUrl, 480)} alt="" loading="lazy" draggable={false} />
                    </div>
                ))}
            </div>
        </div>
    );
}
