export const rememberedUserKey = 'friendsChatUser';
export const rememberEnabledKey = 'friendsChatRememberUser';
export const state = { socket: null, user: null, users: [], publicMessages: [], dialogs: new Map(), activeDialogId: null, mentionAlert: false };
export function resetChatState() { state.socket = null; state.users = []; state.publicMessages = []; state.dialogs = new Map(); state.activeDialogId = null; state.mentionAlert = false; }
export function getDialog(id, user) { if (!state.dialogs.has(id)) state.dialogs.set(id, { id, name: (user && user.name) || id, tag: (user && user.tag) || '', messages: [], unread: false }); const dialog = state.dialogs.get(id); if (user && user.name) dialog.name = user.name; if (user && user.tag) dialog.tag = user.tag; return dialog; }
export function saveRememberedUser(remember) { if (remember && state.user) { localStorage.setItem(rememberEnabledKey, 'true'); localStorage.setItem(rememberedUserKey, JSON.stringify(state.user)); return; } localStorage.removeItem(rememberEnabledKey); localStorage.removeItem(rememberedUserKey); }
export function loadRememberedUser() { if (localStorage.getItem(rememberEnabledKey) !== 'true') return null; try { return JSON.parse(localStorage.getItem(rememberedUserKey) || 'null'); } catch { return null; } }
