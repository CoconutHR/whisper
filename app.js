const MAX_NICKNAME_LENGTH = 7;
const RESERVED_NICKNAME = "coco";
const HTTP_PROTOCOLS = new Set(["http:", "https:"]);
const FRIEND_MESSAGE_COLORS = new Set(["default", "green", "blue", "cyan", "amber", "rose"]);
const STORAGE_KEY_MIGRATIONS = [
  ["tmpchat-profile-name", "whisper-profile-name"],
  ["tmpchat-profile-signature", "whisper-profile-signature"],
  ["tmpchat-friend-colors", "whisper-friend-colors"],
  ["tmpchat-pin-sidebar", "whisper-pin-sidebar"],
  ["tmpchat-show-message-time", "whisper-show-message-time"],
  ["tmpchat-parse-latex", "whisper-parse-latex"],
  ["tmpchat-theme", "whisper-theme"]
];

STORAGE_KEY_MIGRATIONS.forEach(([legacyKey, currentKey]) => {
  if (localStorage.getItem(currentKey) === null) {
    const legacyValue = localStorage.getItem(legacyKey);
    if (legacyValue !== null) localStorage.setItem(currentKey, legacyValue);
  }
});

let backendEnabled = false;
let backendAuthenticated = false;
let authMode = "login";
let socket = null;
let reconnectTimer = null;

function seedTimestamp(daysAgo, hour, minute, second, millisecond) {
  const date = new Date();
  date.setHours(hour, minute, second, millisecond);
  date.setDate(date.getDate() - daysAgo);
  return date.toISOString();
}

function previousYearTimestamp(month, day, hour, minute, second, millisecond) {
  const now = new Date();
  return new Date(
    now.getFullYear() - 1,
    month - 1,
    day,
    hour,
    minute,
    second,
    millisecond
  ).toISOString();
}

const members = [
  { name: "coco", online: true, reserved: true, signature: "仅自己可见" },
  { name: "小林", online: true, signature: "项目主页：[查看](https://example.com/docs)" },
  { name: "小周", online: true, signature: "今天专心写代码 😊" },
  { name: "小陈", online: false, signature: "稍后回来" },
  { name: "小许", online: false, signature: "" }
];

function nicknameValidationError(value, checkMembers = true) {
  const name = value.trim();
  if (!name) return "个人名称不能为空";
  if (Array.from(name).length > MAX_NICKNAME_LENGTH) {
    return `个人名称不能超过${MAX_NICKNAME_LENGTH}个字符`;
  }
  if (name.toLocaleLowerCase("en-US") === RESERVED_NICKNAME) {
    return "coco 是系统保留名称";
  }
  if (checkMembers && members.some((member) => member.name === name)) return "该名称已被使用";
  return "";
}

const storedProfileName = localStorage.getItem("whisper-profile-name") || "你";
const initialProfileName = nicknameValidationError(storedProfileName) ? "你" : storedProfileName;
const initialSignature = localStorage.getItem("whisper-profile-signature")
  || "保持联系 😊 [个人主页](https://example.com)";

function loadStoredFriendColors() {
  try {
    const stored = JSON.parse(localStorage.getItem("whisper-friend-colors") || "{}");
    return new Map(
      Object.entries(stored).filter(([name, color]) => (
        name && color !== "default" && FRIEND_MESSAGE_COLORS.has(color)
      ))
    );
  } catch (_) {
    return new Map();
  }
}

const state = {
  currentUser: initialProfileName,
  currentConversation: "group",
  friendMenuMember: null,
  friends: new Set([RESERVED_NICKNAME]),
  friendColors: loadStoredFriendColors(),
  profile: {
    signature: initialSignature,
    password: ""
  },
  conversations: {
    group: [
      { from: "*", text: "已连接，4 人在线。", sentAt: seedTimestamp(0, 15, 54, 1, 120), system: true },
      { from: "小林", text: "下午四点开始？", sentAt: seedTimestamp(1, 15, 56, 12, 345) },
      { from: initialProfileName, text: "可以，群里确认一下。", sentAt: seedTimestamp(2, 15, 57, 23, 456) },
      { from: "小陈", text: "我晚点上线。", sentAt: seedTimestamp(7, 15, 58, 34, 567) }
    ],
    "dm:coco": [
      { from: "coco", text: "这里的消息仅你自己可见。", sentAt: seedTimestamp(0, 14, 20, 8, 210) }
    ],
    "dm:小林": [
      { from: "小林", text: "会议链接我稍后发你。", sentAt: previousYearTimestamp(12, 31, 15, 40, 45, 678) },
      { from: initialProfileName, text: "好的。", sentAt: seedTimestamp(3, 15, 41, 56, 789) }
    ],
    "dm:小周": [
      { from: "小周", text: "文件我已经看过了。", sentAt: seedTimestamp(0, 15, 32, 7, 890) }
    ],
    "dm:小陈": [],
    "dm:小许": []
  }
};

function persistDemoFriendColors() {
  localStorage.setItem(
    "whisper-friend-colors",
    JSON.stringify(Object.fromEntries(state.friendColors))
  );
}

const messagesEl = document.querySelector("#messages");
const appMainEl = document.querySelector("#app-main");
const authPanelEl = document.querySelector("#auth-panel");
const authFormEl = document.querySelector("#auth-form");
const authNameEl = document.querySelector("#auth-name");
const authPasswordEl = document.querySelector("#auth-password");
const authStatusEl = document.querySelector("#auth-status");
const authSubmitEl = document.querySelector("#auth-submit");
const authLoginModeEl = document.querySelector("#auth-login-mode");
const authRegisterModeEl = document.querySelector("#auth-register-mode");
const profilePanelEl = document.querySelector("#profile-panel");
const footerEl = document.querySelector("#footer");
const conversationTitleEl = document.querySelector("#conversation-title");
const conversationStatusEl = document.querySelector("#conversation-status");
const profileSignatureEl = document.querySelector("#profile-signature");
const backToGroupEl = document.querySelector("#back-to-group");
const groupConversationEl = document.querySelector("#group-conversation");
const groupCountEl = document.querySelector("#group-count");
const formEl = document.querySelector("#chatform");
const inputEl = document.querySelector("#chatinput");
const sidebarEl = document.querySelector("#sidebar");
const sidebarContentEl = document.querySelector("#sidebar-content");
const sidebarToggleEl = document.querySelector("#sidebar-toggle");
const pinSidebarEl = document.querySelector("#pin-sidebar");
const friendsEl = document.querySelector("#friends");
const friendsEmptyEl = document.querySelector("#friends-empty");
const usersEl = document.querySelector("#users");
const friendMenuEl = document.querySelector("#friend-menu");
const friendColorButtons = document.querySelectorAll("[data-friend-color]");
const friendColorStatusEl = document.querySelector("#friend-color-status");
const clearFriendMessagesEl = document.querySelector("#clear-friend-messages");
const deleteFriendEl = document.querySelector("#delete-friend");

const profileFormEl = document.querySelector("#profile-form");
const profileNameEl = document.querySelector("#profile-name");
const profileSignatureInputEl = document.querySelector("#profile-signature-input");
const profileSaveStatusEl = document.querySelector("#profile-save-status");
const passwordFormEl = document.querySelector("#password-form");
const currentPasswordEl = document.querySelector("#current-password");
const newPasswordEl = document.querySelector("#new-password");
const confirmPasswordEl = document.querySelector("#confirm-password");
const passwordSaveStatusEl = document.querySelector("#password-save-status");
const showMessageTimeEl = document.querySelector("#show-message-time");
const parseLatexEl = document.querySelector("#parse-latex");
const clearMessagesEl = document.querySelector("#clear-messages");
const schemeSelectorEl = document.querySelector("#scheme-selector");
const logoutEl = document.querySelector("#logout");

function conversationIdFor(name) {
  return `dm:${name}`;
}

function memberByName(name) {
  return members.find((member) => member.name === name) || null;
}

function sortedMembers(memberList) {
  return [...memberList].sort((left, right) => {
    if (left.reserved !== right.reserved) return left.reserved ? -1 : 1;
    if (left.online !== right.online) return left.online ? -1 : 1;
    return left.name.localeCompare(right.name, "zh-CN");
  });
}

function currentMember() {
  if (state.currentConversation === "group" || state.currentConversation === "self") {
    return null;
  }
  return memberByName(state.currentConversation.slice(3));
}

function currentMessages() {
  return state.conversations[state.currentConversation] || [];
}

function pad(value, length = 2) {
  return String(value).padStart(length, "0");
}

function calendarDayDifference(messageDate, currentDate) {
  const messageDay = Date.UTC(
    messageDate.getFullYear(),
    messageDate.getMonth(),
    messageDate.getDate()
  );
  const currentDay = Date.UTC(
    currentDate.getFullYear(),
    currentDate.getMonth(),
    currentDate.getDate()
  );
  return Math.round((currentDay - messageDay) / 86400000);
}

function formatClock(date) {
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

function messageTimeParts(sentAt, currentDate = new Date()) {
  const date = new Date(sentAt);
  const dayDifference = calendarDayDifference(date, currentDate);
  const clock = formatClock(date);

  if (dayDifference === 0) return { clock, context: "" };
  if (dayDifference === 1) return { clock, context: "昨天" };
  if (dayDifference === 2) return { clock, context: "前天" };

  const monthAndDay = `${pad(date.getMonth() + 1)}月${pad(date.getDate())}日`;
  if (date.getFullYear() === currentDate.getFullYear()) {
    return { clock, context: monthAndDay };
  }
  return { clock, context: `${date.getFullYear()}年${monthAndDay}` };
}

function formatFullMessageTime(sentAt) {
  const date = new Date(sentAt);
  return `${date.getFullYear()}年${pad(date.getMonth() + 1)}月${pad(date.getDate())}日 ${formatClock(date)}.${pad(date.getMilliseconds(), 3)}`;
}

function createExternalLink(label, url) {
  const link = document.createElement("a");
  link.href = url;
  link.textContent = label;
  link.target = "_blank";
  link.rel = "noopener noreferrer";
  return link;
}

function renderSignature(container, value) {
  container.replaceChildren();
  const pattern = /\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)|(https?:\/\/[^\s<]+)/g;
  let cursor = 0;
  let match;

  while ((match = pattern.exec(value)) !== null) {
    container.append(document.createTextNode(value.slice(cursor, match.index)));
    if (match[2]) {
      container.append(createExternalLink(match[1], match[2]));
    } else {
      let url = match[3];
      let trailing = "";
      while (/[.,!?，。！？]$/.test(url)) {
        trailing = url.slice(-1) + trailing;
        url = url.slice(0, -1);
      }
      container.append(createExternalLink(url, url));
      if (trailing) container.append(document.createTextNode(trailing));
    }
    cursor = pattern.lastIndex;
  }
  container.append(document.createTextNode(value.slice(cursor)));
}

function setConversationMeta(status, signature = "") {
  conversationStatusEl.textContent = status;
  profileSignatureEl.hidden = !signature.trim();
  if (signature.trim()) renderSignature(profileSignatureEl, signature);
  else profileSignatureEl.replaceChildren();
}

function appendFormattedText(container, value) {
  if (!parseLatexEl.checked) {
    container.textContent = value;
    return;
  }

  const parts = value.split(/(\$[^$\n]+\$)/g);
  parts.forEach((part) => {
    if (part.startsWith("$") && part.endsWith("$") && part.length > 2) {
      const math = document.createElement("span");
      math.className = "math";
      math.textContent = part.slice(1, -1);
      container.append(math);
      return;
    }
    container.append(document.createTextNode(part));
  });
}

function switchConversation(conversationId) {
  if (conversationId !== "self" && !state.conversations[conversationId]) return;

  state.currentConversation = conversationId;
  inputEl.value = "";
  resizeInput();
  hideFriendMenu();
  renderConversation();

  if (!pinSidebarEl.checked) setSidebar(false);
  if (conversationId === "self") {
    window.scrollTo({ top: 0 });
    profileNameEl.focus();
  } else {
    inputEl.focus();
    window.scrollTo({ top: document.body.scrollHeight });
  }
}

function mentionInGroup(name) {
  if (state.currentConversation !== "group") return;
  inputEl.value = `@${name} `;
  resizeInput();
  inputEl.focus();
  inputEl.setSelectionRange(inputEl.value.length, inputEl.value.length);
}

function renderConversationHeader() {
  const onlineCount = members.filter((member) => member.online).length + 1;

  if (state.currentConversation === "self") {
    document.body.dataset.conversationKind = "profile";
    conversationTitleEl.textContent = `@ ${state.currentUser}`;
    setConversationMeta("在线", state.profile.signature);
    backToGroupEl.hidden = false;
    return;
  }

  const member = currentMember();
  if (!member) {
    document.body.dataset.conversationKind = "group";
    conversationTitleEl.textContent = "# 群聊";
    setConversationMeta(`${onlineCount} 人在线`);
    inputEl.placeholder = "发送到群聊...";
    backToGroupEl.hidden = true;
    return;
  }

  document.body.dataset.conversationKind = "private";
  conversationTitleEl.textContent = `@ ${member.name}`;
  setConversationMeta(member.online ? "在线" : "离线，可留言", member.signature || "");
  inputEl.placeholder = member.online
    ? `发送给${member.name}...`
    : `给${member.name}留言...`;
  backToGroupEl.hidden = false;
}

function createMessageMeta(message, canMention) {
  const meta = document.createElement("span");
  meta.className = "message-meta";

  if (showMessageTimeEl.checked) {
    const time = document.createElement("time");
    const fullTime = formatFullMessageTime(message.sentAt);
    const { clock, context } = messageTimeParts(message.sentAt);
    time.className = "message-time";
    time.dateTime = message.sentAt;
    time.title = fullTime;
    time.setAttribute("aria-label", fullTime);

    const clockEl = document.createElement("span");
    clockEl.className = "message-clock";
    clockEl.textContent = clock;
    time.append(clockEl);

    if (context) {
      const contextEl = document.createElement("span");
      contextEl.className = "message-date-context";
      contextEl.textContent = context;
      time.append(contextEl);
    }
    meta.append(time);
  }

  const nickName = document.createElement(canMention ? "a" : "span");
  nickName.className = "nick-name";
  nickName.textContent = message.from;

  if (message.from === state.currentUser) nickName.classList.add("me");
  if (message.system) nickName.classList.add("system");

  if (canMention) {
    nickName.href = "#chatinput";
    nickName.title = `在群聊中提及${message.from}`;
    nickName.addEventListener("click", (event) => {
      event.preventDefault();
      mentionInGroup(message.from);
    });
  } else {
    nickName.title = message.from;
  }

  meta.append(nickName);
  return meta;
}

function renderMessages() {
  messagesEl.replaceChildren();
  const conversationMessages = currentMessages();

  if (conversationMessages.length === 0) {
    const row = document.createElement("div");
    row.className = "message";
    row.append(
      createMessageMeta({ from: "*", sentAt: new Date().toISOString(), system: true }, false)
    );

    const text = document.createElement("p");
    text.className = "text";
    text.textContent = "还没有消息。";
    row.append(text);
    messagesEl.append(row);
    return;
  }

  conversationMessages.forEach((message) => {
    const canMention = !message.system
      && message.from !== state.currentUser
      && state.currentConversation === "group";
    const row = document.createElement("div");
    row.className = "message";

    const text = document.createElement("p");
    text.className = "text";
    const friendColor = state.friendColors.get(message.from);
    if (!message.system && message.from !== state.currentUser && friendColor) {
      text.dataset.friendColor = friendColor;
    }
    appendFormattedText(text, message.text);

    if (message.delivery === "queued") {
      const delivery = document.createElement("span");
      delivery.className = "message-delivery";
      delivery.textContent = "待送达";
      text.append(delivery);
    }

    row.append(createMessageMeta(message, canMention), text);
    messagesEl.append(row);
  });
}

function renderProfilePanel() {
  profileNameEl.value = state.currentUser;
  profileSignatureInputEl.value = state.profile.signature;
}

function renderFriends() {
  friendsEl.replaceChildren();
  const friendMembers = sortedMembers(
    members.filter((member) => state.friends.has(member.name))
  );
  friendsEmptyEl.hidden = friendMembers.length > 0;

  friendMembers.forEach((member) => {
    const conversationId = conversationIdFor(member.name);
    const item = document.createElement("li");
    item.className = "friend-row";
    item.classList.toggle("offline", !member.online);
    item.classList.toggle("active", state.currentConversation === conversationId);
    item.classList.toggle("reserved", Boolean(member.reserved));

    const nameButton = document.createElement("button");
    nameButton.type = "button";
    nameButton.className = "friend-name";
    nameButton.textContent = member.reserved ? `@${member.name}` : member.name;
    nameButton.setAttribute("aria-label", `与${member.name}私聊（好友）`);
    nameButton.title = member.online ? `与${member.name}私聊` : `给${member.name}留言`;
    nameButton.addEventListener("click", () => switchConversation(conversationId));

    const moreButton = document.createElement("button");
    moreButton.type = "button";
    moreButton.className = "friend-more";
    moreButton.textContent = "⋮";
    moreButton.setAttribute("aria-label", `${member.name}的更多操作`);
    moreButton.title = "更多操作";
    moreButton.addEventListener("click", (event) => {
      event.stopPropagation();
      const rect = moreButton.getBoundingClientRect();
      showFriendMenu(member.name, rect.right, rect.bottom + 4);
    });

    item.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      event.stopPropagation();
      showFriendMenu(member.name, event.clientX, event.clientY);
    });

    item.append(nameButton, moreButton);
    friendsEl.append(item);
  });
}

function renderUsers() {
  usersEl.replaceChildren();
  const allUsers = [
    { name: state.currentUser, online: true, self: true },
    ...sortedMembers(members)
  ];

  allUsers.forEach((member) => {
    const item = document.createElement("li");
    if (!member.online) item.classList.add("offline");
    if (member.reserved) item.classList.add("reserved");

    const button = document.createElement("button");
    button.type = "button";
    button.textContent = member.self
      ? `${member.name}（本人）`
      : member.reserved ? `@${member.name}` : member.name;

    if (member.self) {
      const active = state.currentConversation === "self";
      button.classList.toggle("active", active);
      button.setAttribute("aria-current", active ? "page" : "false");
      button.setAttribute("aria-label", "打开个人设置");
      button.title = "个人设置";
      button.addEventListener("click", () => switchConversation("self"));
    } else {
      const conversationId = conversationIdFor(member.name);
      const active = state.currentConversation === conversationId;
      button.classList.toggle("active", active);
      button.setAttribute("aria-current", active ? "page" : "false");
      button.setAttribute("aria-label", `与${member.name}私聊（成员）`);
      button.title = member.online ? `与${member.name}私聊` : `给${member.name}留言`;
      button.addEventListener("click", () => switchConversation(conversationId));
    }

    item.append(button);
    usersEl.append(item);
  });
}

function renderConversationNavigation() {
  const groupActive = state.currentConversation === "group";
  groupConversationEl.classList.toggle("active", groupActive);
  groupConversationEl.setAttribute("aria-current", groupActive ? "page" : "false");
  groupCountEl.textContent = String(members.filter((member) => member.online).length + 1);
  renderFriends();
  renderUsers();
}

function renderConversation() {
  const profileMode = state.currentConversation === "self";
  messagesEl.hidden = profileMode;
  profilePanelEl.hidden = !profileMode;
  footerEl.hidden = profileMode;
  renderConversationHeader();
  if (profileMode) renderProfilePanel();
  else renderMessages();
  renderConversationNavigation();
}

function sendCurrentMessage() {
  if (state.currentConversation === "self") return;
  const text = inputEl.value.trim();
  if (!text) return;

  const member = currentMember();
  if (backendEnabled) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      inputEl.placeholder = "正在重新连接...";
      return;
    }
    socket.send(JSON.stringify({
      type: "message",
      scope: member ? "private" : "group",
      to: member?.name || "",
      text
    }));
    inputEl.value = "";
    resizeInput();
    return;
  }

  currentMessages().push({
    from: state.currentUser,
    text,
    sentAt: new Date().toISOString(),
    delivery: member && !member.online ? "queued" : "sent"
  });

  if (member) state.friends.add(member.name);
  inputEl.value = "";
  resizeInput();
  renderMessages();
  renderConversationNavigation();
  window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" });
}

function resizeInput() {
  inputEl.style.height = "auto";
  inputEl.style.height = `${Math.min(inputEl.scrollHeight, 144)}px`;
}

function setSidebar(open) {
  sidebarEl.classList.toggle("open", open);
  sidebarContentEl.hidden = !open;
  sidebarToggleEl.setAttribute("aria-expanded", String(open));
  sidebarToggleEl.setAttribute("aria-label", open ? "关闭会话列表" : "打开会话列表");
  if (!open) hideFriendMenu();
}

function updateFriendColorMenu(name) {
  const selectedColor = state.friendColors.get(name) || "default";
  friendColorButtons.forEach((button) => {
    const selected = button.dataset.friendColor === selectedColor;
    button.setAttribute("aria-checked", String(selected));
  });
}

function showFriendMenu(name, x, y) {
  const member = memberByName(name);
  if (!member) return;

  state.friendMenuMember = name;
  deleteFriendEl.hidden = Boolean(member.reserved);
  updateFriendColorMenu(name);
  setFormStatus(friendColorStatusEl, "");
  friendMenuEl.hidden = false;

  const menuRect = friendMenuEl.getBoundingClientRect();
  const left = Math.max(8, Math.min(x, window.innerWidth - menuRect.width - 8));
  const top = Math.max(8, Math.min(y, window.innerHeight - menuRect.height - 8));
  friendMenuEl.style.left = `${left}px`;
  friendMenuEl.style.top = `${top}px`;
  const selectedButton = Array.from(friendColorButtons).find(
    (button) => button.getAttribute("aria-checked") === "true"
  );
  selectedButton?.focus();
}

function hideFriendMenu() {
  friendMenuEl.hidden = true;
  state.friendMenuMember = null;
  setFormStatus(friendColorStatusEl, "");
}

function setFormStatus(element, message, error = false) {
  element.textContent = message;
  element.classList.toggle("error", error);
}

function updateOwnMessageNames(previousName, nextName) {
  Object.values(state.conversations).forEach((messages) => {
    messages.forEach((message) => {
      if (message.from === previousName) message.from = nextName;
    });
  });
}

function updateMemberPresence(name, online) {
  const member = memberByName(name);
  if (!member || member.reserved) return;

  member.online = online;
  if (online) {
    state.conversations[conversationIdFor(name)].forEach((message) => {
      if (message.delivery === "queued") message.delivery = "sent";
    });
  }
  renderConversation();
}

function receivePrivateMessage(from, text) {
  const member = memberByName(from);
  if (!member) return;

  state.conversations[conversationIdFor(from)].push({
    from,
    text,
    sentAt: new Date().toISOString(),
    delivery: "sent"
  });
  state.friends.add(from);
  renderConversationNavigation();
  if (state.currentConversation === conversationIdFor(from)) renderMessages();
}

class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.status = status;
  }
}

async function apiRequest(path, options = {}) {
  const request = { ...options, headers: { ...(options.headers || {}) } };
  if (request.body && typeof request.body !== "string") {
    request.headers["Content-Type"] = "application/json";
    request.body = JSON.stringify(request.body);
  }
  const response = await fetch(path, request);
  if (!response.ok) {
    let message = `请求失败（${response.status}）`;
    try {
      const payload = await response.json();
      if (payload.error) message = payload.error;
    } catch (_) {}
    throw new ApiError(message, response.status);
  }
  if (response.status === 204) return null;
  return response.json();
}

function closeSocket() {
  window.clearTimeout(reconnectTimer);
  reconnectTimer = null;
  if (socket) {
    socket.onclose = null;
    socket.close();
    socket = null;
  }
}

function showAuthUI(message = "") {
  backendAuthenticated = false;
  closeSocket();
  authPanelEl.hidden = false;
  appMainEl.hidden = true;
  footerEl.hidden = true;
  sidebarEl.hidden = true;
  logoutEl.hidden = true;
  setFormStatus(authStatusEl, message, Boolean(message));
  authNameEl.focus();
}

function showApplicationUI() {
  backendAuthenticated = true;
  authPanelEl.hidden = true;
  appMainEl.hidden = false;
  sidebarEl.hidden = false;
  logoutEl.hidden = !backendEnabled;
  renderConversation();
}

function setAuthMode(mode) {
  authMode = mode;
  const login = mode === "login";
  authLoginModeEl.classList.toggle("active", login);
  authRegisterModeEl.classList.toggle("active", !login);
  authLoginModeEl.setAttribute("aria-selected", String(login));
  authRegisterModeEl.setAttribute("aria-selected", String(!login));
  authSubmitEl.textContent = login ? "登录" : "注册";
  authPasswordEl.autocomplete = login ? "current-password" : "new-password";
  setFormStatus(authStatusEl, "");
}

function hydrateBootstrap(payload) {
  state.currentUser = payload.self.name;
  state.profile.signature = payload.self.signature || "";
  state.currentConversation = "group";
  state.friendMenuMember = null;
  state.friends = new Set(payload.friends || []);
  state.friendColors = new Map(Object.entries(payload.friendColors || {}));
  state.conversations = payload.conversations || { group: [], "dm:coco": [] };

  members.splice(0, members.length, ...(payload.members || []));
  members.forEach((member) => {
    const key = conversationIdFor(member.name);
    if (!state.conversations[key]) state.conversations[key] = [];
  });
  if (!state.conversations.group) state.conversations.group = [];

  showMessageTimeEl.checked = payload.self.settings.showMessageTime;
  parseLatexEl.checked = payload.self.settings.parseLatex;
  schemeSelectorEl.value = payload.self.settings.theme || "dune";
  document.body.dataset.theme = schemeSelectorEl.value;
  profileNameEl.value = state.currentUser;
  profileSignatureInputEl.value = state.profile.signature;
}

function renameRemoteMember(previousName, memberView) {
  const member = memberByName(previousName);
  if (!member) return;
  const previousKey = conversationIdFor(previousName);
  const nextKey = conversationIdFor(memberView.name);
  member.name = memberView.name;
  member.online = memberView.online;
  member.signature = memberView.signature || "";
  if (previousKey !== nextKey) {
    state.conversations[nextKey] = state.conversations[previousKey] || [];
    delete state.conversations[previousKey];
    if (state.friends.delete(previousName)) state.friends.add(memberView.name);
    if (state.friendColors.has(previousName)) {
      state.friendColors.set(memberView.name, state.friendColors.get(previousName));
      state.friendColors.delete(previousName);
    }
    if (state.currentConversation === previousKey) state.currentConversation = nextKey;
    Object.values(state.conversations).forEach((messages) => {
      messages.forEach((message) => {
        if (message.from === previousName) message.from = memberView.name;
      });
    });
  }
}

function handleSocketEvent(event) {
  if (event.type === "message") {
    if (!state.conversations[event.conversation]) state.conversations[event.conversation] = [];
    const messages = state.conversations[event.conversation];
    if (!messages.some((message) => message.id === event.message.id)) messages.push(event.message);
    if (event.friend) state.friends.add(event.friend);
    if (state.currentConversation === event.conversation) renderMessages();
    renderConversationNavigation();
    return;
  }

  if (event.type === "delivered") {
    Object.values(state.conversations).forEach((messages) => {
      const message = messages.find((candidate) => candidate.id === event.messageId);
      if (message) message.delivery = "sent";
    });
    renderMessages();
    return;
  }

  if (event.type === "presence") {
    const member = memberByName(event.name);
    if (member) {
      member.online = event.online;
      member.signature = event.signature || member.signature || "";
      renderConversation();
    }
    return;
  }

  if (event.type === "profile") {
    renameRemoteMember(event.previousName, event.member);
    renderConversation();
  }
}

function connectSocket() {
  closeSocket();
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  socket = new WebSocket(`${protocol}//${location.host}/ws`);
  socket.onmessage = (message) => {
    try {
      handleSocketEvent(JSON.parse(message.data));
    } catch (_) {}
  };
  socket.onclose = () => {
    socket = null;
    if (!backendAuthenticated) return;
    reconnectTimer = window.setTimeout(async () => {
      try {
        const payload = await apiRequest("/api/bootstrap");
        hydrateBootstrap(payload);
        renderConversation();
        connectSocket();
      } catch (error) {
        if (error.status === 401) showAuthUI("登录已失效");
        else connectSocket();
      }
    }, 2000);
  };
}

async function startApplication() {
  if (!HTTP_PROTOCOLS.has(location.protocol)) {
    logoutEl.hidden = true;
    renderConversation();
    resizeInput();
    return;
  }

  try {
    appMainEl.hidden = true;
    footerEl.hidden = true;
    sidebarEl.hidden = true;
    const health = await fetch("/healthz", { cache: "no-store" });
    if (!health.ok || !health.headers.get("content-type")?.includes("application/json")) {
      throw new Error("not backend");
    }
    backendEnabled = true;
  } catch (_) {
    appMainEl.hidden = false;
    sidebarEl.hidden = false;
    logoutEl.hidden = true;
    renderConversation();
    resizeInput();
    return;
  }

  try {
    const payload = await apiRequest("/api/bootstrap");
    hydrateBootstrap(payload);
    showApplicationUI();
    connectSocket();
  } catch (error) {
    if (error.status === 401) showAuthUI();
    else showAuthUI(error.message);
  }
}

authLoginModeEl.addEventListener("click", () => setAuthMode("login"));
authRegisterModeEl.addEventListener("click", () => setAuthMode("register"));

authFormEl.addEventListener("submit", async (event) => {
  event.preventDefault();
  const name = authNameEl.value.trim();
  const password = authPasswordEl.value;
  const validation = nicknameValidationError(name, false);
  if (validation) {
    setFormStatus(authStatusEl, validation, true);
    return;
  }

  authSubmitEl.disabled = true;
  try {
    await apiRequest(authMode === "login" ? "/api/login" : "/api/register", {
      method: "POST",
      body: { name, password }
    });
    const payload = await apiRequest("/api/bootstrap");
    hydrateBootstrap(payload);
    authFormEl.reset();
    showApplicationUI();
    connectSocket();
  } catch (error) {
    setFormStatus(authStatusEl, error.message, true);
  } finally {
    authSubmitEl.disabled = false;
  }
});

formEl.addEventListener("submit", (event) => {
  event.preventDefault();
  sendCurrentMessage();
  inputEl.focus();
});

inputEl.addEventListener("keydown", (event) => {
  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault();
    sendCurrentMessage();
  }
});

inputEl.addEventListener("input", resizeInput);
backToGroupEl.addEventListener("click", () => switchConversation("group"));
groupConversationEl.addEventListener("click", () => switchConversation("group"));

sidebarToggleEl.addEventListener("click", () => {
  setSidebar(!sidebarEl.classList.contains("open"));
});

sidebarEl.addEventListener("mouseenter", () => {
  if (window.matchMedia("(min-width: 601px)").matches) setSidebar(true);
});

sidebarEl.addEventListener("mouseleave", () => {
  if (!pinSidebarEl.checked && window.matchMedia("(min-width: 601px)").matches) {
    setSidebar(false);
  }
});

pinSidebarEl.addEventListener("change", () => {
  localStorage.setItem("whisper-pin-sidebar", String(pinSidebarEl.checked));
  setSidebar(pinSidebarEl.checked);
});

profileFormEl.addEventListener("submit", async (event) => {
  event.preventDefault();
  const nextName = profileNameEl.value.trim();
  const error = nicknameValidationError(nextName);
  if (error) {
    setFormStatus(profileSaveStatusEl, error, true);
    return;
  }

  const signature = profileSignatureInputEl.value.trim();
  try {
    const result = backendEnabled
      ? await apiRequest("/api/profile", { method: "PATCH", body: { name: nextName, signature } })
      : { name: nextName, signature };
    const previousName = state.currentUser;
    state.currentUser = result.name;
    state.profile.signature = result.signature || "";
    updateOwnMessageNames(previousName, state.currentUser);
    if (!backendEnabled) {
      localStorage.setItem("whisper-profile-name", state.currentUser);
      localStorage.setItem("whisper-profile-signature", state.profile.signature);
    }
    setFormStatus(profileSaveStatusEl, "已保存");
    renderConversationHeader();
    renderConversationNavigation();
  } catch (requestError) {
    setFormStatus(profileSaveStatusEl, requestError.message, true);
  }
});

passwordFormEl.addEventListener("submit", async (event) => {
  event.preventDefault();
  const currentPassword = currentPasswordEl.value;
  const nextPassword = newPasswordEl.value;
  const confirmation = confirmPasswordEl.value;

  if (!currentPassword) {
    setFormStatus(passwordSaveStatusEl, "请输入当前密码", true);
    return;
  }
  if (!nextPassword) {
    setFormStatus(passwordSaveStatusEl, "新密码不能为空", true);
    return;
  }
  if (nextPassword !== confirmation) {
    setFormStatus(passwordSaveStatusEl, "两次输入的新密码不一致", true);
    return;
  }

  try {
    if (backendEnabled) {
      await apiRequest("/api/password", {
        method: "PATCH",
        body: { currentPassword, newPassword: nextPassword }
      });
    } else {
      state.profile.password = nextPassword;
    }
    passwordFormEl.reset();
    setFormStatus(passwordSaveStatusEl, "密码已更新");
  } catch (requestError) {
    setFormStatus(passwordSaveStatusEl, requestError.message, true);
  }
});

async function persistSettings() {
  if (backendEnabled) {
    try {
      await apiRequest("/api/settings", {
        method: "PATCH",
        body: {
          showMessageTime: showMessageTimeEl.checked,
          parseLatex: parseLatexEl.checked,
          theme: schemeSelectorEl.value
        }
      });
    } catch (error) {
      setFormStatus(profileSaveStatusEl, error.message, true);
    }
    return;
  }
  localStorage.setItem("whisper-show-message-time", String(showMessageTimeEl.checked));
  localStorage.setItem("whisper-parse-latex", String(parseLatexEl.checked));
  localStorage.setItem("whisper-theme", schemeSelectorEl.value);
}

showMessageTimeEl.addEventListener("change", () => {
  persistSettings();
});

parseLatexEl.addEventListener("change", () => {
  persistSettings();
});

clearMessagesEl.addEventListener("click", async () => {
  try {
    if (backendEnabled) {
      await apiRequest("/api/conversations/clear", { method: "POST", body: { target: "group" } });
    }
    state.conversations.group = [];
    setFormStatus(profileSaveStatusEl, "群聊记录已清空");
  } catch (error) {
    setFormStatus(profileSaveStatusEl, error.message, true);
  }
});

clearFriendMessagesEl.addEventListener("click", async () => {
  const name = state.friendMenuMember;
  if (!name) return;
  try {
    if (backendEnabled) {
      await apiRequest("/api/conversations/clear", { method: "POST", body: { target: name } });
    }
    const conversationId = conversationIdFor(name);
    state.conversations[conversationId] = [];
    hideFriendMenu();
    if (state.currentConversation === conversationId) renderMessages();
  } catch (_) {}
});

friendColorButtons.forEach((button) => {
  button.addEventListener("click", async (event) => {
    event.stopPropagation();
    const name = state.friendMenuMember;
    const color = button.dataset.friendColor;
    if (!name || !FRIEND_MESSAGE_COLORS.has(color)) return;

    friendColorButtons.forEach((item) => { item.disabled = true; });
    try {
      if (backendEnabled) {
        await apiRequest("/api/friends/color", {
          method: "PATCH",
          body: { name, color }
        });
      }
      if (color === "default") state.friendColors.delete(name);
      else state.friendColors.set(name, color);
      if (!backendEnabled) persistDemoFriendColors();
      updateFriendColorMenu(name);
      renderMessages();
      setFormStatus(friendColorStatusEl, "已保存");
    } catch (error) {
      setFormStatus(friendColorStatusEl, error.message, true);
    } finally {
      friendColorButtons.forEach((item) => { item.disabled = false; });
      button.focus();
    }
  });
});

deleteFriendEl.addEventListener("click", async () => {
  const name = state.friendMenuMember;
  const member = memberByName(name);
  if (!name || member?.reserved) return;
  try {
    if (backendEnabled) {
      await apiRequest("/api/friends/delete", { method: "POST", body: { name } });
    }
    const conversationId = conversationIdFor(name);
    state.friends.delete(name);
    state.friendColors.delete(name);
    if (!backendEnabled) persistDemoFriendColors();
    hideFriendMenu();
    if (state.currentConversation === conversationId) {
      switchConversation("group");
      return;
    }
    renderFriends();
  } catch (_) {
    hideFriendMenu();
  }
});

schemeSelectorEl.addEventListener("change", () => {
  document.body.dataset.theme = schemeSelectorEl.value;
  persistSettings();
});

logoutEl.addEventListener("click", async () => {
  if (!backendEnabled) return;
  try {
    await apiRequest("/api/logout", { method: "POST" });
  } catch (_) {}
  showAuthUI();
});

document.addEventListener("click", (event) => {
  if (!friendMenuEl.hidden && !friendMenuEl.contains(event.target)) hideFriendMenu();
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    hideFriendMenu();
    setSidebar(false);
  }
});

const storedTheme = localStorage.getItem("whisper-theme");
if (["dune", "ocean", "paper"].includes(storedTheme)) {
  document.body.dataset.theme = storedTheme;
  schemeSelectorEl.value = storedTheme;
}

members.forEach((member) => {
  if (state.conversations[conversationIdFor(member.name)].length > 0) {
    state.friends.add(member.name);
  }
});

window.whisperDemo = {
  receivePrivateMessage,
  updateMemberPresence
};

pinSidebarEl.checked = localStorage.getItem("whisper-pin-sidebar") === "true";
showMessageTimeEl.checked = localStorage.getItem("whisper-show-message-time") !== "false";
parseLatexEl.checked = localStorage.getItem("whisper-parse-latex") !== "false";
setSidebar(pinSidebarEl.checked);
setAuthMode("login");
startApplication();
