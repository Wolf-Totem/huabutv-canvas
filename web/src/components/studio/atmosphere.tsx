import { useEffect, useRef, useState } from "react";

import { paintStudioFx } from "@/lib/studio-fx-canvas";
import { cn } from "@/lib/utils";

export type StudioAtmosphereSkin = {
    film?: string;
    poster?: string;
    fx?: string;
    hue?: number;
};

export function StudioAtmosphere({
    skin,
    mode,
    className,
}: {
    skin: StudioAtmosphereSkin | null | undefined;
    mode: "light" | "dark";
    className?: string;
}) {
    const layerRef = useRef<HTMLDivElement>(null);
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const pointerRef = useRef({ nx: 0.5, ny: 0.42 });
    const reducedMotion = typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const film = String(skin?.film || "").trim();
    const poster = String(skin?.poster || "").trim();
    const fx = String(skin?.fx || "rings");
    const [allowFilm, setAllowFilm] = useState(false);

    useEffect(() => {
        if (!film || reducedMotion) {
            setAllowFilm(false);
            return;
        }
        const start = () => setAllowFilm(true);
        const idle = "requestIdleCallback" in window ? window.requestIdleCallback(start, { timeout: 1800 }) : window.setTimeout(start, 800);
        return () => {
            if ("cancelIdleCallback" in window) window.cancelIdleCallback(idle as number);
            else window.clearTimeout(idle as number);
        };
    }, [film, reducedMotion]);

    useEffect(() => {
        const node = layerRef.current;
        if (!node) return;
        let targetX = 0.5;
        let targetY = 0.42;
        let currentX = 0.5;
        let currentY = 0.42;
        let frame = 0;
        const onMove = (event: PointerEvent) => {
            const box = node.getBoundingClientRect();
            targetX = (event.clientX - box.left) / Math.max(1, box.width);
            targetY = (event.clientY - box.top) / Math.max(1, box.height);
        };
        const tick = () => {
            currentX += (targetX - currentX) * 0.08;
            currentY += (targetY - currentY) * 0.08;
            pointerRef.current.nx = currentX;
            pointerRef.current.ny = currentY;
            node.style.setProperty("--mx", `${(currentX * 100).toFixed(2)}%`);
            node.style.setProperty("--my", `${(currentY * 100).toFixed(2)}%`);
            frame = requestAnimationFrame(tick);
        };
        window.addEventListener("pointermove", onMove, { passive: true });
        frame = requestAnimationFrame(tick);
        return () => {
            window.removeEventListener("pointermove", onMove);
            cancelAnimationFrame(frame);
        };
    }, []);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas || reducedMotion) return;
        const context = canvas.getContext("2d");
        if (!context) return;
        let frame = 0;
        let visible = document.visibilityState !== "hidden";
        const ratio = Math.min(1.4, window.devicePixelRatio || 1);
        const resize = () => {
            const box = canvas.getBoundingClientRect();
            canvas.width = Math.max(1, Math.floor(box.width * ratio));
            canvas.height = Math.max(1, Math.floor(box.height * ratio));
        };
        resize();
        const observer = new ResizeObserver(resize);
        observer.observe(canvas);
        const onVisibility = () => {
            visible = document.visibilityState !== "hidden";
        };
        document.addEventListener("visibilitychange", onVisibility);
        const started = performance.now();
        const tick = (now: number) => {
            if (visible) paintStudioFx(context, canvas.width, canvas.height, (now - started) / 1000, fx, mode, pointerRef.current.nx, pointerRef.current.ny);
            frame = requestAnimationFrame(tick);
        };
        frame = requestAnimationFrame(tick);
        return () => {
            cancelAnimationFrame(frame);
            observer.disconnect();
            document.removeEventListener("visibilitychange", onVisibility);
        };
    }, [fx, mode, reducedMotion]);

    if (!film && !poster) return null;

    return (
        <div ref={layerRef} className={cn("studio-fx", className)} data-fx={fx} data-mode={mode} aria-hidden="true">
            {reducedMotion || !film || !allowFilm ? (
                <div className="studio-fx-still" style={{ backgroundImage: `url(${poster || film})` }} />
            ) : (
                <video className="studio-fx-film" autoPlay loop muted playsInline preload="none" poster={poster || undefined}>
                    <source src={film} type="video/mp4" />
                </video>
            )}
            <div className="studio-fx-shade" />
            <div className="studio-fx-mesh" />
            {reducedMotion ? null : <canvas ref={canvasRef} className="studio-fx-canvas" />}
            <div className="studio-fx-orb" />
            <div className="studio-fx-scan" />
            <div className="studio-fx-grain" />
            <div className="studio-fx-vignette" />
        </div>
    );
}
