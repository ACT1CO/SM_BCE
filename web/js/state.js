export const rememberedUserKey = 'SM_BCE_user';
export const rememberEnabledKey = 'SM_BCE_remember_user';

export const state = {
  socket: null,
  user: null,
  users: [],
  publicMessages: [],
  dialogs: new Map(),
  activeDialogId: null,
  mentionAlert: false,
  publicUnreadCount: 0,
  readState: { public: '', dialogs: {} }
};

export function resetChatState() {
  state.socket = null;
  state.users = [];
  state.publicMessages = [];
  state.dialogs = new Map();
  state.activeDialogId = null;
  state.mentionAlert = false;
  state.publicUnreadCount = 0;
  state.readState = { public: '', dialogs: {} };
}

export function getDialog(id, user) {
  if (!state.dialogs.has(id)) {
    state.dialogs.set(id, {
      id,
      name: (user && user.name) || id,
      tag: (user && user.tag) || '',
      online: Boolean(user && user.online),
      messages: [],
      unreadCount: 0
    });
  }
  const dialog = state.dialogs.get(id);
  if (user && user.name) dialog.name = user.name;
  if (user && user.tag) dialog.tag = user.tag;
  if (user && typeof user.online === 'boolean') dialog.online = user.online;
  return dialog;
}

export function getReadStateKey() {
  return state.user ? `SM_BCE_read_state_${state.user.id}` : 'SM_BCE_read_state';
}

export function loadReadState() {
  try {
    state.readState = JSON.parse(localStorage.getItem(getReadStateKey()) || '{"public":"","dialogs":{}}');
  } catch {
    state.readState = { public: '', dialogs: {} };
  }
  if (!state.readState.dialogs) state.readState.dialogs = {};
}

export function saveReadState() {
  if (!state.user) return;
  localStorage.setItem(getReadStateKey(), JSON.stringify(state.readState));
}

export function markPublicRead() {
  const last = lastMessageID(state.publicMessages);
  if (last) state.readState.public = last;
  state.publicUnreadCount = 0;
  state.mentionAlert = false;
  saveReadState();
}

export function markDialogRead(dialogId) {
  const dialog = state.dialogs.get(dialogId);
  if (!dialog) return;
  const last = lastMessageID(dialog.messages);
  if (last) state.readState.dialogs[dialogId] = last;
  dialog.unreadCount = 0;
  saveReadState();
}

export function isMessageUnreadInPublic(msg) {
  return Boolean(msg.id && state.readState.public && msg.id > state.readState.public && msg.from !== state.user?.id);
}

export function isMessageUnreadInDialog(dialogId, msg) {
  const lastRead = state.readState.dialogs[dialogId] || '';
  return Boolean(msg.id && lastRead && msg.id > lastRead && msg.from !== state.user?.id);
}

export function lastMessageID(messages) {
  for (let i = messages.length - 1; i >= 0; i -= 1) {
    if (messages[i].id) return messages[i].id;
  }
  return '';
}

export function saveRememberedUser(remember) {
  if (remember && state.user) {
    localStorage.setItem(rememberEnabledKey, 'true');
    localStorage.setItem(rememberedUserKey, JSON.stringify(state.user));
    return;
  }
  localStorage.removeItem(rememberEnabledKey);
  localStorage.removeItem(rememberedUserKey);
}

export function loadRememberedUser() {
  if (localStorage.getItem(rememberEnabledKey) !== 'true') return null;
  try {
    return JSON.parse(localStorage.getItem(rememberedUserKey) || 'null');
  } catch {
    return null;
  }
}
