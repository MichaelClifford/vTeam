/**
 * Google OAuth Integration API service
 * Handles all Google OAuth-related API calls
 */

export type GoogleOAuthStatus = {
  connected: boolean;
  email?: string;
  userId?: string;
  updatedAt?: string;
};

/**
 * Initiate Google OAuth flow
 * Opens OAuth authorization URL in a new window
 */
export async function connectGoogle(): Promise<void> {
  const clientId = process.env.NEXT_PUBLIC_GOOGLE_OAUTH_CLIENT_ID;
  if (!clientId) {
    throw new Error('Google OAuth client ID not configured');
  }

  // Construct OAuth URL for Google Drive access
  const scopes = [
    'https://www.googleapis.com/auth/userinfo.email',
    'https://www.googleapis.com/auth/drive',
    'https://www.googleapis.com/auth/drive.file',
    'https://www.googleapis.com/auth/drive.readonly',
  ];

  // Redirect to backend OAuth callback endpoint
  // In local dev: http://localhost:8080/oauth2callback
  // In production: same origin as frontend (via proxy/ingress)
  const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
  const redirectUri = `${backendUrl}/oauth2callback`;
  const state = btoa(`google:${Date.now()}`);

  const authUrl = new URL('https://accounts.google.com/o/oauth2/v2/auth');
  authUrl.searchParams.set('client_id', clientId);
  authUrl.searchParams.set('redirect_uri', redirectUri);
  authUrl.searchParams.set('response_type', 'code');
  authUrl.searchParams.set('scope', scopes.join(' '));
  authUrl.searchParams.set('access_type', 'offline');
  authUrl.searchParams.set('state', state);
  authUrl.searchParams.set('prompt', 'consent');

  // Open OAuth flow in new window
  const width = 600;
  const height = 700;
  const left = window.screen.width / 2 - width / 2;
  const top = window.screen.height / 2 - height / 2;

  window.open(
    authUrl.toString(),
    'Google OAuth',
    `width=${width},height=${height},left=${left},top=${top}`
  );
}

/**
 * Get Google OAuth connection status
 * Note: Currently returns mock data - backend endpoints can be added later
 */
export async function getGoogleStatus(): Promise<GoogleOAuthStatus> {
  // For now, check if we have OAuth callback data stored
  // Backend can add GET /api/auth/google/status endpoint later
  // When backend adds status endpoint, use: apiClient.get<GoogleOAuthStatus>('/auth/google/status')
  return { connected: false };
}

/**
 * Disconnect Google OAuth
 * Note: Backend endpoint can be added later at POST /api/auth/google/disconnect
 */
export async function disconnectGoogle(): Promise<string> {
  // Backend can add POST /api/auth/google/disconnect endpoint later
  // For now, return success message
  return 'Google account will be disconnected';
}
