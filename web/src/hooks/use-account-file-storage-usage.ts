import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import { accountFileStorageUsageQueryKey } from "@/lib/account-storage-usage";
import { getAccountFileStorageUsage } from "@/services/api/resources";

export function useAccountFileStorageUsage(enabled = true) {
    const queryClient = useQueryClient();
    const query = useQuery({
        queryKey: accountFileStorageUsageQueryKey,
        queryFn: getAccountFileStorageUsage,
        enabled,
        staleTime: 30_000,
        refetchOnMount: "always",
    });

    useEffect(() => {
        const handleUpdated = () => void queryClient.invalidateQueries({ queryKey: accountFileStorageUsageQueryKey });
        window.addEventListener("wallet:updated", handleUpdated);
        return () => window.removeEventListener("wallet:updated", handleUpdated);
    }, [queryClient]);

    return query;
}
