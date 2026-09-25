const DEFAULT_PARENT_DOMAIN = "j11.net";

const RESERVED_LABELS = new Set(["www", "app", "api", "agent", "admin", "static", "cdn", "mail", "canvas", "localhost"]);

export function currentHostname() {
    if (typeof window === "undefined") return "";
    return window.location.hostname.toLowerCase();
}

export function leftmostHostLabel(host = currentHostname()) {
    const hostname = host.split(":")[0].trim().toLowerCase();
    return hostname.split(".")[0] || "";
}

export function publicParentDomain(fromApi?: string) {
    const value = String(fromApi || DEFAULT_PARENT_DOMAIN).replace(/^\./, "").trim().toLowerCase();
    return value || DEFAULT_PARENT_DOMAIN;
}

export function publicAgentHost(fromApi?: string, parent = publicParentDomain()) {
    return String(fromApi || `agent.${parent}`).trim().toLowerCase();
}

export function publicCanvasHost(fromApi?: string, parent = publicParentDomain()) {
    return String(fromApi || `canvas.${parent}`).trim().toLowerCase();
}

export function publicOrigin(host: string) {
    const hostname = String(host || "").trim().toLowerCase();
    if (!hostname) return "";
    if (typeof window !== "undefined" && window.location.protocol === "http:" && (hostname.startsWith("localhost") || hostname.endsWith(".local"))) {
        return `${window.location.protocol}//${hostname}${window.location.port ? `:${window.location.port}` : ""}`;
    }
    return `https://${hostname}`;
}

export function isAgentHost(host = currentHostname()) {
    return leftmostHostLabel(host) === "agent";
}

export function isCanvasHost(host = currentHostname()) {
    const label = leftmostHostLabel(host);
    return label === "canvas" || label === "www" || host === publicParentDomain();
}

export function isStreamerMarketingHost(host = currentHostname()) {
    if (!host || isAgentHost(host) || isCanvasHost(host)) return false;
    const label = leftmostHostLabel(host);
    if (!label || RESERVED_LABELS.has(label)) return false;
    return host.endsWith(`.${publicParentDomain()}`);
}

export function canvasWorkspaceURL() {
    if (typeof window === "undefined") return "https://canvas.j11.net/";
    if (isCanvasHost() || currentHostname() === "localhost" || currentHostname() === "127.0.0.1") {
        return "/";
    }
    return `${publicOrigin(publicCanvasHost())}/`;
}

export function agentConsoleURL() {
    if (typeof window === "undefined") return "https://agent.j11.net/";
    if (isAgentHost()) return "/agent";
    return `${publicOrigin(publicAgentHost())}/`;
}
