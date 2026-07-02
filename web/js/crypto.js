const encoder = new TextEncoder();
const decoder = new TextDecoder();
const encryptionSaltText = 'SM_BCE-v1';
const encryptionSalt = encoder.encode(encryptionSaltText);
const hasWebCrypto = Boolean(globalThis.crypto && globalThis.crypto.subtle);
const decryptErrorText = 'Не удалось расшифровать сообщение';

let currentCryptoKey = null;
let currentKeyDay = '';
let cryptoKeys = new Map();

export async function setServerKeys(data) {
  if (!data.key || !data.day) throw new Error('empty key');

  currentKeyDay = data.day;
  cryptoKeys = new Map();

  const keys = data.keys || { [data.day]: data.key };
  for (const [day, key] of Object.entries(keys)) {
    cryptoKeys.set(day, await createCryptoKey(key));
  }
  currentCryptoKey = cryptoKeys.get(currentKeyDay);
}

export function resetCrypto() {
  currentCryptoKey = null;
  currentKeyDay = '';
  cryptoKeys = new Map();
}

export async function encryptText(text) {
  if (hasWebCrypto) return encryptWithWebCrypto(text);
  return encryptWithFallback(text);
}

export async function decryptMessage(msg) {
  try {
    const payload = JSON.parse(msg.text);
    if (payload.v !== 1 || !payload.iv || !payload.data) return msg.text;
    if (payload.alg === 'SM-FALLBACK') return decryptWithFallback(payload, msg.keyDay || currentKeyDay);
    if (payload.alg === 'AES-GCM' && hasWebCrypto) return decryptWithWebCrypto(payload, msg.keyDay || currentKeyDay);
    return decryptErrorText;
  } catch {
    return decryptErrorText;
  }
}

async function encryptWithWebCrypto(text) {
  const iv = globalThis.crypto.getRandomValues(new Uint8Array(12));
  const encrypted = await globalThis.crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    currentCryptoKey,
    encoder.encode(text)
  );
  return {
    keyDay: currentKeyDay,
    text: JSON.stringify({ v: 1, alg: 'AES-GCM', iv: bytesToBase64(iv), data: bytesToBase64(new Uint8Array(encrypted)) })
  };
}

async function decryptWithWebCrypto(payload, keyDay) {
  const key = cryptoKeys.get(keyDay) || currentCryptoKey;
  if (!key) return decryptErrorText;

  const decrypted = await globalThis.crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: base64ToBytes(payload.iv) },
    key,
    base64ToBytes(payload.data)
  );
  return decoder.decode(decrypted);
}

async function createCryptoKey(passphrase) {
  if (!hasWebCrypto) return createFallbackKey(passphrase);

  const baseKey = await globalThis.crypto.subtle.importKey('raw', encoder.encode(passphrase), 'PBKDF2', false, ['deriveKey']);
  return globalThis.crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt: encryptionSalt, iterations: 120000, hash: 'SHA-256' },
    baseKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

function createFallbackKey(passphrase) {
  const bytes = encoder.encode(encryptionSaltText + ':' + passphrase);
  let seed = 2166136261;
  for (const byte of bytes) {
    seed ^= byte;
    seed = Math.imul(seed, 16777619) >>> 0;
  }
  return seed || 1;
}

function encryptWithFallback(text) {
  const iv = randomFallbackIV();
  const key = cryptoKeys.get(currentKeyDay) || currentCryptoKey;
  const encrypted = xorBytes(encoder.encode(text), key, iv);
  return {
    keyDay: currentKeyDay,
    text: JSON.stringify({ v: 1, alg: 'SM-FALLBACK', iv: String(iv), data: bytesToBase64(encrypted) })
  };
}

function decryptWithFallback(payload, keyDay) {
  const key = cryptoKeys.get(keyDay) || currentCryptoKey;
  if (!key) return decryptErrorText;

  const decrypted = xorBytes(base64ToBytes(payload.data), key, Number(payload.iv) || 1);
  return decoder.decode(decrypted);
}

function xorBytes(bytes, key, iv) {
  let state = (key ^ iv) >>> 0;
  const out = new Uint8Array(bytes.length);

  for (let i = 0; i < bytes.length; i += 1) {
    state = (Math.imul(state, 1664525) + 1013904223) >>> 0;
    out[i] = bytes[i] ^ (state & 255);
  }
  return out;
}

function randomFallbackIV() {
  if (globalThis.crypto && globalThis.crypto.getRandomValues) {
    const value = new Uint32Array(1);
    globalThis.crypto.getRandomValues(value);
    return value[0] || 1;
  }
  return Math.floor(Math.random() * 4294967295) || 1;
}

function bytesToBase64(bytes) {
  let binary = '';
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary);
}

function base64ToBytes(base64) {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
