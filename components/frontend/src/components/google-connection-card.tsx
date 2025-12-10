'use client'

import React from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { useGoogleStatus, useConnectGoogle, useDisconnectGoogle } from '@/services/queries'
import { successToast, errorToast } from '@/hooks/use-toast'

export function GoogleConnectionCard() {
  const { data: status, isLoading, refetch } = useGoogleStatus()
  const connectMutation = useConnectGoogle()
  const disconnectMutation = useDisconnectGoogle()

  const handleConnect = () => {
    connectMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('Google OAuth window opened. Please complete authorization.')
        // Refetch status after a delay to check if user completed OAuth
        setTimeout(() => refetch(), 5000)
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to initiate Google OAuth')
      },
    })
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('Google disconnected successfully')
        refetch()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to disconnect Google')
      },
    })
  }

  return (
    <Card className="bg-card border border-gray-200 shadow-sm">
      <div className="p-6">
        {/* Header section with icon and title */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-white border border-gray-200 rounded-lg flex items-center justify-center">
            <svg className="w-8 h-8" viewBox="0 0 24 24" aria-hidden="true">
              <path
                fill="#4285F4"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="#34A853"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="#FBBC05"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              />
              <path
                fill="#EA4335"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">Google Drive</h3>
            <p className="text-muted-foreground">Access and manage Google Drive files</p>
          </div>
        </div>

        {/* Status section */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className={`w-2 h-2 rounded-full ${status?.connected ? 'bg-green-500' : 'bg-gray-400'}`}></span>
            <span className="text-sm font-medium text-foreground/80">
              {status?.connected ? (
                <>Connected{status.email ? ` as ${status.email}` : ''}</>
              ) : (
                'Not Connected'
              )}
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            Connect your Google account to enable Google Drive access in agentic sessions
          </p>
        </div>

        {/* Action buttons */}
        <div className="flex gap-3">
          {status?.connected ? (
            <Button
              variant="destructive"
              onClick={handleDisconnect}
              disabled={isLoading || disconnectMutation.isPending}
            >
              Disconnect
            </Button>
          ) : (
            <Button
              onClick={handleConnect}
              disabled={isLoading || connectMutation.isPending}
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              Connect Google Drive
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}
