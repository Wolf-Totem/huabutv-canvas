import { useCallback, useState } from "react";

export function useCanvasAssistantVisibility() {
    const [assistantOpen, setAssistantOpen] = useState(true);

    const openAgent = useCallback(() => {
        setAssistantOpen(true);
    }, []);

    const closeAgent = useCallback(() => {
        setAssistantOpen(false);
    }, []);

    return {
        assistantClosing: false,
        assistantMounted: true,
        assistantOpen,
        closeAgent,
        openAgent,
    };
}
