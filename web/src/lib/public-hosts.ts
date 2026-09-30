export const KNOWN_PARENT_DOMAINS = ["huabutv.com", "j11.net"] as const;
export const DEFAULT_PARENT_DOMAIN = "huabutv.com";

const RESERVED_LABELS = new Set(["www", "app", "api", "agent", "admin", "static", "cdn", "mail", "canvas", "localhost", "oss", "smtp", "track", "mx", "ns", "email"]);

export function currentHostname() {
    if (typeof window === "undefined") return "";
    return window.location.hostname.toLowerCase();
}

function normalizeHost(host: string) {
    return String(host || "").split(":")[0].trim().toLowerCase().replace(/\.$/, "");
}

export function leftmostHostLabel(host = currentHostname()) {
    const hostname = normalizeHost(host);
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
    return String(fromApi || `www.${parent}`).trim().toLowerCase();
}

export function parentDomainOfHost(host: string): string | "" {
    const hostname = normalizeHost(host);
    if (!hostname) return "";
    for (const parent of KNOWN_PARENT_DOMAINS) {
        if (hostname === parent || hostname.endsWith(`.${parent}`)) return parent;
    }
    const primary = publicParentDomain();
    if (hostname === primary || hostname.endsWith(`.${primary}`)) return primary;
    return "";
}

export function canvasHostForParent(parent: string) {
    if (parent === "j11.net") return "canvas.j11.net";
    return publicCanvasHost(undefined, parent || DEFAULT_PARENT_DOMAIN);
}

export function agentHostForParent(parent: string) {
    if (parent === "j11.net") return "agent.j11.net";
    return publicAgentHost(undefined, parent || DEFAULT_PARENT_DOMAIN);
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
    const hostname = normalizeHost(host);
    const label = leftmostHostLabel(hostname);
    if (label === "canvas" || label === "www") return true;
    if (KNOWN_PARENT_DOMAINS.some((parent) => hostname === parent)) return true;
    return hostname === publicParentDomain();
}

export function isStreamerMarketingHost(host = currentHostname()) {
    if (!host || isAgentHost(host) || isCanvasHost(host)) return false;
    const label = leftmostHostLabel(host);
    if (!label || RESERVED_LABELS.has(label)) return false;
    return Boolean(parentDomainOfHost(host));
}

function isLoopback(host: string) {
    const hostname = normalizeHost(host);
    return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "::1";
}

export function canvasWorkspaceURL(host = currentHostname()) {
    if (!host) return "https://www.huabutv.com/";
    if (isCanvasHost(host) || isLoopback(host)) return "/";
    return `${publicOrigin(canvasHostForParent(parentDomainOfHost(host)))}/`;
}

export function agentConsoleURL(host = currentHostname()) {
    if (!host) return "https://agent.huabutv.com/";
    if (isAgentHost(host)) return "/agent";
    return `${publicOrigin(agentHostForParent(parentDomainOfHost(host)))}/`;
}
