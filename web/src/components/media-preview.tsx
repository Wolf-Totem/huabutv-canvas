import { ImageOff } from "lucide-react";
import { useState } from "react";

import { mediaThumbUrl } from "@/lib/media-thumb";
import { cn } from "@/lib/utils";

const DEFAULT_UNAVAILABLE_LABEL = "预览不可用，素材可能已删除";

export function MediaPreview({
    src,
    kind,
    alt = "",
    className,
    fallbackClassName,
    fallbackLabel = DEFAULT_UNAVAILABLE_LABEL,
    controls = false,
    loading,
    width,
    height,
    onUnavailable,
}: {
    src: string;
    kind: "image" | "video";
    alt?: string;
    className?: string;
    fallbackClassName?: string;
    fallbackLabel?: string;
    controls?: boolean;
    loading?: "eager" | "lazy";
    width?: number;
    height?: number;
    onUnavailable?: () => void;
}) {
    const [failedSrc, setFailedSrc] = useState("");
    const unavailable = failedSrc === src;

    const handleUnavailable = () => {
        setFailedSrc(src);
        onUnavailable?.();
    };

    if (unavailable) {
        return (
            <span className={cn("media-unavailable", fallbackClassName)} role="img" aria-label={fallbackLabel} title={fallbackLabel}>
                <ImageOff aria-hidden="true" />
                <span>{fallbackLabel}</span>
            </span>
        );
    }

    if (kind === "video") {
        if (controls) {
            return <video src={src} width={width} height={height} muted={false} playsInline controls preload="metadata" className={className} onError={handleUnavailable} />;
        }
        const poster = mediaThumbUrl({ kind: "video", originalUrl: src, width: width && width > 0 ? width : 480 });
        if (!poster) {
            return (
                <span className={cn("media-unavailable", fallbackClassName)} role="img" aria-label={fallbackLabel} title={fallbackLabel}>
                    <ImageOff aria-hidden="true" />
                    <span>{fallbackLabel}</span>
                </span>
            );
        }
        return <img src={poster} alt={alt} width={width} height={height} loading={loading} className={className} onError={handleUnavailable} />;
    }

    const thumb = mediaThumbUrl({ kind: "image", originalUrl: src, width: width && width > 0 ? Math.max(width, 480) : 720 }) || src;
    return <img src={thumb} alt={alt} width={width} height={height} loading={loading} className={className} onError={handleUnavailable} />;
}
