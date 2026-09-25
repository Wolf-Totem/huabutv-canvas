import type { ImgHTMLAttributes, ReactNode } from "react";

import { mediaThumbUrl } from "@/lib/media-thumb";

type MediaThumbProps = {
    kind?: string;
    coverUrl?: string;
    originalUrl?: string;
    width?: number;
    alt?: string;
    className?: string;
    fallback?: ReactNode;
} & Omit<ImgHTMLAttributes<HTMLImageElement>, "src" | "alt" | "width">;

export function MediaThumb({ kind, coverUrl, originalUrl, width = 480, alt = "", className, fallback = null, ...props }: MediaThumbProps) {
    const src = mediaThumbUrl({ kind, coverUrl, originalUrl, width });
    if (!src) return fallback;
    return <img src={src} alt={alt} loading="lazy" decoding="async" className={className} {...props} />;
}
