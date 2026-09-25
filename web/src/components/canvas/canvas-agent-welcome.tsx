import { ArrowUpRight, Clapperboard, Layers3, Sparkles } from "lucide-react";
import { BrandLogo } from "@/components/brand/brand-logo";
import type { SkillCategory } from "@/services/api/skills";

type AgentWelcomeProps = {
    brandName: string;
    nodeCount: number;
    categories?: SkillCategory[];
    onChooseSkill: (tag?: string) => void;
    onDraftPrompt: (prompt: string) => void;
};

export function AgentWelcome({ brandName, nodeCount, categories = [], onChooseSkill, onDraftPrompt }: AgentWelcomeProps) {
    const name = brandName.trim() || "Agent";
    return (
        <section className="agent-welcome" aria-label="开始 Agent 创作">
            <div className="agent-welcome-intro">
                <span className="agent-welcome-logo-wrap" aria-hidden="true">
                    <BrandLogo className="agent-welcome-logo" alt="" fallback={<img src="/logo.png" alt="" className="agent-welcome-logo" draggable={false} />} />
                </span>
                <h2>在这里，和{name}让灵感，慢慢成形</h2>
                <p>从一个想法开始，和 Agent 一起创作。</p>
            </div>
            <div className="agent-welcome-actions">
                <button type="button" onClick={() => onChooseSkill()}>
                    <Sparkles aria-hidden="true" />
                    <span>
                        <strong>选择技能，开始创作</strong>
                        <small>为这次创作找到合适的帮手</small>
                    </span>
                    <ArrowUpRight className="agent-welcome-arrow" aria-hidden="true" />
                </button>
                <button type="button" onClick={() => onDraftPrompt("我想创作一段短片，请先和我一起梳理故事方向。先询问我的想法，不要直接生成。")}>
                    <Clapperboard aria-hidden="true" />
                    <span>
                        <strong>从灵感构思故事</strong>
                        <small>聊聊主题、角色，或一个难忘的画面</small>
                    </span>
                    <ArrowUpRight className="agent-welcome-arrow" aria-hidden="true" />
                </button>
                <button type="button" disabled={nodeCount === 0} onClick={() => onDraftPrompt("请先阅读当前画布，梳理素材与节点之间的关系，给出接下来的创作建议。先不要修改节点或提交生成任务。")}>
                    <Layers3 aria-hidden="true" />
                    <span>
                        <strong>一起梳理当前画布</strong>
                        <small>{nodeCount > 0 ? `${nodeCount} 个节点，看看下一步可以做什么` : "添加节点后，一起梳理创作思路"}</small>
                    </span>
                    <ArrowUpRight className="agent-welcome-arrow" aria-hidden="true" />
                </button>
            </div>
            <p className="agent-welcome-footnote">先聊想法，再决定下一步</p>
            {categories.length ? (
                <div className="agent-skill-picks">
                    <div className="agent-skill-picks-title">技能组合推荐</div>
                    <div className="agent-skill-picks-row" role="list">
                        {categories.filter((item) => item.value && item.value !== "all").slice(0, 8).map((item) => (
                            <button key={item.value} type="button" role="listitem" onClick={() => onChooseSkill(item.value)}>
                                {item.label || item.value}
                            </button>
                        ))}
                    </div>
                    <p>仅本会话生效 · 缺失技能将加入技能库 · Agent 按任务调用</p>
                </div>
            ) : null}
        </section>
    );
}
