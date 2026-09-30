import { expect, test } from "bun:test";

import {
    agentConsoleURL,
    agentHostForParent,
    canvasHostForParent,
    canvasWorkspaceURL,
    isAgentHost,
    isCanvasHost,
    isStreamerMarketingHost,
    parentDomainOfHost,
    publicAgentHost,
    publicCanvasHost,
} from "../src/lib/public-hosts";

test("public host defaults use huabutv.com and www canvas", () => {
    expect(publicCanvasHost()).toBe("www.huabutv.com");
    expect(publicAgentHost()).toBe("agent.huabutv.com");
    expect(canvasHostForParent("j11.net")).toBe("canvas.j11.net");
    expect(canvasHostForParent("huabutv.com")).toBe("www.huabutv.com");
    expect(agentHostForParent("j11.net")).toBe("agent.j11.net");
    expect(agentHostForParent("huabutv.com")).toBe("agent.huabutv.com");
});

test("parentDomainOfHost matches known parents only", () => {
    expect(parentDomainOfHost("www.huabutv.com")).toBe("huabutv.com");
    expect(parentDomainOfHost("a.j11.net")).toBe("j11.net");
    expect(parentDomainOfHost("canvas.j11.net")).toBe("j11.net");
    expect(parentDomainOfHost("huabutv.com")).toBe("huabutv.com");
    expect(parentDomainOfHost("evil.example")).toBe("");
    expect(parentDomainOfHost("localhost")).toBe("");
});

test("streamer marketing hosts stay on both parents after default flip", () => {
    expect(isStreamerMarketingHost("a.j11.net")).toBe(true);
    expect(isStreamerMarketingHost("a.huabutv.com")).toBe(true);
    expect(isStreamerMarketingHost("agent.j11.net")).toBe(false);
    expect(isStreamerMarketingHost("www.huabutv.com")).toBe(false);
    expect(isStreamerMarketingHost("canvas.j11.net")).toBe(false);
    expect(isStreamerMarketingHost("oss.huabutv.com")).toBe(false);
    expect(isAgentHost("agent.j11.net")).toBe(true);
    expect(isCanvasHost("www.huabutv.com")).toBe(true);
    expect(isCanvasHost("canvas.j11.net")).toBe(true);
    expect(isCanvasHost("huabutv.com")).toBe(true);
    expect(isCanvasHost("j11.net")).toBe(true);
});

test("workspace and agent URLs stay on the request parent", () => {
    expect(canvasWorkspaceURL("a.j11.net")).toBe("https://canvas.j11.net/");
    expect(canvasWorkspaceURL("a.huabutv.com")).toBe("https://www.huabutv.com/");
    expect(agentConsoleURL("a.j11.net")).toBe("https://agent.j11.net/");
    expect(canvasWorkspaceURL("www.huabutv.com")).toBe("/");
    expect(canvasWorkspaceURL("canvas.j11.net")).toBe("/");
    expect(canvasWorkspaceURL("")).toBe("https://www.huabutv.com/");
    expect(agentConsoleURL("")).toBe("https://agent.huabutv.com/");
    expect(agentConsoleURL("agent.huabutv.com")).toBe("/agent");
    expect(canvasWorkspaceURL("localhost")).toBe("/");
});
