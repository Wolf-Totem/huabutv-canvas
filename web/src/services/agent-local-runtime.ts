import localforage from "localforage";

import { getAgentCapabilities } from "@/services/api/agent";
import { listAddedSkills } from "@/services/api/skills";

const store = localforage.createInstance({ name: "infinite-canvas", storeName: "agent_runtime" });
const BOOTSTRAP_KEY = "cloud-agent-runtime-bootstrap-v1";

export type LocalAgentBootstrap = {
    fetchedAt: string;
    capabilities: unknown;
    skillCount: number;
};

export async function ensureLocalAgentRuntime() {
    const cached = await store.getItem<LocalAgentBootstrap>(BOOTSTRAP_KEY);
    if (cached?.fetchedAt) return cached;
    return refreshLocalAgentRuntime();
}

export async function refreshLocalAgentRuntime() {
    const [capabilities, added] = await Promise.all([getAgentCapabilities(), listAddedSkills()]);
    const snapshot: LocalAgentBootstrap = {
        fetchedAt: new Date().toISOString(),
        capabilities,
        skillCount: added.skills?.length || 0,
    };
    await store.setItem(BOOTSTRAP_KEY, snapshot);
    return snapshot;
}
