import { state, getDialog, markDialogRead, markPublicRead } from './state.js';

export const el = {
  login: document.getElementById('login'),
  chat: document.getElementById('chat'),
  chatTitle: document.getElementById('chatTitle'),
  loginForm: document.getElementById('loginForm'),
  messageForm: document.getElementById('messageForm'),
  nameInput: document.getElementById('nameInput'),
  tagInput: document.getElementById('tagInput'),
  rememberInput: document.getElementById('rememberInput'),
  loginError: document.getElementById('loginError'),
  messageInput: document.getElementById('messageInput'),
  messages: document.getElementById('messages'),
  statusEl: document.getElementById('status'),
  publicBtn: document.getElementById('publicBtn'),
  publicPulse: document.getElementById('publicPulse'),
  privateBtn: document.getElementById('privateBtn'),
  privatePulse: document.getElementById('privatePulse'),
  settingsBtn: document.getElementById('settingsBtn'),
  settingsMenu: document.getElementById('settingsMenu'),
  closeSettingsBtn: document.getElementById('closeSettingsBtn'),
  leaveBtn: document.getElementById('leaveBtn'),
  peopleDrawer: document.getElementById('peopleDrawer'),
  drawerBackdrop: document.getElementById('drawerBackdrop'),
  closeDrawerBtn: document.getElementById('closeDrawerBtn'),
  usersList: document.getElementById('usersList'),
  offlineUsersList: document.getElementById('offlineUsersList'),
  dialogsList: document.getElementById('dialogsList'),
  onlineCount: document.getElementById('onlineCount'),
  mentionSuggestions: document.getElementById('mentionSuggestions')
};

export function showLoginError(text) {
  el.loginError.textContent = text;
  el.loginError.classList.remove('hidden');
}

export function clearLoginError() {
  el.loginError.textContent = '';
  el.loginError.classList.add('hidden');
}

export function showChat() {
  el.login.classList.add('hidden');
  el.chat.classList.remove('hidden');
}

export function showLogin() {
  el.chat.classList.add('hidden');
  el.login.classList.remove('hidden');
}

export function openDrawer() {
  el.peopleDrawer.classList.add('open');
  el.peopleDrawer.setAttribute('aria-hidden', 'false');
  el.drawerBackdrop.classList.remove('hidden');
}

export function closeDrawer() {
  el.peopleDrawer.classList.remove('open');
  el.peopleDrawer.setAttribute('aria-hidden', 'true');
  if (el.settingsMenu.classList.contains('hidden')) el.drawerBackdrop.classList.add('hidden');
}

export function toggleSettings() {
  const willOpen = el.settingsMenu.classList.contains('hidden');
  el.settingsMenu.classList.toggle('hidden', !willOpen);
  el.settingsMenu.classList.toggle('open', willOpen);
  el.settingsMenu.setAttribute('aria-hidden', String(!willOpen));
  el.drawerBackdrop.classList.toggle('hidden', !willOpen && !el.peopleDrawer.classList.contains('open'));
}

export function closeSettings() {
  el.settingsMenu.classList.add('hidden');
  el.settingsMenu.classList.remove('open');
  el.settingsMenu.setAttribute('aria-hidden', 'true');
  if (!el.peopleDrawer.classList.contains('open')) el.drawerBackdrop.classList.add('hidden');
}

export function showPublicChat() {
  state.activeDialogId = null;
  markPublicRead();
  el.chatTitle.textContent = 'Общий чат';
  el.publicBtn.classList.add('active');
  el.privateBtn.classList.remove('active');
  el.messageInput.placeholder = 'Напиши сообщение...';
  renderCurrentChat();
  renderPeople();
  updatePulse();
  el.messageInput.focus();
}

export function openDialog(user) {
  const dialog = getDialog(user.id, user);
  state.activeDialogId = dialog.id;
  markDialogRead(dialog.id);
  el.chatTitle.textContent = `${dialog.name} @${dialog.tag}`;
  el.publicBtn.classList.remove('active');
  el.privateBtn.classList.add('active');
  el.messageInput.placeholder = `Сообщение для ${dialog.name}...`;
  closeDrawer();
  renderCurrentChat();
  renderPeople();
  updatePulse();
  el.messageInput.focus();
}

export function renderCurrentChat() {
  el.messages.innerHTML = '';
  const current = state.activeDialogId ? (state.dialogs.get(state.activeDialogId)?.messages || []) : state.publicMessages;
  let unreadDividerShown = false;
  current.forEach((msg) => {
    if (msg.unread && !unreadDividerShown) {
      const divider = document.createElement('div');
      divider.className = 'unread-divider';
      divider.textContent = 'Новые сообщения';
      el.messages.appendChild(divider);
      unreadDividerShown = true;
    }
    renderMessageElement(msg);
  });
  el.messages.scrollTop = el.messages.scrollHeight;
}

export function renderPeople() {
  el.usersList.innerHTML = '';
  el.offlineUsersList.innerHTML = '';
  el.dialogsList.innerHTML = '';

  const currentUserId = state.user && state.user.id;
  const online = state.users.filter((user) => user.online);
  const offline = state.users.filter((user) => !user.online);
  el.onlineCount.textContent = `${online.length} онлайн · ${offline.length} оффлайн`;

  renderUserGroup(el.usersList, online, currentUserId);
  renderUserGroup(el.offlineUsersList, offline, currentUserId);

  const dialogs = Array.from(state.dialogs.values()).sort((a, b) => {
    if (a.unreadCount !== b.unreadCount) return b.unreadCount - a.unreadCount;
    return a.name.localeCompare(b.name);
  });
  if (dialogs.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'empty-dialogs';
    empty.textContent = 'Пока нет личных диалогов';
    el.dialogsList.appendChild(empty);
  } else {
    dialogs.forEach((dialog) => el.dialogsList.appendChild(createUserRow(dialog, false, false)));
  }
}

function renderUserGroup(container, users, currentUserId) {
  if (users.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'empty-dialogs';
    empty.textContent = 'Никого нет';
    container.appendChild(empty);
    return;
  }
  users.forEach((user) => container.appendChild(createUserRow(user, user.id === currentUserId, true)));
}

export function renderMentionSuggestions(onPick) {
  const value = el.messageInput.value;
  if (state.activeDialogId || !value.startsWith('@')) {
    hideMentionSuggestions();
    return;
  }
  const query = value.slice(1).toLowerCase();
  const matches = state.users
    .filter((user) => user.id !== (state.user && state.user.id) && user.tag.toLowerCase().startsWith(query))
    .slice(0, 6);
  if (matches.length === 0) {
    hideMentionSuggestions();
    return;
  }
  el.mentionSuggestions.innerHTML = '';
  matches.forEach((user) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'mention-item';
    button.textContent = `${user.name} @${user.tag}${user.online ? '' : ' · оффлайн'}`;
    button.addEventListener('click', () => onPick(user));
    el.mentionSuggestions.appendChild(button);
  });
  el.mentionSuggestions.classList.remove('hidden');
}

export function hideMentionSuggestions() {
  el.mentionSuggestions.classList.add('hidden');
  el.mentionSuggestions.innerHTML = '';
}

export function updatePulse() {
  const hasPrivateUnread = Array.from(state.dialogs.values()).some((dialog) => dialog.unreadCount > 0);
  el.privatePulse.classList.toggle('hidden', !hasPrivateUnread);
  el.publicPulse.classList.toggle('hidden', !state.mentionAlert && state.publicUnreadCount === 0);
  document.title = hasPrivateUnread || state.mentionAlert || state.publicUnreadCount > 0 ? '• Соцсети-ВСЁ!' : 'Соцсети-ВСЁ!';
}

function createUserRow(user, isSelf, fromParticipants) {
  const row = document.createElement('div');
  row.className = 'user-row';
  if (isSelf) row.classList.add('self');
  if (user.id === state.activeDialogId) row.classList.add('active');
  if (!user.online) row.classList.add('offline');
  if (user.unreadCount > 0) row.classList.add('has-unread');

  const info = document.createElement('div');
  info.className = 'user-info';
  if (user.unreadCount > 0) {
    const badge = document.createElement('span');
    badge.className = 'unread-badge';
    info.appendChild(badge);
  }
  const dot = document.createElement('span');
  dot.className = 'user-dot';
  if (!user.online) dot.classList.add('offline-dot');
  if (!fromParticipants) dot.classList.add('dialog-dot');
  const name = document.createElement('span');
  name.className = 'user-name';
  name.textContent = isSelf ? `${user.name} @${user.tag} · это ты` : `${user.name} @${user.tag}`;
  info.appendChild(dot);
  info.appendChild(name);
  row.appendChild(info);

  if (!isSelf) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'user-message-btn';
    button.textContent = user.id === state.activeDialogId ? 'Открыто' : 'Написать';
    button.addEventListener('click', () => openDialog(user));
    row.appendChild(button);
  }
  return row;
}

function renderMessageElement(msg) {
  const item = document.createElement('div');
  if (msg.type === 'system') {
    item.className = 'message system';
    item.textContent = `${msg.time} · ${msg.text}`;
    el.messages.appendChild(item);
    return;
  }
  item.className = msg.private ? 'message private-message' : 'message';
  if (msg.unread) item.classList.add('unread-message');
  const meta = document.createElement('div');
  meta.className = 'meta';
  meta.textContent = msg.private
    ? (msg.from === (state.user && state.user.id) ? `Вы · ${msg.time}` : `${msg.name} @${msg.fromTag} · ${msg.time}`)
    : `${msg.name} @${msg.fromTag || ''} · ${msg.time}`;
  const text = document.createElement('div');
  text.className = 'text';
  text.textContent = msg.text;
  item.appendChild(meta);
  item.appendChild(text);
  el.messages.appendChild(item);
}
