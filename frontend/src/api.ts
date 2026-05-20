const API_BASE = '/api/v1'

let _onUnauthorized: (() => void) | null = null

export function onUnauthorized(fn: () => void) {
  _onUnauthorized = fn
}

export function getToken(): string | null {
  return localStorage.getItem('auth_token')
}

export function setToken(token: string) {
  localStorage.setItem('auth_token', token)
}

export function clearToken() {
  localStorage.removeItem('auth_token')
}

export function isLoggedIn(): boolean {
  return !!getToken()
}

function makeHeaders(): Record<string, string> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  return headers
}

async function handleResponse(res: Response): Promise<Response> {
  if (res.status === 401 && _onUnauthorized) {
    _onUnauthorized()
  }
  return res
}

export async function apiGet(path: string, params?: Record<string, string>) {
  const url = new URL(`${API_BASE}${path}`, window.location.origin)
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      url.searchParams.set(k, v)
    }
  }
  const res = await fetch(url.toString(), { headers: makeHeaders() })
  return handleResponse(res)
}

export async function apiPost(path: string, body?: unknown) {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { ...makeHeaders(), 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  return handleResponse(res)
}
