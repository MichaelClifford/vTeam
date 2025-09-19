import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> }
) {
  try {
    const { name, sessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/stop`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...headers },
    });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error stopping agentic session:', error);
    return Response.json({ error: 'Failed to stop agentic session' }, { status: 500 });
  }
}


