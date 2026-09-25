import type { ReactNode } from "react";

import { CachedResourceImage } from "@/components/cached-resource-image";
import { MediaThumb } from "@/components/media-thumb";
import { mediaThumbUrl } from "@/lib/media-thumb";
import type { Asset } from "@/stores/use-asset-store";

type AssetMediaPreviewProps = {
    asset?: Asset | null;
    alt: string;
    className?: string;
    fallback?: ReactNode;
};

export function AssetMediaPreview({ asset, alt, className = "", fallback = null }: AssetMediaPreviewProps) {
    if (!asset) return fallback;

    if (asset.kind === "video") {
        return <MediaThumb kind="video" coverUrl={asset.coverUrl} originalUrl={asset.data.url} alt={alt} className={className} fallback={fallback} />;
    }

    if (asset.kind !== "image") return fallback;

    const original = asset.coverUrl || asset.data.dataUrl || "";
    const thumb = mediaThumbUrl({ kind: "image", coverUrl: asset.coverUrl, originalUrl: original });
    if (thumb && /^https?:\/\//i.test(thumb)) {
        return <img src={thumb} alt={alt} loading="lazy" decoding="async" className={className} />;
    }
    if (!thumb && !asset.data.storageKey) return fallback;
    return <CachedResourceImage storageKey={asset.data.storageKey} src={thumb || original} alt={alt} loading="lazy" decoding="async" className={className} fallback={fallback} />;
}
