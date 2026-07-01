import { loadServerKeys, registerUser } from './api.js';
import { decryptMessage, encryptText, resetCrypto, setServerKeys } from './crypto.js';
import { state, getDialog, loadReadState, loadRememberedUser, markDialogRead, markPublicRead, resetChatState, saveReadState, saveRememberedUser } from './state.js';
import { clearLoginError, closeDrawer, closeSettings, el, hideMentionSuggestions, openDrawer, renderCurrentChat, renderMentionSuggestions, renderPeople, showChat, showLogin, showLoginError, showPublicChat, toggleSettings, updatePulse } from './ui.js';

export function initChat() {
  const remembered = loadRememberedUser();
  if (remembered) {
    state.user = remembered;
    el.nameInput.value = remembered.name;
    el.tagInput.value = remembered.tag;
    el.rememberInput.checked = true;
  }
  el.loginForm.addEventListener('submit', handleLogin);
  el.messageForm.addEventListener('submit', handleSend);
  el.leaveBtn.addEventListener('click', leaveChat);
  el.publicBtn.addEventListener('click', showPublicChat);
  el.privateBtn.addEventListener('click', () => { openDrawer(); closeSettings(); });
  el.settingsBtn.addEventListener('click', () => { toggleSettings(); closeDrawer(); });
  el.closeSettingsBtn.addEventListener('click', closeSettings);
  el.closeDrawerBtn.addEventListener('click', closeDrawer);
  el.drawerBackdrop.addEventListener('click', () => { closeDrawer(); closeSettings(); });
  el.messageInput.addEventListener('input', () => renderMentionSuggestions((user) => {
    el.messageInput.value = '@' + user.tag + ' ';
    hideMentionSuggestions();
    el.messageInput.focus();
  }));
}

async function handleLogin(event) {
  event.preventDefault();
  clearLoginError();
  const name = el.nameInput.value.trim();
  const tag = el.tagInput.value.trim();
  try {
    state.user = await registerUser(name, tag);
    loadReadState();
    await setServerKeys(await loadServerKeys());
    saveRememberedUser(el.rememberInput.checked);
    showChat();
    showPublicChat();
    connect();
  } catch (error) {
    showLoginError(loginErrorText(error));
  }
}

async function handleSend(event) {
  event.preventDefault();
  const text = el.messageInput.value.trim();
  if (!text || !state.socket || state.socket.readyState !== WebSocket.OPEN) return;
  const encrypted = await encryptText(text);
  const outgoing = { scope: state.activeDialogId ? 'private' : 'public', to: state.activeDialogId || '', text: encrypted.text, keyDay: encrypted.keyDay };
  state.socket.send(JSON.stringify(outgoing));
  el.messageInput.value = '';
  hideMentionSuggestions();
  el.messageInput.focus();
}

function connect() {
  const protocol = location.protocol === 'https:' ? 'wss' : 'ws';
  state.socket = new WebSocket(protocol + '://' + location.host + '/ws?id=' + encodeURIComponent(state.user.id));
  state.socket.addEventListener('open', () => {
    el.statusEl.textContent = 'Онлайн · @' + state.user.tag;
    el.messageInput.focus();
  });
  state.socket.addEventListener('message', handleSocketMessage);
  state.socket.addEventListener('close', () => { el.statusEl.textContent = 'Отключено'; });
  state.socket.addEventListener('error', () => { el.statusEl.textContent = 'Ошибка соединения'; });
}

async function handleSocketMessage(event) {
  const msg = JSON.parse(event.data);
  if (msg.type === 'hello') {
    state.user = msg.user || state.user;
    return;
  }
  if (msg.type === 'users') {
    state.users = msg.users || [];
    syncDialogsWithUsers();
    renderPeople();
    return;
  }
  if (msg.type === 'history') {
    await loadHistory(msg.messages || []);
    return;
  }
  if (msg.type === 'message') msg.text = await decryptMessage(msg);
  storeMessage(msg, false);
}

async function loadHistory(messages) {
  state.publicMessages = [];
  state.dialogs = new Map();
  state.mentionAlert = false;
  state.publicUnreadCount = 0;
  for (const msg of messages) {
    if (msg.type === 'message') msg.text = await decryptMessage(msg);
    storeMessage(msg, true);
  }
  recalculateUnreadFromHistory();
  renderCurrentChat();
  renderPeople();
  updatePulse();
}

function storeMessage(msg, fromHistory) {
  if (msg.type === 'system' || !msg.private) {
    state.publicMessages.push(msg);
    if (!fromHistory && state.activeDialogId) {
      maybeMention(msg);
      if (msg.from !== state.user.id) {
        msg.unread = true;
        state.publicUnreadCount += 1;
      }
    }
    if (!state.activeDialogId) {
      markPublicRead();
      renderCurrentChat();
    }
    updatePulse();
    return;
  }

  const dialogId = msg.from === state.user.id ? msg.to : msg.from;
  const dialogUser = { id: dialogId, name: msg.from === state.user.id ? msg.toName : msg.name, tag: msg.from === state.user.id ? msg.toTag : msg.fromTag };
  const knownUser = state.users.find((user) => user.id === dialogId);
  const dialog = getDialog(dialogId, knownUser || dialogUser);
  dialog.messages.push(msg);

  if (state.activeDialogId === dialogId) {
    markDialogRead(dialogId);
    renderCurrentChat();
  } else if (!fromHistory && msg.from !== state.user.id) {
    msg.unread = true;
    dialog.unreadCount += 1;
  }
  renderPeople();
  updatePulse();
}

function recalculateUnreadFromHistory() {
  const lastPublic = state.readState.public || '';
  let afterPublicRead = false;
  state.publicUnreadCount = 0;
  for (const msg of state.publicMessages) {
    if (msg.id === lastPublic) afterPublicRead = true;
    msg.unread = Boolean(lastPublic && afterPublicRead && msg.id && msg.id !== lastPublic && msg.from !== state.user.id && msg.type !== 'system');
    if (msg.unread) {
      state.publicUnreadCount += 1;
      maybeMention(msg);
    }
  }

  for (const dialog of state.dialogs.values()) {
    const lastRead = state.readState.dialogs[dialog.id] || '';
    let afterDialogRead = false;
    dialog.unreadCount = 0;
    for (const msg of dialog.messages) {
      if (msg.id === lastRead) afterDialogRead = true;
      msg.unread = Boolean(lastRead && afterDialogRead && msg.id && msg.id !== lastRead && msg.from !== state.user.id);
      if (msg.unread) dialog.unreadCount += 1;
    }
  }
  if (!state.readState.public && state.publicMessages.length > 0) {
    markPublicRead();
  }
  saveReadState();
}

function syncDialogsWithUsers() {
  for (const user of state.users) {
    const dialog = state.dialogs.get(user.id);
    if (dialog) {
      dialog.name = user.name;
      dialog.tag = user.tag;
      dialog.online = user.online;
    }
  }
}

function maybeMention(msg) {
  if (!msg.text || msg.from === state.user.id || !state.user || !state.user.tag) return;
  const lowerText = msg.text.toLowerCase();
  const needle = '@' + state.user.tag.toLowerCase();
  if (lowerText.split(/\s+/).some((part) => part.replace(/[.,!?;:]$/, '') === needle)) state.mentionAlert = true;
}

function leaveChat() {
  if (state.socket) state.socket.close();
  resetChatState();
  resetCrypto();
  updatePulse();
  closeDrawer();
  closeSettings();
  showLogin();
}

function loginErrorText(error) {
  const message = String(error && error.message ? error.message : '');
  if (message.includes('занят') || message.includes('taken') || message.includes('already')) return 'Этот тег уже занят. Придумай другой.';
  if (message.includes('короче') || message.includes('least 3')) return 'Тег должен быть не короче 3 символов.';
  if (message.includes('Введите имя') || message.includes('name is required')) return 'Введите имя.';
  return message || 'Не удалось войти. Проверь имя и тег.';
}
