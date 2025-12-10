/**
 * React Query hooks for Google OAuth integration
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as googleApi from '../api/google';

/**
 * Query keys for Google OAuth
 */
export const googleKeys = {
  all: ['google'] as const,
  status: () => [...googleKeys.all, 'status'] as const,
};

/**
 * Hook to fetch Google OAuth connection status
 */
export function useGoogleStatus() {
  return useQuery({
    queryKey: googleKeys.status(),
    queryFn: googleApi.getGoogleStatus,
    // Check status less frequently
    staleTime: 60 * 1000, // 1 minute
  });
}

/**
 * Hook to initiate Google OAuth connection
 */
export function useConnectGoogle() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: googleApi.connectGoogle,
    onSuccess: () => {
      // Invalidate status after OAuth window opens
      // Status will update when OAuth callback completes
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: googleKeys.status() });
      }, 2000);
    },
  });
}

/**
 * Hook to disconnect Google OAuth
 */
export function useDisconnectGoogle() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: googleApi.disconnectGoogle,
    onSuccess: () => {
      // Invalidate status to show disconnected state
      queryClient.invalidateQueries({ queryKey: googleKeys.status() });
    },
  });
}
