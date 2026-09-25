import { Link } from "react-router";

import type { PublicSiteSkin } from "@/services/api/streamer";

export default function StreamerHomePage({ skin }: { skin: PublicSiteSkin }) {
    const home = skin.home;
    const title = home?.hero?.title || skin.title || skin.displayName || "画布 TV";
    const subtitle = home?.hero?.subtitle || skin.tagline || (skin.displayName ? `通过 ${skin.displayName} 的专属邀请加入` : "登录后进入同一套无线画布");
    const cta = home?.cta?.label || "进入登录";
    const href = home?.cta?.href || "/login";
    const registerEnabled = skin.registrationEnabled !== false;
    return (
        <main className="min-h-dvh bg-[#07080c] text-white">
        <div className="mx-auto flex min-h-dvh max-w-4xl flex-col gap-10 px-6 py-16">
            {skin.logoUrl ? <img src={skin.logoUrl} alt="" className="h-12 w-auto object-contain" /> : null}
            <header>
                {skin.displayName ? <p className="text-xs uppercase tracking-[0.24em] text-white/45">{skin.displayName}</p> : null}
                <h1 className="mt-2 text-4xl font-semibold">{title}</h1>
                {subtitle ? <p className="mt-3 max-w-2xl text-base text-white/70">{subtitle}</p> : null}
            </header>
            {home?.hero?.imageUrl ? <img src={home.hero.imageUrl} alt="" className="max-h-80 w-full rounded-3xl object-cover" /> : null}
            {home?.features?.length ? (
                <section className="grid gap-4 sm:grid-cols-2">
                    {home.features.map((feature) => (
                        <article key={`${feature.title}-${feature.text}`} className="rounded-2xl border border-white/10 p-4">
                            <h2 className="text-lg font-semibold">{feature.title}</h2>
                            <p className="mt-2 text-sm text-white/70">{feature.text}</p>
                        </article>
                    ))}
                </section>
            ) : (
                <section className="grid gap-4 sm:grid-cols-2">
                    <article className="rounded-2xl border border-white/10 p-4">
                        <h2 className="text-lg font-semibold">专属邀请</h2>
                        <p className="mt-2 text-sm text-white/70">从这个域名注册的账号会终身归属当前代理，后续不会改绑。</p>
                    </article>
                    <article className="rounded-2xl border border-white/10 p-4">
                        <h2 className="text-lg font-semibold">同一套画布</h2>
                        <p className="mt-2 text-sm text-white/70">登录后进入官方工作台，创作、生成和项目能力与主站相同。</p>
                    </article>
                </section>
            )}
            <div className="flex flex-wrap gap-3">
                <Link to={href.startsWith("/") ? href : "/login"} className="inline-flex rounded-full bg-white px-5 py-2 text-sm font-semibold text-black">
                    {cta}
                </Link>
                {registerEnabled ? (
                    <Link to="/register" className="inline-flex rounded-full border border-white/20 px-5 py-2 text-sm font-semibold text-white">
                        立即注册
                    </Link>
                ) : null}
            </div>
            {home?.footer?.text ? <footer className="text-xs text-white/40">{home.footer.text}</footer> : null}
        </div>
        </main>
    );
}
