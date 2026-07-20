const MAX_NICKNAME_LENGTH = 7;
const RESERVED_NICKNAME = "coco";
const PUBLIC_GROUP_ID = "public";
const PUBLIC_GROUP_NAME = "公共大厅";
const PUBLIC_GROUP_CONVERSATION = `group:${PUBLIC_GROUP_ID}`;
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
let serverInstance = "";

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
  currentConversation: PUBLIC_GROUP_CONVERSATION,
  friendMenuMember: null,
  groupMenuOpen: false,
  editingGroupId: null,
  friends: new Set([RESERVED_NICKNAME]),
  friendColors: loadStoredFriendColors(),
  unreadCounts: new Map(),
  nudgingConversations: new Set(),
  groups: new Map([
    [PUBLIC_GROUP_ID, {
      id: PUBLIC_GROUP_ID,
      name: PUBLIC_GROUP_NAME,
      signature: "所有成员都可以在这里聊天。",
      owner: "",
      isOwner: false,
      system: true,
      historyVisible: true,
      members: [
        { name: initialProfileName, online: true, owner: false },
        ...members.map((member) => ({ name: member.name, online: member.online, owner: false }))
      ]
    }]
  ]),
  profile: {
    signature: initialSignature,
    password: ""
  },
  conversations: {
    [PUBLIC_GROUP_CONVERSATION]: [
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

const nudgeTimers = new Map();

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
const groupPanelEl = document.querySelector("#group-panel");
const footerEl = document.querySelector("#footer");
const conversationTitleEl = document.querySelector("#conversation-title");
const conversationStatusEl = document.querySelector("#conversation-status");
const profileSignatureEl = document.querySelector("#profile-signature");
const backToGroupEl = document.querySelector("#back-to-group");
const conversationListEl = document.querySelector("#conversation-list");
const formEl = document.querySelector("#chatform");
const inputEl = document.querySelector("#chatinput");
const sidebarEl = document.querySelector("#sidebar");
const sidebarContentEl = document.querySelector("#sidebar-content");
const sidebarToggleEl = document.querySelector("#sidebar-toggle");
const sidebarUnreadEl = document.querySelector("#sidebar-unread");
const pinSidebarEl = document.querySelector("#pin-sidebar");
const friendsEl = document.querySelector("#friends");
const friendsEmptyEl = document.querySelector("#friends-empty");
const usersEl = document.querySelector("#users");
const friendMenuEl = document.querySelector("#friend-menu");
const groupMenuEl = document.querySelector("#group-menu");
const groupMenuToggleEl = document.querySelector("#group-menu-toggle");
const newGroupEl = document.querySelector("#new-group");
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
const conversationTitleButtonEl = document.querySelector("#conversation-title");
const groupFormEl = document.querySelector("#group-form");
const groupPanelTitleEl = document.querySelector("#group-panel-title");
const groupNameInputEl = document.querySelector("#group-name-input");
const groupSignatureInputEl = document.querySelector("#group-signature-input");
const groupHistoryVisibleEl = document.querySelector("#group-history-visible");
const groupHistoryFieldEl = document.querySelector("#group-history-field");
const groupMembersFieldEl = document.querySelector("#group-members-field");
const groupMemberOptionsEl = document.querySelector("#group-member-options");
const groupSaveEl = document.querySelector("#group-save");
const groupSaveStatusEl = document.querySelector("#group-save-status");
const groupOwnerActionsEl = document.querySelector("#group-owner-actions");
const groupTransferOwnerEl = document.querySelector("#group-transfer-owner");
const groupTransferEl = document.querySelector("#group-transfer");
const groupDissolveEl = document.querySelector("#group-dissolve");
const groupOwnerStatusEl = document.querySelector("#group-owner-status");
const groupMemberActionsEl = document.querySelector("#group-member-actions");
const groupLeaveEl = document.querySelector("#group-leave");
const groupMemberStatusEl = document.querySelector("#group-member-status");

function conversationIdFor(name) {
  return `dm:${name}`;
}

function friendNameForConversation(conversationId) {
  return conversationId.startsWith("dm:") ? conversationId.slice(3) : "";
}

function groupConversationIdFor(id) {
  return `group:${id}`;
}

function groupIdForConversation(conversationId = state.currentConversation) {
  return conversationId.startsWith("group:") ? conversationId.slice(6) : "";
}

function isGroupConversation(conversationId = state.currentConversation) {
  return conversationId.startsWith("group:");
}

function isGroupSettingsConversation(conversationId = state.currentConversation) {
  return conversationId.startsWith("group-settings:");
}

function groupSettingsIdFor(groupId) {
  return `group-settings:${groupId}`;
}

function isGroupCreateConversation(conversationId = state.currentConversation) {
  return conversationId === "group-create";
}

function groupIdFromSettings(conversationId = state.currentConversation) {
  return conversationId.startsWith("group-settings:") ? conversationId.slice(15) : "";
}

function groupById(groupId) {
  return state.groups.get(groupId) || null;
}

function currentGroup() {
  if (isGroupConversation()) return groupById(groupIdForConversation());
  if (isGroupSettingsConversation()) return groupById(groupIdFromSettings());
  return null;
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
  if (isGroupConversation() || isGroupSettingsConversation()
    || isGroupCreateConversation() || state.currentConversation === "self") {
    return null;
  }
  return memberByName(state.currentConversation.slice(3));
}

function currentMessages() {
  return state.conversations[state.currentConversation] || [];
}

function unreadCountFor(conversationId) {
  return state.unreadCounts.get(conversationId) || 0;
}

function totalUnreadCount() {
  return Array.from(state.unreadCounts.values()).reduce((total, count) => total + count, 0);
}

function displayUnreadCount(count) {
  return count > 99 ? "99+" : String(count);
}

function stopConversationNudge(conversationId) {
  state.nudgingConversations.delete(conversationId);
  const timer = nudgeTimers.get(conversationId);
  if (timer !== undefined) window.clearTimeout(timer);
  nudgeTimers.delete(conversationId);
}

function clearConversationUnread(conversationId) {
  state.unreadCounts.delete(conversationId);
  stopConversationNudge(conversationId);
}

function resetUnreadState() {
  nudgeTimers.forEach((timer) => window.clearTimeout(timer));
  nudgeTimers.clear();
  state.unreadCounts.clear();
  state.nudgingConversations.clear();
}

function markConversationUnread(conversationId) {
  const previousCount = unreadCountFor(conversationId);
  state.unreadCounts.set(conversationId, previousCount + 1);
  stopConversationNudge(conversationId);
  state.nudgingConversations.add(conversationId);
  nudgeTimers.set(conversationId, window.setTimeout(() => {
    state.nudgingConversations.delete(conversationId);
    nudgeTimers.delete(conversationId);
    renderFriends();
  }, 700));
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
  const groupSettingsId = groupIdFromSettings(conversationId);
  const allowedPanel = conversationId === "self" || conversationId === "group-create"
    || (groupSettingsId && state.groups.has(groupSettingsId));
  if (!allowedPanel && !state.conversations[conversationId]) return;

  clearConversationUnread(conversationId);
  state.currentConversation = conversationId;
  inputEl.value = "";
  resizeInput();
  hideFriendMenu();
  hideGroupMenu();
  renderConversation();

  if (!pinSidebarEl.checked) setSidebar(false);
  if (conversationId === "self" || conversationId === "group-create" || groupSettingsId) {
    window.scrollTo({ top: 0 });
    if (conversationId === "self") profileNameEl.focus();
    else groupNameInputEl.focus();
  } else {
    inputEl.focus();
    window.scrollTo({ top: document.body.scrollHeight });
  }
}

function mentionInGroup(name) {
  if (!isGroupConversation()) return;
  inputEl.value = `@${name} `;
  resizeInput();
  inputEl.focus();
  inputEl.setSelectionRange(inputEl.value.length, inputEl.value.length);
}

function renderConversationHeader() {
  if (state.currentConversation === "self") {
    document.body.dataset.conversationKind = "profile";
    conversationTitleEl.textContent = `@ ${state.currentUser}`;
    conversationTitleEl.disabled = true;
    setConversationMeta("在线", state.profile.signature);
    backToGroupEl.hidden = false;
    return;
  }

  if (isGroupCreateConversation()) {
    document.body.dataset.conversationKind = "profile";
    conversationTitleEl.textContent = "新建群聊";
    conversationTitleEl.disabled = true;
    setConversationMeta("选择成员并设置群聊资料");
    backToGroupEl.hidden = false;
    return;
  }

  if (isGroupSettingsConversation()) {
    const group = currentGroup();
    document.body.dataset.conversationKind = "profile";
    conversationTitleEl.textContent = group ? `# ${group.name}` : "群聊信息";
    conversationTitleEl.disabled = true;
    setConversationMeta(group?.system ? "公共大厅" : group?.isOwner ? "群主配置" : "群聊信息", group?.signature || "");
    backToGroupEl.hidden = false;
    return;
  }

  const group = currentGroup();
  if (group) {
    const onlineCount = group.members.filter((item) => item.online || item.name === state.currentUser).length;
    document.body.dataset.conversationKind = "group";
    conversationTitleEl.textContent = `# ${group.name}`;
    conversationTitleEl.disabled = false;
    conversationTitleEl.setAttribute("aria-label", `查看${group.name}群聊信息`);
    setConversationMeta(`${onlineCount}/${group.members.length} 人在线`, group.signature || "");
    inputEl.placeholder = `发送到${group.name}...`;
    backToGroupEl.hidden = group.id === PUBLIC_GROUP_ID;
    return;
  }

  const member = currentMember();
  document.body.dataset.conversationKind = "private";
  conversationTitleEl.textContent = member ? `@ ${member.name}` : "会话";
  conversationTitleEl.disabled = true;
  setConversationMeta(member?.online ? "在线" : "离线，可留言", member?.signature || "");
  inputEl.placeholder = member?.online
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
      && isGroupConversation();
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
    const unreadCount = unreadCountFor(conversationId);
    const item = document.createElement("li");
    item.className = "friend-row";
    item.classList.toggle("unread", unreadCount > 0);
    item.classList.toggle("offline", !member.online);
    item.classList.toggle("active", state.currentConversation === conversationId);
    item.classList.toggle("reserved", Boolean(member.reserved));

    const nameButton = document.createElement("button");
    nameButton.type = "button";
    nameButton.className = "friend-name";
    const nameLabel = document.createElement("span");
    nameLabel.className = "friend-name-label";
    nameLabel.classList.toggle("nudging", state.nudgingConversations.has(conversationId));
    nameLabel.textContent = member.reserved ? `@${member.name}` : member.name;
    nameButton.append(nameLabel);
    if (unreadCount > 0) {
      const unread = document.createElement("span");
      unread.className = "friend-unread";
      unread.textContent = displayUnreadCount(unreadCount);
      unread.setAttribute("aria-hidden", "true");
      nameButton.append(unread);
    }
    nameButton.setAttribute(
      "aria-label",
      unreadCount > 0
        ? `与${member.name}私聊，${unreadCount}条未读消息`
        : `与${member.name}私聊（好友）`
    );
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

function renderGroups() {
  conversationListEl.replaceChildren();
  const groups = [...state.groups.values()].sort((left, right) => {
    if (left.system !== right.system) return left.system ? -1 : 1;
    return left.name.localeCompare(right.name, "zh-CN");
  });
  groups.forEach((group) => {
    const conversationId = groupConversationIdFor(group.id);
    const unreadCount = unreadCountFor(conversationId);
    const onlineCount = group.members.filter((member) => member.online || member.name === state.currentUser).length;
    const item = document.createElement("li");
    item.className = "conversation-row";
    item.classList.toggle("unread", unreadCount > 0);

    const button = document.createElement("button");
    button.type = "button";
    button.className = "conversation-entry";
    button.classList.toggle("active", state.currentConversation === conversationId);
    button.setAttribute("aria-current", state.currentConversation === conversationId ? "page" : "false");
    button.setAttribute(
      "aria-label",
      unreadCount > 0 ? `打开群聊${group.name}，${unreadCount}条未读消息` : `打开群聊${group.name}`
    );
    const label = document.createElement("span");
    label.textContent = `# ${group.name}`;
    label.className = "conversation-label";
    label.classList.toggle("nudging", state.nudgingConversations.has(conversationId));
    const count = document.createElement("span");
    count.className = "conversation-count";
    count.textContent = unreadCount > 0 ? displayUnreadCount(unreadCount) : String(onlineCount);
    button.append(label, count);
    button.addEventListener("click", () => switchConversation(conversationId));
    item.append(button);
    conversationListEl.append(item);
  });
}

function renderGroupMemberOptions(group, editable) {
  groupMemberOptionsEl.replaceChildren();
  const selectedNames = new Set(group?.members?.map((member) => member.name) || []);
  const availableMembers = [
    { name: state.currentUser, online: true, self: true },
    ...sortedMembers(members.filter((member) => !member.reserved))
  ];
  availableMembers.forEach((member) => {
    const row = document.createElement("label");
    row.className = "group-member-option";
    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.value = member.name;
    checkbox.checked = isGroupCreateConversation()
      ? Boolean(member.self)
      : selectedNames.has(member.name);
    checkbox.disabled = member.self || !editable;
    const text = document.createElement("span");
    text.textContent = member.self ? `${member.name}（本人）` : member.name;
    row.append(checkbox, text);
    groupMemberOptionsEl.append(row);
  });
}

function renderGroupPanel() {
  const createMode = isGroupCreateConversation();
  const group = currentGroup();
  const editable = createMode || Boolean(group?.isOwner && !group.system);
  const readOnlyGroup = !createMode && Boolean(group);
  groupPanelTitleEl.textContent = createMode ? "新建群聊" : group ? `${group.name} 群聊信息` : "群聊信息";
  groupNameInputEl.value = createMode ? "" : group?.name || "";
  groupSignatureInputEl.value = createMode ? "" : group?.signature || "";
  groupHistoryVisibleEl.checked = createMode ? true : Boolean(group?.historyVisible);
  groupNameInputEl.disabled = !editable;
  groupSignatureInputEl.disabled = !editable;
  groupHistoryVisibleEl.disabled = !editable;
  groupHistoryFieldEl.hidden = !createMode && Boolean(group?.system);
  groupMembersFieldEl.hidden = !createMode && !group;
  groupSaveEl.textContent = createMode ? "创建群聊" : "保存群聊配置";
  groupSaveEl.disabled = !editable;
  document.querySelector("#group-save-actions").hidden = !editable;
  if (!groupMembersFieldEl.hidden) renderGroupMemberOptions(group, editable);

  const ownerEditable = readOnlyGroup && group.isOwner && !group.system;
  groupOwnerActionsEl.hidden = !ownerEditable;
  groupMemberActionsEl.hidden = !readOnlyGroup || Boolean(group.isOwner) || Boolean(group.system);
  groupTransferOwnerEl.replaceChildren();
  if (ownerEditable) {
    group.members.filter((member) => !member.owner).forEach((member) => {
      const option = document.createElement("option");
      option.value = member.name;
      option.textContent = member.name;
      groupTransferOwnerEl.append(option);
    });
    groupTransferOwnerEl.disabled = groupTransferOwnerEl.options.length === 0;
  }
  groupSaveStatusEl.textContent = "";
  groupOwnerStatusEl.textContent = "";
  groupMemberStatusEl.textContent = "";
}

function renderConversationNavigation() {
  renderGroups();
  renderFriends();
  renderUsers();
  updateSidebarUnreadIndicator();
}

function renderConversation() {
  const profileMode = state.currentConversation === "self";
  const groupPanelMode = isGroupCreateConversation() || isGroupSettingsConversation();
  const settingsMode = profileMode || groupPanelMode;
  messagesEl.hidden = settingsMode;
  profilePanelEl.hidden = !profileMode;
  groupPanelEl.hidden = !groupPanelMode;
  footerEl.hidden = settingsMode;
  renderConversationHeader();
  if (profileMode) renderProfilePanel();
  else if (groupPanelMode) renderGroupPanel();
  else renderMessages();
  renderConversationNavigation();
}

function sendCurrentMessage() {
  if (state.currentConversation === "self" || isGroupCreateConversation() || isGroupSettingsConversation()) return;
  const text = inputEl.value.trim();
  if (!text) return;

  const member = currentMember();
  const group = currentGroup();
  if (backendEnabled) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      inputEl.placeholder = "正在重新连接...";
      return;
    }
    socket.send(JSON.stringify({
      type: "message",
      scope: member ? "private" : "group",
      to: member?.name || group?.id || PUBLIC_GROUP_ID,
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
  updateSidebarUnreadIndicator();
  if (!open) hideFriendMenu();
}

function updateSidebarUnreadIndicator() {
  const unreadCount = totalUnreadCount();
  const open = sidebarEl.classList.contains("open");
  sidebarUnreadEl.hidden = open || unreadCount === 0;
  sidebarUnreadEl.textContent = unreadCount > 0 ? displayUnreadCount(unreadCount) : "";
  const action = open ? "关闭会话列表" : "打开会话列表";
  sidebarToggleEl.setAttribute(
    "aria-label",
    unreadCount > 0 ? `${action}，${unreadCount}条未读消息` : action
  );
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

function hideGroupMenu() {
  groupMenuEl.hidden = true;
  state.groupMenuOpen = false;
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
  state.groups.forEach((group) => {
    group.members.forEach((member) => {
      if (member.name === previousName) member.name = nextName;
    });
    if (group.owner === previousName) group.owner = nextName;
  });
}

function updateMemberPresence(name, online) {
  const member = memberByName(name);
  if (!member || member.reserved) return;

  member.online = online;
  state.groups.forEach((group) => {
    const groupMember = group.members.find((item) => item.name === name);
    if (groupMember) groupMember.online = online;
  });
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
  const conversationId = conversationIdFor(from);
  if (state.currentConversation !== conversationId || document.hidden) {
    markConversationUnread(conversationId);
  }
  renderConversationNavigation();
  if (state.currentConversation === conversationId) renderMessages();
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
  resetUnreadState();
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

function applyGroupView(group) {
  if (!group?.id) return;
  state.groups.set(group.id, group);
  const conversationId = groupConversationIdFor(group.id);
  if (!state.conversations[conversationId]) state.conversations[conversationId] = [];
}

function removeGroup(groupId) {
  state.groups.delete(groupId);
  delete state.conversations[groupConversationIdFor(groupId)];
  clearConversationUnread(groupConversationIdFor(groupId));
  if (state.currentConversation === groupConversationIdFor(groupId)
    || state.currentConversation === groupSettingsIdFor(groupId)) {
    state.currentConversation = PUBLIC_GROUP_CONVERSATION;
  }
}

function hydrateBootstrap(payload) {
  if (serverInstance && payload.serverInstance && serverInstance !== payload.serverInstance) {
    window.location.reload();
    return false;
  }
  if (payload.serverInstance) serverInstance = payload.serverInstance;
  state.currentUser = payload.self.name;
  state.profile.signature = payload.self.signature || "";
  state.currentConversation = PUBLIC_GROUP_CONVERSATION;
  state.friendMenuMember = null;
  state.friends = new Set(payload.friends || []);
  state.friendColors = new Map(Object.entries(payload.friendColors || {}));
  state.conversations = payload.conversations || { [PUBLIC_GROUP_CONVERSATION]: [], "dm:coco": [] };
  if (state.conversations.group && !state.conversations[PUBLIC_GROUP_CONVERSATION]) {
    state.conversations[PUBLIC_GROUP_CONVERSATION] = state.conversations.group;
    delete state.conversations.group;
  }
  state.groups = new Map();
  (payload.groups || []).forEach((group) => applyGroupView(group));
  if (!state.groups.has(PUBLIC_GROUP_ID)) {
    applyGroupView({
      id: PUBLIC_GROUP_ID, name: PUBLIC_GROUP_NAME, signature: "", owner: "",
      isOwner: false, system: true, historyVisible: true, members: []
    });
  }

  members.splice(0, members.length, ...(payload.members || []));
  members.forEach((member) => {
    const key = conversationIdFor(member.name);
    if (!state.conversations[key]) state.conversations[key] = [];
  });
  if (!state.conversations[PUBLIC_GROUP_CONVERSATION]) state.conversations[PUBLIC_GROUP_CONVERSATION] = [];

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
  state.groups.forEach((group) => {
    group.members.forEach((groupMember) => {
      if (groupMember.name === previousName) {
        groupMember.name = memberView.name;
        groupMember.online = memberView.online;
      }
    });
    if (group.owner === previousName) group.owner = memberView.name;
  });
  if (previousKey !== nextKey) {
    state.conversations[nextKey] = state.conversations[previousKey] || [];
    delete state.conversations[previousKey];
    if (state.friends.delete(previousName)) state.friends.add(memberView.name);
    if (state.friendColors.has(previousName)) {
      state.friendColors.set(memberView.name, state.friendColors.get(previousName));
      state.friendColors.delete(previousName);
    }
    if (state.unreadCounts.has(previousKey)) {
      state.unreadCounts.set(nextKey, state.unreadCounts.get(previousKey));
      state.unreadCounts.delete(previousKey);
    }
    stopConversationNudge(previousKey);
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
    const isNewMessage = !messages.some((message) => message.id === event.message.id);
    if (isNewMessage) messages.push(event.message);
    const conversationFriend = friendNameForConversation(event.conversation);
    const isIncomingPrivateMessage = event.conversation.startsWith("dm:")
      && event.message.from !== state.currentUser;
    const isIncomingGroupMessage = isGroupConversation(event.conversation)
      && event.message.from !== state.currentUser;
    if (event.friend) state.friends.add(event.friend);
    if (isIncomingPrivateMessage) state.friends.add(event.friend || conversationFriend);
    if (isNewMessage && isIncomingPrivateMessage
      && (state.currentConversation !== event.conversation || document.hidden)) {
      markConversationUnread(event.conversation);
    }
    if (isNewMessage && isIncomingGroupMessage
      && (state.currentConversation !== event.conversation || document.hidden)) {
      markConversationUnread(event.conversation);
    }
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
      state.groups.forEach((group) => {
        const groupMember = group.members.find((item) => item.name === event.name);
        if (groupMember) groupMember.online = event.online;
      });
      renderConversation();
    }
    return;
  }

  if (event.type === "profile") {
    renameRemoteMember(event.previousName, event.member);
    renderConversation();
    return;
  }

  if (event.type === "group") {
    applyGroupView(event.group);
    renderConversation();
    return;
  }

  if (event.type === "group_removed") {
    removeGroup(event.groupId);
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
        if (hydrateBootstrap(payload) === false) return;
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
    if (hydrateBootstrap(payload) === false) return;
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
    if (hydrateBootstrap(payload) === false) return;
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
backToGroupEl.addEventListener("click", () => {
  if (isGroupSettingsConversation()) switchConversation(groupConversationIdFor(groupIdFromSettings()));
  else switchConversation(PUBLIC_GROUP_CONVERSATION);
});
conversationTitleButtonEl.addEventListener("click", () => {
  if (isGroupConversation()) switchConversation(groupSettingsIdFor(groupIdForConversation()));
});

groupMenuToggleEl.addEventListener("click", (event) => {
  event.stopPropagation();
  if (!groupMenuEl.hidden) {
    hideGroupMenu();
    return;
  }
  hideFriendMenu();
  const rect = groupMenuToggleEl.getBoundingClientRect();
  groupMenuEl.hidden = false;
  state.groupMenuOpen = true;
  const menuRect = groupMenuEl.getBoundingClientRect();
  groupMenuEl.style.left = `${Math.max(8, Math.min(rect.right - menuRect.width, window.innerWidth - menuRect.width - 8))}px`;
  groupMenuEl.style.top = `${Math.min(rect.bottom + 4, window.innerHeight - menuRect.height - 8)}px`;
});

newGroupEl.addEventListener("click", () => {
  hideGroupMenu();
  switchConversation("group-create");
});

function selectedGroupMemberNames() {
  return [...groupMemberOptionsEl.querySelectorAll("input[type=checkbox]:checked")]
    .map((input) => input.value);
}

function demoGroupView(name, signature, historyVisible, memberNames) {
  const id = `demo-${Date.now()}`;
  const names = [...new Set([state.currentUser, ...memberNames])];
  return {
    id, name, signature, owner: state.currentUser, isOwner: true, system: false,
    historyVisible,
    members: names.map((memberName) => {
      const member = memberName === state.currentUser ? null : memberByName(memberName);
      return { name: memberName, online: member ? member.online : true, owner: memberName === state.currentUser };
    })
  };
}

groupFormEl.addEventListener("submit", async (event) => {
  event.preventDefault();
  const name = groupNameInputEl.value.trim();
  if (!name) {
    setFormStatus(groupSaveStatusEl, "群聊名称不能为空", true);
    return;
  }
  if (name === PUBLIC_GROUP_NAME) {
    setFormStatus(groupSaveStatusEl, `${PUBLIC_GROUP_NAME}是系统保留群名`, true);
    return;
  }
  const memberNames = selectedGroupMemberNames().filter((memberName) => memberName !== state.currentUser);
  if (isGroupCreateConversation() && memberNames.length === 0) {
    setFormStatus(groupSaveStatusEl, "至少选择一名成员", true);
    return;
  }
  const editing = isGroupSettingsConversation();
  const groupId = editing ? groupIdFromSettings() : "";
  const body = {
    id: groupId,
    name,
    signature: groupSignatureInputEl.value.trim(),
    historyVisible: groupHistoryVisibleEl.checked,
    members: memberNames
  };
  groupSaveEl.disabled = true;
  try {
    const group = backendEnabled
      ? await apiRequest("/api/groups", { method: editing ? "PATCH" : "POST", body })
      : demoGroupView(name, body.signature, body.historyVisible, memberNames);
    applyGroupView(group);
    setFormStatus(groupSaveStatusEl, editing ? "群聊配置已保存" : "群聊已创建");
    switchConversation(groupConversationIdFor(group.id));
  } catch (error) {
    setFormStatus(groupSaveStatusEl, error.message, true);
    groupSaveEl.disabled = false;
  }
});

groupTransferEl.addEventListener("click", async () => {
  const group = currentGroup();
  const newOwner = groupTransferOwnerEl.value;
  if (!group || !newOwner) return;
  try {
    const updated = await apiRequest("/api/groups/transfer", {
      method: "POST", body: { id: group.id, newOwner }
    });
    applyGroupView(updated);
    setFormStatus(groupOwnerStatusEl, "群主已转移");
    switchConversation(groupSettingsIdFor(group.id));
  } catch (error) {
    setFormStatus(groupOwnerStatusEl, error.message, true);
  }
});

groupDissolveEl.addEventListener("click", async () => {
  const group = currentGroup();
  if (!group || !window.confirm(`确定解散“${group.name}”吗？群消息也会被删除。`)) return;
  try {
    await apiRequest("/api/groups/dissolve", { method: "POST", body: { id: group.id } });
    removeGroup(group.id);
    switchConversation(PUBLIC_GROUP_CONVERSATION);
  } catch (error) {
    setFormStatus(groupOwnerStatusEl, error.message, true);
  }
});

groupLeaveEl.addEventListener("click", async () => {
  const group = currentGroup();
  if (!group || !window.confirm(`确定退出“${group.name}”吗？`)) return;
  try {
    await apiRequest("/api/groups/leave", { method: "POST", body: { id: group.id } });
    removeGroup(group.id);
    switchConversation(PUBLIC_GROUP_CONVERSATION);
  } catch (error) {
    setFormStatus(groupMemberStatusEl, error.message, true);
  }
});

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
      await apiRequest("/api/conversations/clear", {
        method: "POST", body: { target: PUBLIC_GROUP_CONVERSATION }
      });
    }
    state.conversations[PUBLIC_GROUP_CONVERSATION] = [];
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
    clearConversationUnread(conversationId);
    hideFriendMenu();
    if (state.currentConversation === conversationId) renderMessages();
    renderConversationNavigation();
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
    clearConversationUnread(conversationId);
    if (!backendEnabled) persistDemoFriendColors();
    hideFriendMenu();
    if (state.currentConversation === conversationId) {
      switchConversation(PUBLIC_GROUP_CONVERSATION);
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
  if (!groupMenuEl.hidden && !groupMenuEl.contains(event.target)
    && event.target !== groupMenuToggleEl) hideGroupMenu();
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    hideFriendMenu();
    hideGroupMenu();
    setSidebar(false);
  }
});

document.addEventListener("visibilitychange", () => {
  if (document.hidden || !state.currentConversation.startsWith("dm:")) return;
  if (unreadCountFor(state.currentConversation) === 0) return;
  clearConversationUnread(state.currentConversation);
  renderConversationNavigation();
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
