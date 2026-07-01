export async function loadServerKeys() {
  const response = await fetch('/key', { cache: 'no-store' });
  if (!response.ok) throw new Error('key request failed');
  return response.json();
}

export async function registerUser(name, tag) {
  const response = await fetch('/register', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name, tag }) });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || 'registration failed');
  return data.user;
}
