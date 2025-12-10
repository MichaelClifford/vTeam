# Google Drive Integration

This document describes the Google Drive OAuth integration in the ambient frontend.

## Overview

Users can connect their Google accounts to enable Google Drive access in agentic sessions. The integration uses the existing OAuth2 callback infrastructure built in the backend.

## Architecture

### OAuth Flow

1. **User clicks "Connect Google Drive"** in the Integrations page
2. **Frontend opens OAuth window** with Google authorization URL
3. **User authorizes** in Google's OAuth consent screen
4. **Google redirects to** `/oauth2callback` (backend endpoint)
5. **Backend exchanges code** for access token and stores in Secret
6. **User sees success** page and closes OAuth window
7. **Frontend refetches status** to show connected state

### Data Flow for Agentic Sessions

The implementation is designed to support injecting OAuth credentials into runner pods:

```
User Connects Google
    ↓
Backend stores tokens in Secret: `oauth-callbacks` (keyed by state)
    ↓
Future: Backend copies tokens to Secret: `google-oauth-connections` (keyed by userId)
    ↓
Future: Runner pod mounts Secret or receives credentials via env vars
    ↓
Future: MCP Google Workspace uses credentials in agentic session
```

**Note**: Currently the OAuth callback data is stored in `oauth-callbacks` Secret. To support runner pod injection, you'll need to:

1. Add backend endpoint to retrieve user's OAuth tokens by userId
2. Store persistent connection in `google-oauth-connections` Secret
3. Modify runner pod spec to mount or inject credentials
4. Configure MCP server to use injected credentials instead of local OAuth

## Frontend Components

### Files Created

1. **`src/services/api/google.ts`**
   - `connectGoogle()` - Opens Google OAuth window
   - `getGoogleStatus()` - Checks connection status (currently returns mock data)
   - `disconnectGoogle()` - Disconnects Google account (currently returns mock)

2. **`src/services/queries/use-google.ts`**
   - `useGoogleStatus()` - React Query hook for status
   - `useConnectGoogle()` - React Query hook to initiate OAuth
   - `useDisconnectGoogle()` - React Query hook to disconnect

3. **`src/components/google-connection-card.tsx`**
   - UI card component following GitHub card pattern
   - Shows connection status
   - Connect/Disconnect buttons
   - Google Drive branding

### Files Modified

1. **`src/services/queries/index.ts`** - Added `use-google` export
2. **`src/app/integrations/IntegrationsClient.tsx`** - Added GoogleConnectionCard

## Configuration

### Environment Variables

Add to your frontend `.env.local` or environment:

```bash
# Google OAuth Client ID (public, safe for frontend)
NEXT_PUBLIC_GOOGLE_OAUTH_CLIENT_ID=your-google-client-id
```

This matches the backend's `GOOGLE_OAUTH_CLIENT_ID` and `GOOGLE_OAUTH_CLIENT_SECRET`.

### Backend Configuration

The backend OAuth handler (`components/backend/handlers/oauth.go`) already supports Google OAuth:

- **Callback URL**: `http://localhost:8000/oauth2callback` (or `GOOGLE_OAUTH_REDIRECT_URI` env var)
- **Token Storage**: Kubernetes Secret `oauth-callbacks`
- **Scopes**: userinfo.email, drive, drive.file, drive.readonly

## Future Enhancements

To fully enable Google Drive in agentic sessions, implement:

### 1. Backend User Connection Management

Add endpoints to `components/backend/handlers/oauth.go`:

```go
// GET /api/auth/google/status
func GetGoogleStatusGlobal(c *gin.Context) {
    // Check if user has active Google connection
    // Return { connected: bool, email: string, updatedAt: string }
}

// POST /api/auth/google/disconnect
func DisconnectGoogleGlobal(c *gin.Context) {
    // Delete user's Google connection from Secret
}

// Helper: After OAuth callback succeeds, move token to user-specific storage
func moveOAuthCallbackToUserConnection(state, userId string) {
    // 1. Get callback data from oauth-callbacks Secret
    // 2. Create GoogleOAuthConnection with userId
    // 3. Store in google-oauth-connections Secret
    // 4. Delete from oauth-callbacks
}
```

### 2. Update Frontend API Client

Modify `src/services/api/google.ts`:

```typescript
export async function getGoogleStatus(): Promise<GoogleOAuthStatus> {
  return apiClient.get<GoogleOAuthStatus>('/auth/google/status');
}

export async function disconnectGoogle(): Promise<string> {
  const response = await apiClient.post<{ message: string }>('/auth/google/disconnect');
  return response.message;
}
```

### 3. Runner Pod Credential Injection

In the operator (`components/operator/internal/handlers/sessions.go`), when creating the runner pod Job:

```go
// Add Google OAuth credentials to runner pod
func injectGoogleCredentials(pod *corev1.PodSpec, userID string) error {
    // Option A: Mount Secret
    pod.Volumes = append(pod.Volumes, corev1.Volume{
        Name: "google-oauth",
        VolumeSource: corev1.VolumeSource{
            Secret: &corev1.SecretVolumeSource{
                SecretName: "google-oauth-connections",
                Items: []corev1.KeyToPath{{
                    Key:  userID,
                    Path: "credentials.json",
                }},
            },
        },
    })

    // Option B: Environment variables
    // (Less secure, but simpler for MCP integration)
    pod.Containers[0].Env = append(pod.Containers[0].Env,
        corev1.EnvVar{
            Name: "GOOGLE_OAUTH_ACCESS_TOKEN",
            ValueFrom: &corev1.EnvVarSource{
                SecretKeyRef: &corev1.SecretKeySelector{
                    LocalObjectReference: corev1.LocalObjectReference{
                        Name: "google-oauth-connections",
                    },
                    Key: userID + "_access_token",
                },
            },
        },
    )
}
```

### 4. MCP Server Configuration in Runner

Update runner's `.mcp.json` dynamically:

```python
# In runner startup script
def configure_mcp_with_google_creds():
    """Configure MCP Google Workspace server with injected credentials"""

    # Read injected credentials
    access_token = os.getenv("GOOGLE_OAUTH_ACCESS_TOKEN")
    refresh_token = os.getenv("GOOGLE_OAUTH_REFRESH_TOKEN")

    # Update .mcp.json
    mcp_config = {
        "mcpServers": {
            "google_workspace": {
                "type": "stdio",
                "command": "uvx",
                "args": ["workspace-mcp", "--tools", "drive"],
                "env": {
                    "GOOGLE_OAUTH_ACCESS_TOKEN": access_token,
                    "GOOGLE_OAUTH_REFRESH_TOKEN": refresh_token,
                    # Or use credentials file path if mounted
                    # "GOOGLE_APPLICATION_CREDENTIALS": "/secrets/google-oauth/credentials.json"
                }
            }
        }
    }

    with open(".mcp.json", "w") as f:
        json.dump(mcp_config, f)
```

## Testing

### Manual Testing

1. Start frontend: `npm run dev`
2. Navigate to `/integrations`
3. Click "Connect Google Drive" on the Google card
4. OAuth window should open
5. Complete authorization in Google
6. Backend receives callback at `/oauth2callback`
7. Success page displays, window can be closed
8. Frontend status should update (when backend endpoints are added)

### Backend Testing

```bash
# Check oauth-callbacks Secret
kubectl get secret oauth-callbacks -n ambient-code -o yaml

# Decode callback data
kubectl get secret oauth-callbacks -n ambient-code -o jsonpath='{.data}' | jq

# View backend logs during OAuth
kubectl logs -f deployment/vteam-backend -n ambient-code
```

## Security Considerations

1. **Token Storage**: OAuth tokens stored in Kubernetes Secrets (not ConfigMaps)
2. **Scope Limitation**: Only requests necessary Drive scopes
3. **State Parameter**: CSRF protection via state parameter
4. **Token Refresh**: Requests `offline` access for refresh tokens
5. **Runner Isolation**: Each runner pod gets isolated credentials via Secret projection

## Design Decisions

### Why popup window instead of redirect?

- Better UX: User stays on integrations page
- Simpler state management: No navigation state to preserve
- Consistent with GitHub App flow pattern

### Why not use MCP's built-in OAuth?

- MCP's OAuth server runs in runner pods (ephemeral)
- We need persistent, user-scoped credentials
- Backend can refresh tokens and manage lifecycle
- Credentials can be shared across sessions

### Why separate `oauth-callbacks` and `google-oauth-connections`?

- `oauth-callbacks`: Temporary storage for OAuth flow (keyed by state)
- `google-oauth-connections`: Persistent user connections (keyed by userId)
- Separation allows cleanup and different access patterns

## Troubleshooting

### OAuth window doesn't open

- Check `NEXT_PUBLIC_GOOGLE_OAUTH_CLIENT_ID` is set
- Check browser popup blocker settings
- Verify OAuth client ID is valid

### "Not connected" after completing OAuth

- Backend endpoints not implemented yet (expected)
- Check backend logs for callback processing
- Verify `oauth-callbacks` Secret was created

### Runner pod can't access Google Drive

- Future feature - credential injection not yet implemented
- Will require backend and operator changes described above

## Related Files

- Backend OAuth handler: `components/backend/handlers/oauth.go`
- Backend OAuth docs: `components/backend/OAUTH_INTEGRATION.md`
- Backend routes: `components/backend/routes.go`
- Frontend integration page: `src/app/integrations/IntegrationsClient.tsx`
