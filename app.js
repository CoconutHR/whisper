const MAX_NICKNAME_LENGTH = 7;
const RESERVED_NICKNAME = "coco";
const PUBLIC_GROUP_ID = "public";
const PUBLIC_GROUP_NAME = "公共大厅";
const PUBLIC_GROUP_CONVERSATION = `group:${PUBLIC_GROUP_ID}`;
const DEFAULT_DOCUMENT_TITLE = document.title;
const NEW_MESSAGE_TITLE = `【新消息】${DEFAULT_DOCUMENT_TITLE}    `;
const HTTP_PROTOCOLS = new Set(["http:", "https:"]);
const FRIEND_MESSAGE_COLORS = new Set(["default", "green", "blue", "cyan", "amber", "rose"]);
const MAX_ATTACHMENT_SIZE = 50 * 1024 * 1024;
const MAX_ATTACHMENT_TOTAL_SIZE = 100 * 1024 * 1024;
const MAX_ATTACHMENTS_PER_MESSAGE = 5;
const MAX_STICKER_SIZE = 10 * 1024 * 1024;
const VIDEO_CONTENT_TYPES = new Set([
  "video/mp4", "video/webm", "video/ogg", "video/quicktime", "video/x-m4v"
]);
const AUDIO_CONTENT_TYPES = new Set([
  "audio/mpeg", "audio/mp4", "audio/aac", "audio/wav", "audio/x-wav",
  "audio/ogg", "audio/webm", "audio/flac", "audio/x-flac"
]);
const DOCUMENT_CONTENT_TYPES = new Set([
  "application/pdf", "text/plain", "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.ms-excel",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.ms-powerpoint",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation"
]);
const FILE_EXTENSION_CONTENT_TYPES = new Map([
  ["pdf", "application/pdf"], ["txt", "text/plain"],
  ["doc", "application/msword"],
  ["docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"],
  ["xls", "application/vnd.ms-excel"],
  ["xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"],
  ["ppt", "application/vnd.ms-powerpoint"],
  ["pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"],
  ["mp3", "audio/mpeg"], ["m4a", "audio/mp4"], ["aac", "audio/aac"],
  ["wav", "audio/wav"], ["oga", "audio/ogg"], ["ogg", "audio/ogg"],
  ["opus", "audio/ogg"], ["weba", "audio/webm"], ["flac", "audio/flac"],
  ["mp4", "video/mp4"], ["m4v", "video/x-m4v"], ["mov", "video/quicktime"],
  ["webm", "video/webm"], ["ogv", "video/ogg"]
]);
const CONTENT_TYPE_ALIASES = new Map([
  ["audio/mp3", "audio/mpeg"], ["audio/x-m4a", "audio/mp4"],
  ["audio/x-aac", "audio/aac"], ["application/x-pdf", "application/pdf"]
]);
const EMOJI_BATCH_SIZE = 160;
const EMOJI_OPTIONS = [
  "😀", "😃", "😄", "😁", "😆", "😅", "😂", "🤣",
  "🥲", "🥹", "😊", "😇", "🙂", "🙃", "😉", "😌",
  "😍", "🥰", "😘", "😗", "😙", "😚", "😋", "😛",
  "😝", "😜", "🤪", "🤨", "🧐", "🤓", "😎", "🥸",
  "🤩", "🥳", "🙂‍↕️", "😏", "😒", "🙂‍↔️", "😞", "😔",
  "😟", "😕", "🙁", "☹️", "😣", "😖", "😫", "😩",
  "🥺", "😢", "😭", "😤", "😠", "😡", "🤬", "🤯",
  "😳", "🥵", "🥶", "😱", "😨", "😰", "😥", "😓",
  "🫣", "🤗", "🫡", "🤔", "🫢", "🤭", "🤫", "🤥",
  "😶", "😶‍🌫️", "😐", "😑", "😬", "🫨", "🙄", "😯",
  "😦", "😧", "😮", "😲", "🥱", "😴", "🤤", "😪",
  "😵", "😵‍💫", "🤐", "🥴", "🤢", "🤮", "🤧", "😷",
  "🤒", "🤕", "🤑", "🤠", "😈", "👿", "👹", "👺",
  "🤡", "💩", "👻", "💀", "☠️", "👽", "🤖", "🎃",

  "😺", "😸", "😹", "😻", "😼", "😽", "🙀", "😿",
  "😾", "🙈", "🙉", "🙊", "💋", "💌", "💘", "💝",
  "💖", "💗", "💓", "💞", "💕", "💟", "❣️", "💔",
  "❤️", "🩷", "🧡", "💛", "💚", "💙", "🩵", "💜",
  "🤎", "🖤", "🩶", "🤍", "❤️‍🔥", "❤️‍🩹", "💯", "💢",
  "💥", "💫", "💦", "💨", "🕳️", "💬", "👁️‍🗨️", "🗨️",

  "👋", "🤚", "🖐️", "✋", "🖖", "🫱", "🫲", "🫳",
  "🫴", "👌", "🤌", "🤏", "✌️", "🤞", "🫰", "🤟",
  "🤘", "🤙", "👈", "👉", "👆", "🖕", "👇", "☝️",
  "🫵", "👍", "👎", "✊", "👊", "🤛", "🤜", "👏",
  "🙌", "🫶", "👐", "🤲", "🤝", "🙏", "✍️", "💅",
  "🤳", "💪", "🦾", "🦿", "🦵", "🦶", "👂", "🦻",
  "👃", "🧠", "🫀", "🫁", "🦷", "🦴", "👀", "👁️",
  "👅", "👄", "🫦", "👶", "🧒", "👦", "👧", "🧑",
  "👱", "👨", "🧔", "👩", "🧓", "👴", "👵", "🙍",
  "🙎", "🙅", "🙆", "💁", "🙋", "🧏", "🙇", "🤦",
  "🤷", "👮", "👷", "💂", "🕵️", "👩‍⚕️", "👩‍🎓", "👩‍💻",
  "👩‍🚀", "🧙", "🧚", "🧛", "🧜", "🧝", "🦸", "🦹",

  "🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼",
  "🐻‍❄️", "🐨", "🐯", "🦁", "🐮", "🐷", "🐸", "🐵",
  "🐔", "🐧", "🐦", "🐤", "🦆", "🦅", "🦉", "🦇",
  "🐺", "🐗", "🐴", "🦄", "🐝", "🪲", "🐞", "🦋",
  "🐌", "🐛", "🪱", "🐜", "🕷️", "🦂", "🐢", "🐍",
  "🦎", "🦖", "🦕", "🐙", "🦑", "🦐", "🦞", "🦀",
  "🐡", "🐠", "🐟", "🐬", "🐳", "🦈", "🦭", "🐊",
  "🐅", "🐆", "🦓", "🦍", "🦧", "🐘", "🦛", "🦏",
  "🐪", "🦒", "🦘", "🦬", "🐃", "🐂", "🐄", "🐎",
  "🐖", "🐏", "🦙", "🐐", "🦌", "🐕", "🐈", "🪶",
  "🌵", "🎄", "🌲", "🌳", "🌴", "🪵", "🌱", "🌿",
  "☘️", "🍀", "🎍", "🪴", "🎋", "🍃", "🍂", "🍁",
  "🍄", "🐚", "🪨", "🌾", "💐", "🌷", "🌹", "🥀",
  "🪻", "🌺", "🌸", "🌼", "🌻", "🌞", "🌝", "🌚",
  "🌙", "⭐", "🌟", "✨", "⚡", "🔥", "🌈", "☀️",
  "🌤️", "⛅", "🌧️", "⛈️", "❄️", "☃️", "🌊", "💧",

  "🍏", "🍎", "🍐", "🍊", "🍋", "🍌", "🍉", "🍇",
  "🍓", "🫐", "🍈", "🍒", "🍑", "🥭", "🍍", "🥥",
  "🥝", "🍅", "🍆", "🥑", "🥦", "🥬", "🥒", "🌶️",
  "🫑", "🌽", "🥕", "🫒", "🧄", "🧅", "🥔", "🍠",
  "🥐", "🥯", "🍞", "🥖", "🥨", "🧀", "🥚", "🍳",
  "🧈", "🥞", "🧇", "🥓", "🥩", "🍗", "🍖", "🌭",
  "🍔", "🍟", "🍕", "🫓", "🥪", "🥙", "🧆", "🌮",
  "🌯", "🫔", "🥗", "🥘", "🫕", "🥫", "🍝", "🍜",
  "🍲", "🍛", "🍣", "🍱", "🥟", "🦪", "🍤", "🍙",
  "🍚", "🍘", "🍥", "🥠", "🥮", "🍢", "🍡", "🍧",
  "🍨", "🍦", "🥧", "🧁", "🍰", "🎂", "🍮", "🍭",
  "🍬", "🍫", "🍿", "🍩", "🍪", "🌰", "🥜", "🍯",
  "🥛", "🍼", "🫖", "☕", "🍵", "🧃", "🥤", "🧋",
  "🍶", "🍺", "🍻", "🥂", "🍷", "🥃", "🍸", "🍹",

  "⚽", "🏀", "🏈", "⚾", "🥎", "🎾", "🏐", "🏉",
  "🥏", "🎱", "🪀", "🏓", "🏸", "🏒", "🏑", "🥍",
  "🏏", "🪃", "🥅", "⛳", "🪁", "🏹", "🎣", "🤿",
  "🥊", "🥋", "🎽", "🛹", "🛼", "🛷", "⛸️", "🥌",
  "🎿", "⛷️", "🏂", "🏋️", "🤸", "⛹️", "🤺", "🤾",
  "🏌️", "🏇", "🧘", "🏄", "🏊", "🚣", "🧗", "🚴",
  "🏆", "🥇", "🥈", "🥉", "🏅", "🎖️", "🎗️", "🎫",
  "🎪", "🤹", "🎭", "🩰", "🎨", "🎬", "🎤", "🎧",
  "🎼", "🎹", "🥁", "🪘", "🎷", "🎺", "🪗", "🎸",
  "🎻", "🎲", "♟️", "🎯", "🎳", "🎮", "🕹️", "🧩",

  "🚗", "🚕", "🚙", "🚌", "🚎", "🏎️", "🚓", "🚑",
  "🚒", "🚐", "🛻", "🚚", "🚛", "🚜", "🛵", "🏍️",
  "🚲", "🛴", "🚨", "🚔", "🚍", "🚘", "🚖", "🚡",
  "🚠", "🚟", "🚃", "🚋", "🚆", "🚄", "🚅", "🚈",
  "🚂", "🚇", "🚊", "🚉", "✈️", "🛫", "🛬", "🛩️",
  "💺", "🚁", "🚀", "🛸", "🚢", "⛵", "🚤", "🛥️",
  "🗺️", "🗿", "🗽", "🗼", "🏰", "🏯", "🏟️", "🎡",
  "🎢", "🎠", "⛲", "⛱️", "🏖️", "🏝️", "🏜️", "🌋",
  "⛰️", "🏕️", "⛺", "🛖", "🏠", "🏡", "🏢", "🏥",
  "🏦", "🏨", "🏪", "🏫", "⛪", "🕌", "🛕", "🌁",
  "🌃", "🏙️", "🌄", "🌅", "🌆", "🌇", "🌉", "♨️",

  "⌚", "📱", "💻", "⌨️", "🖥️", "🖨️", "🖱️", "💽",
  "💾", "💿", "📀", "🧮", "🎥", "📷", "📸", "📹",
  "📺", "📻", "🎙️", "⏱️", "⏰", "⌛", "🔋", "🔌",
  "💡", "🔦", "🕯️", "🧯", "🛢️", "💸", "💵", "💴",
  "💶", "💷", "🪙", "💳", "💎", "⚖️", "🧰", "🔧",
  "🔨", "⚒️", "🛠️", "⛏️", "🪚", "🔩", "⚙️", "🧱",
  "⛓️", "🧲", "🔫", "💣", "🧨", "🪓", "🔪", "🛡️",
  "🚬", "⚰️", "🪦", "⚱️", "🏺", "🔮", "📿", "🧿",
  "💈", "⚗️", "🔭", "🔬", "🩹", "🩺", "💊", "🧪",
  "💉", "🩸", "🧬", "🦠", "🧹", "🧺", "🧻", "🚽",
  "🚿", "🛁", "🧼", "🪥", "🪒", "🧽", "🪣", "🧴",
  "🔑", "🗝️", "🚪", "🪑", "🛋️", "🛏️", "🧸", "🪆",
  "🖼️", "🪞", "🛍️", "🛒", "🎁", "🎈", "🎏", "🎀",
  "🪄", "🪅", "🎊", "🎉", "🎎", "🏮", "🎐", "🧧",
  "✉️", "📩", "📨", "📧", "📥", "📤", "📦", "🏷️",
  "📪", "📫", "📬", "📭", "📮", "📜", "📃", "📄",
  "📑", "🧾", "📊", "📈", "📉", "🗒️", "🗓️", "📅",
  "📆", "🗑️", "📁", "📂", "🗂️", "📰", "📓", "📔",
  "📒", "📕", "📗", "📘", "📙", "📚", "🔖", "📎",
  "🖇️", "📐", "📏", "✂️", "🖊️", "🖋️", "✒️", "🖌️",
  "📝", "✏️", "🔍", "🔎", "🔏", "🔐", "🔒", "🔓",

  "✅", "❌", "❓", "❗", "‼️", "⁉️", "⭕", "🚫",
  "⛔", "📛", "♻️", "⚠️", "🚸", "🔱", "⚜️", "🔰",
  "⬆️", "↗️", "➡️", "↘️", "⬇️", "↙️", "⬅️", "↖️",
  "↕️", "↔️", "🔄", "🔃", "▶️", "⏸️", "⏹️", "⏺️",
  "⏭️", "⏮️", "⏩", "⏪", "🔀", "🔁", "🔂", "➕",
  "➖", "➗", "✖️", "🟰", "♾️", "™️", "©️", "®️",
  "#️⃣", "*️⃣", "0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣",
  "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟", "🔴", "🟠", "🟡",
  "🟢", "🔵", "🟣", "🟤", "⚫", "⚪", "🟥", "🟧",
  "🟨", "🟩", "🟦", "🟪", "🟫", "⬛", "⬜", "🔈",
  "🔉", "🔊", "🔇", "📢", "📣", "🔔", "🔕", "🎵",

  "🏁", "🚩", "🏳️", "🏴", "🏳️‍🌈", "🏳️‍⚧️", "🇨🇳", "🇭🇰",
  "🇲🇴", "🇯🇵", "🇰🇷", "🇸🇬", "🇺🇸", "🇨🇦",
  "🇬🇧", "🇫🇷", "🇩🇪", "🇮🇹", "🇪🇸", "🇵🇹", "🇳🇱", "🇨🇭",
  "🇸🇪", "🇳🇴", "🇫🇮", "🇩🇰", "🇮🇸", "🇦🇺", "🇳🇿", "🇮🇳",
  "🇹🇭", "🇻🇳", "🇵🇭", "🇮🇩", "🇲🇾", "🇧🇷", "🇦🇷", "🇲🇽",
  "🇿🇦", "🇪🇬", "🇦🇪", "🇸🇦", "🇹🇷", "🇬🇷", "🇺🇦", "🇺🇳"
];
const LEGACY_STICKERS = [
  { id: "bright-day", visual: "🌞", label: "元气满满" },
  { id: "great-job", visual: "🙌", label: "太棒了" },
  { id: "got-it", visual: "👌", label: "收到" },
  { id: "keep-going", visual: "💪", label: "继续加油" },
  { id: "many-thanks", visual: "🫶", label: "非常感谢" },
  { id: "wow-moment", visual: "🤯", label: "震惊" }
];
const STORAGE_KEY_MIGRATIONS = [
  ["tmpchat-profile-name", "whisper-profile-name"],
  ["tmpchat-profile-signature", "whisper-profile-signature"],
  ["tmpchat-friend-colors", "whisper-friend-colors"],
  ["tmpchat-pin-sidebar", "whisper-pin-sidebar"],
  ["tmpchat-show-message-time", "whisper-show-message-time"],
  ["tmpchat-full-message-time", "whisper-full-message-time"],
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
  uploadsEnabled: false,
  stickers: [],
  attachmentDrafts: [],
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
let titleScrollTimer = null;
let titleScrollOffset = 0;
let composerSending = false;
let renderedEmojiCount = 0;
let pageAttentionActive = !document.hidden;
const pendingMessageSends = new Map();
let mediaMenuContext = null;
let forwardAttachment = null;
let imageViewerItems = [];
let imageViewerIndex = 0;
let imageViewerScale = 1;
let imageViewerOffsetX = 0;
let imageViewerOffsetY = 0;
let activeMediaElement = null;

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
const contentPickerToggleEl = document.querySelector("#content-picker-toggle");
const contentPickerEl = document.querySelector("#content-picker");
const pickerTabEls = document.querySelectorAll("[data-picker-tab]");
const emojiPanelEl = document.querySelector("#emoji-panel");
const stickerPanelEl = document.querySelector("#sticker-panel");
const filePanelEl = document.querySelector("#file-panel");
const chooseFilesEl = document.querySelector("#choose-files");
const filePanelStatusEl = document.querySelector("#file-panel-status");
const attachmentFileInputEl = document.querySelector("#attachment-file-input");
const stickerFileInputEl = document.querySelector("#sticker-file-input");
const attachmentDraftsEl = document.querySelector("#attachment-drafts");
const composerStatusEl = document.querySelector("#composer-status");
const sendMessageEl = document.querySelector("#send-message");
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
const mediaMenuEl = document.querySelector("#media-menu");
const forwardDialogEl = document.querySelector("#forward-dialog");
const forwardCloseEl = document.querySelector("#forward-close");
const forwardTargetsEl = document.querySelector("#forward-targets");
const forwardStatusEl = document.querySelector("#forward-status");
const imageViewerEl = document.querySelector("#image-viewer");
const imageViewerImageEl = document.querySelector("#image-viewer-image");
const imageViewerStageEl = document.querySelector("#image-viewer-stage");
const imageViewerCountEl = document.querySelector("#image-viewer-count");
const imageViewerCaptionEl = document.querySelector("#image-viewer-caption");
const imageViewerPreviousEl = document.querySelector("#image-viewer-previous");
const imageViewerNextEl = document.querySelector("#image-viewer-next");
const imageViewerCloseEl = document.querySelector("#image-viewer-close");
const imageViewerDownloadEl = document.querySelector("#image-viewer-download");
const imageZoomOutEl = document.querySelector("#image-zoom-out");
const imageZoomResetEl = document.querySelector("#image-zoom-reset");
const imageZoomInEl = document.querySelector("#image-zoom-in");

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
const fullMessageTimeEl = document.querySelector("#full-message-time");
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
  updateDocumentTitle();
}

function setPageAttentionActive(active) {
  pageAttentionActive = active;
  if (!active || unreadCountFor(state.currentConversation) === 0) return;
  clearConversationUnread(state.currentConversation);
  renderConversationNavigation();
}

function shouldMarkConversationUnread(conversationId) {
  return state.currentConversation !== conversationId || !pageAttentionActive;
}

function resetUnreadState() {
  nudgeTimers.forEach((timer) => window.clearTimeout(timer));
  nudgeTimers.clear();
  state.unreadCounts.clear();
  state.nudgingConversations.clear();
  updateDocumentTitle();
}

function updateDocumentTitle() {
  if (totalUnreadCount() === 0) {
    if (titleScrollTimer !== null) {
      window.clearInterval(titleScrollTimer);
      titleScrollTimer = null;
    }
    titleScrollOffset = 0;
    document.title = DEFAULT_DOCUMENT_TITLE;
    return;
  }

  if (titleScrollTimer !== null) return;
  const scrollTitle = () => {
    const offset = titleScrollOffset % NEW_MESSAGE_TITLE.length;
    document.title = NEW_MESSAGE_TITLE.slice(offset) + NEW_MESSAGE_TITLE.slice(0, offset);
    titleScrollOffset += 1;
  };
  scrollTitle();
  titleScrollTimer = window.setInterval(scrollTitle, 200);
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
  updateDocumentTitle();
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

function stickerById(id) {
  return LEGACY_STICKERS.find((sticker) => sticker.id === id) || null;
}

function formatFileSize(size) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function baseContentType(contentType) {
  return String(contentType || "").split(";", 1)[0].trim().toLowerCase();
}

function fileExtension(name) {
  const match = String(name || "").toLowerCase().match(/\.([a-z0-9]+)$/);
  return match ? match[1] : "";
}

function attachmentContentType(file) {
  const reportedType = baseContentType(file.type);
  if (reportedType && reportedType !== "application/octet-stream") {
    return CONTENT_TYPE_ALIASES.get(reportedType) || reportedType;
  }
  return FILE_EXTENSION_CONTENT_TYPES.get(fileExtension(file.name)) || "application/octet-stream";
}

function previewableImageType(contentType) {
  return ["image/gif", "image/jpeg", "image/png", "image/webp"].includes(baseContentType(contentType));
}

function currentAttachmentDrafts() {
  return state.attachmentDrafts.filter((draft) => draft.conversationId === state.currentConversation);
}

function setComposerStatus(message = "") {
  composerStatusEl.textContent = message;
  composerStatusEl.hidden = !message;
}

function draftStatusText(draft) {
  if (draft.status === "uploading") return "上传中...";
  if (draft.status === "ready") return "准备发送";
  if (draft.status === "error") return draft.error || "上传失败";
  return formatFileSize(draft.file.size);
}

function renderAttachmentDrafts() {
  attachmentDraftsEl.replaceChildren();
  const drafts = currentAttachmentDrafts();
  attachmentDraftsEl.hidden = drafts.length === 0;

  drafts.forEach((draft) => {
    const card = document.createElement("div");
    card.className = "attachment-draft";
    card.classList.toggle("error", draft.status === "error");

    let visual;
    if (draft.previewUrl) {
      visual = document.createElement("img");
      visual.className = "attachment-draft-preview";
      visual.src = draft.previewUrl;
      visual.alt = "";
    } else {
      visual = document.createElement("span");
      visual.className = "attachment-draft-icon";
      visual.textContent = "↥";
      visual.setAttribute("aria-hidden", "true");
    }

    const name = document.createElement("span");
    name.className = "attachment-draft-name";
    name.textContent = draft.file.name;
    name.title = draft.file.name;

    const status = document.createElement("span");
    status.className = "attachment-draft-status";
    status.textContent = draftStatusText(draft);

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "attachment-draft-remove";
    remove.textContent = "×";
    remove.disabled = draft.status === "uploading";
    remove.setAttribute("aria-label", `移除文件${draft.file.name}`);
    remove.addEventListener("click", () => { void removeAttachmentDraft(draft); });

    card.append(visual, name, status, remove);
    attachmentDraftsEl.append(card);
  });
}

async function removeAttachmentDraft(draft) {
  if (draft.status === "uploading") return;
  if (draft.attachmentId && backendEnabled) {
    try {
      await apiRequest(`/api/attachments/${draft.attachmentId}`, { method: "DELETE" });
    } catch (error) {
      setComposerStatus(error.message);
      return;
    }
  }
  if (draft.previewUrl) URL.revokeObjectURL(draft.previewUrl);
  state.attachmentDrafts = state.attachmentDrafts.filter((candidate) => candidate !== draft);
  setComposerStatus();
  renderAttachmentDrafts();
  resizeInput();
}

function addAttachmentFiles(files) {
  const drafts = currentAttachmentDrafts();
  const selected = Array.from(files);
  if (drafts.length + selected.length > MAX_ATTACHMENTS_PER_MESSAGE) {
    setComposerStatus(`每条消息最多添加${MAX_ATTACHMENTS_PER_MESSAGE}个文件`);
    return;
  }
  let totalSize = drafts.reduce((total, draft) => total + draft.file.size, 0);
  for (const file of selected) {
    if (file.size <= 0) {
      setComposerStatus(`${file.name} 是空文件`);
      return;
    }
    if (file.size > MAX_ATTACHMENT_SIZE) {
      setComposerStatus(`${file.name} 超过50 MB`);
      return;
    }
    totalSize += file.size;
    if (totalSize > MAX_ATTACHMENT_TOTAL_SIZE) {
      setComposerStatus("每条消息的文件总大小不能超过100 MB");
      return;
    }
  }
  selected.forEach((file) => {
    state.attachmentDrafts.push({
      conversationId: state.currentConversation,
      file,
      status: "selected",
      attachmentId: "",
      previewUrl: previewableImageType(file.type) ? URL.createObjectURL(file) : "",
      error: ""
    });
  });
  setComposerStatus();
  renderAttachmentDrafts();
  resizeInput();
}

function setPickerTab(name, openFileDialog = false) {
  const panels = { emoji: emojiPanelEl, sticker: stickerPanelEl, file: filePanelEl };
  Object.entries(panels).forEach(([panelName, panel]) => {
    panel.hidden = panelName !== name;
  });
  pickerTabEls.forEach((tab) => {
    const selected = tab.dataset.pickerTab === name;
    tab.classList.toggle("active", selected);
    tab.setAttribute("aria-selected", String(selected));
  });
  if (name === "file" && openFileDialog && state.uploadsEnabled) attachmentFileInputEl.click();
}

function setContentPicker(open) {
  contentPickerEl.hidden = !open;
  contentPickerToggleEl.setAttribute("aria-expanded", String(open));
  if (open) setPickerTab("emoji");
}

function insertEmoji(emoji) {
  const start = inputEl.selectionStart ?? inputEl.value.length;
  const end = inputEl.selectionEnd ?? start;
  inputEl.setRangeText(emoji, start, end, "end");
  inputEl.dispatchEvent(new Event("input", { bubbles: true }));
  inputEl.focus();
}

function appendEmojiOptions() {
  const nextEmojiCount = Math.min(renderedEmojiCount + EMOJI_BATCH_SIZE, EMOJI_OPTIONS.length);
  EMOJI_OPTIONS.slice(renderedEmojiCount, nextEmojiCount).forEach((emoji) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "emoji-option";
    button.textContent = emoji;
    button.setAttribute("aria-label", `插入${emoji}`);
    button.addEventListener("click", () => insertEmoji(emoji));
    emojiPanelEl.append(button);
  });
  renderedEmojiCount = nextEmojiCount;
}

function renderContentPickerOptions() {
  emojiPanelEl.replaceChildren();
  renderedEmojiCount = 0;
  appendEmojiOptions();

  stickerPanelEl.replaceChildren();
  const addButton = document.createElement("button");
  addButton.type = "button";
  addButton.className = "sticker-option sticker-add";
  addButton.setAttribute("aria-label", "添加表情包");
  addButton.title = "添加表情包";
  const addIcon = document.createElement("span");
  addIcon.className = "sticker-add-icon";
  addIcon.textContent = "+";
  addIcon.setAttribute("aria-hidden", "true");
  addButton.append(addIcon);
  addButton.addEventListener("click", () => stickerFileInputEl.click());
  stickerPanelEl.append(addButton);

  state.stickers.forEach((sticker) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "sticker-option";
    button.setAttribute("aria-label", `发送表情包${sticker.name}`);
    button.title = sticker.name;
    const visual = document.createElement("img");
    visual.className = "sticker-option-image";
    visual.src = sticker.url;
    visual.alt = "";
    button.append(visual);
    button.addEventListener("click", () => { void sendStickerAttachment(sticker); });
    bindMediaContextMenu(button, sticker, null, true);
    stickerPanelEl.append(button);
  });
}

function updateUploadControls() {
  chooseFilesEl.disabled = !state.uploadsEnabled;
  document.querySelector("#file-tab").disabled = !state.uploadsEnabled;
  filePanelStatusEl.textContent = state.uploadsEnabled
    ? `单文件最大50 MB，每条消息最多${MAX_ATTACHMENTS_PER_MESSAGE}个文件`
    : "服务器尚未配置文件存储";
  renderContentPickerOptions();
}

function switchConversation(conversationId) {
  const groupSettingsId = groupIdFromSettings(conversationId);
  const allowedPanel = conversationId === "self" || conversationId === "group-create"
    || (groupSettingsId && state.groups.has(groupSettingsId));
  if (!allowedPanel && !state.conversations[conversationId]) return;

  clearConversationUnread(conversationId);
  state.currentConversation = conversationId;
  inputEl.value = "";
  setContentPicker(false);
  setComposerStatus();
  renderAttachmentDrafts();
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
    const defaultsToFullTime = fullMessageTimeEl.checked;
    time.className = "message-time";
    time.dateTime = message.sentAt;
    time.setAttribute("aria-label", fullTime);

    const clockEl = document.createElement("span");
    clockEl.className = "message-clock";
    clockEl.textContent = clock;
    const compactTimeChildren = [clockEl];
    const fullTimeEl = document.createElement("span");
    fullTimeEl.className = "message-full-time";
    fullTimeEl.textContent = fullTime;

    if (context) {
      const contextEl = document.createElement("span");
      contextEl.className = "message-date-context";
      contextEl.textContent = context;
      compactTimeChildren.push(contextEl);
    }

    const showFullTime = () => {
      time.replaceChildren(fullTimeEl);
      time.classList.add("expanded");
      meta.classList.add("time-expanded");
    };
    const showCompactTime = () => {
      time.replaceChildren(...compactTimeChildren);
      time.classList.remove("expanded");
      meta.classList.remove("time-expanded");
    };
    const showDefaultTime = defaultsToFullTime ? showFullTime : showCompactTime;
    const showAlternateTime = defaultsToFullTime ? showCompactTime : showFullTime;

    showDefaultTime();
    time.addEventListener("mouseenter", showAlternateTime);
    time.addEventListener("mouseleave", showDefaultTime);
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

function normalizedAttachmentKind(attachment) {
  if (attachment.kind) return attachment.kind;
  const contentType = baseContentType(attachment.contentType);
  if (previewableImageType(contentType)) return "image";
  if (VIDEO_CONTENT_TYPES.has(contentType)) return "video";
  if (AUDIO_CONTENT_TYPES.has(contentType)) return "audio";
  if (DOCUMENT_CONTENT_TYPES.has(contentType)) return "document";
  return "file";
}

function openAttachmentDownload(attachment) {
  const separator = attachment.url.includes("?") ? "&" : "?";
  window.open(`${attachment.url}${separator}download=1`, "_blank", "noopener,noreferrer");
}

function openAttachmentPreview(attachment) {
  window.open(attachment.url, "_blank", "noopener,noreferrer");
}

function previewAttachment(attachment) {
  if (normalizedAttachmentKind(attachment) === "image") openImageViewer(attachment);
  else openAttachmentPreview(attachment);
}

function createMediaMoreButton(attachment, message, collectedSticker = false) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "media-more-button";
  button.textContent = "⋮";
  button.setAttribute("aria-label", `${attachment.name}的更多操作`);
  button.title = "更多操作";
  button.addEventListener("click", (event) => {
    event.stopPropagation();
    const rect = button.getBoundingClientRect();
    showMediaMenu(rect.right, rect.bottom + 4, { attachment, message, collectedSticker });
  });
  return button;
}

function bindMediaContextMenu(element, attachment, message, collectedSticker = false) {
  element.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    showMediaMenu(event.clientX, event.clientY, { attachment, message, collectedSticker });
  });
  let longPressTimer = null;
  let longPressTriggered = false;
  element.addEventListener("pointerdown", (event) => {
    if (event.pointerType !== "touch") return;
    longPressTriggered = false;
    longPressTimer = window.setTimeout(() => {
      longPressTriggered = true;
      showMediaMenu(event.clientX, event.clientY, { attachment, message, collectedSticker });
      longPressTimer = null;
    }, 550);
  });
  ["pointerup", "pointercancel", "pointermove"].forEach((eventName) => {
    element.addEventListener(eventName, () => {
      if (longPressTimer !== null) window.clearTimeout(longPressTimer);
      longPressTimer = null;
    });
  });
  element.addEventListener("click", (event) => {
    if (!longPressTriggered) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    longPressTriggered = false;
  }, true);
}

function createInlineImage(attachment, message) {
  const item = document.createElement("figure");
  item.className = "message-image";
  const button = document.createElement("button");
  button.type = "button";
  button.className = "message-image-open";
  button.setAttribute("aria-label", `查看图片${attachment.name}`);
  const image = document.createElement("img");
  image.src = attachment.url;
  image.alt = attachment.name;
  image.loading = "lazy";
  image.decoding = "async";
  button.append(image);
  button.addEventListener("click", () => openImageViewer(attachment));
  const caption = document.createElement("figcaption");
  const name = document.createElement("span");
  name.textContent = attachment.name;
  name.title = attachment.name;
  caption.append(name, createMediaMoreButton(attachment, message));
  item.append(button, caption);
  bindMediaContextMenu(item, attachment, message);
  return item;
}

function formatMediaTime(seconds) {
  if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
  const total = Math.floor(seconds);
  const minutes = Math.floor(total / 60);
  return `${minutes}:${String(total % 60).padStart(2, "0")}`;
}

function mediaPlaybackContentType(contentType) {
  switch (baseContentType(contentType)) {
    case "audio/mp3": return "audio/mpeg";
    case "audio/x-m4a": return "audio/mp4";
    case "audio/x-aac": return "audio/aac";
    case "audio/x-flac": return "audio/flac";
    case "audio/x-wav": return "audio/wav";
    case "video/x-m4v": return "video/mp4";
    default: return baseContentType(contentType);
  }
}

function createPlaybackRateControl(media, label = "播放速度") {
  const rate = document.createElement("select");
  rate.className = "media-rate";
  rate.setAttribute("aria-label", label);
  rate.title = label;
  [0.75, 1, 1.25, 1.5, 2].forEach((value) => {
    const option = document.createElement("option");
    option.value = String(value);
    option.textContent = `${value}×`;
    if (value === 1) option.selected = true;
    rate.append(option);
  });
  rate.addEventListener("change", () => {
    media.playbackRate = Number(rate.value);
  });
  return rate;
}

function activateMedia(media) {
  if (activeMediaElement && activeMediaElement !== media) activeMediaElement.pause();
  activeMediaElement = media;
}

function releaseMedia(media) {
  if (activeMediaElement === media) activeMediaElement = null;
}

function mediaCanPlay(media, attachment) {
  const contentType = mediaPlaybackContentType(attachment.contentType);
  return !contentType || media.canPlayType(contentType) !== "";
}

function createVideoPlayer(attachment, message) {
  const player = document.createElement("div");
  player.className = "message-video";
  const video = document.createElement("video");
  video.preload = "metadata";
  video.playsInline = true;
  video.controls = false;
  video.setAttribute("aria-label", attachment.name);
  if (!mediaCanPlay(video, attachment)) {
    return createFileCard(attachment, message, "当前浏览器不支持在线播放");
  }

  const controls = document.createElement("div");
  controls.className = "video-controls";
  const play = document.createElement("button");
  play.type = "button";
  play.className = "video-play";
  play.textContent = "▶";
  play.setAttribute("aria-label", "播放");
  play.title = "播放";
  const timeline = document.createElement("input");
  timeline.className = "video-timeline";
  timeline.type = "range";
  timeline.min = "0";
  timeline.max = "1000";
  timeline.value = "0";
  timeline.setAttribute("aria-label", "播放进度");
  const time = document.createElement("span");
  time.className = "video-time";
  time.textContent = "0:00 / 0:00";
  const mute = document.createElement("button");
  mute.type = "button";
  mute.className = "video-mute";
  mute.textContent = "♪";
  mute.setAttribute("aria-label", "静音");
  mute.title = "静音";
  const volume = document.createElement("input");
  volume.className = "video-volume";
  volume.type = "range";
  volume.min = "0";
  volume.max = "1";
  volume.step = "0.05";
  volume.value = "1";
  volume.setAttribute("aria-label", "音量");
  const rate = createPlaybackRateControl(video);
  const pip = document.createElement("button");
  pip.type = "button";
  pip.className = "video-pip";
  pip.textContent = "▣";
  pip.setAttribute("aria-label", "画中画");
  pip.title = "画中画";
  const pipSupported = Boolean(document.pictureInPictureEnabled)
    && typeof video.requestPictureInPicture === "function";
  pip.hidden = !pipSupported;
  const fullscreen = document.createElement("button");
  fullscreen.type = "button";
  fullscreen.className = "video-fullscreen";
  fullscreen.textContent = "⛶";
  fullscreen.setAttribute("aria-label", "全屏");
  fullscreen.title = "全屏";
  const more = createMediaMoreButton(attachment, message);
  controls.append(play, timeline, time, mute, volume, rate, pip, fullscreen, more);
  player.append(video, controls);

  const showFallback = () => {
    if (player.parentNode) player.replaceWith(createFileCard(attachment, message, "当前浏览器不支持在线播放"));
  };
  const updatePlaybackState = () => {
    const playing = !video.paused && !video.ended;
    play.textContent = playing ? "Ⅱ" : "▶";
    play.setAttribute("aria-label", playing ? "暂停" : "播放");
    play.title = playing ? "暂停" : "播放";
  };
  const togglePlayback = async () => {
    if (video.paused || video.ended) {
      try { await video.play(); } catch (_) {}
    } else {
      video.pause();
    }
  };
  play.addEventListener("click", () => { void togglePlayback(); });
  video.addEventListener("click", () => { void togglePlayback(); });
  video.addEventListener("play", () => { activateMedia(video); updatePlaybackState(); });
  video.addEventListener("pause", () => { releaseMedia(video); updatePlaybackState(); });
  video.addEventListener("ended", () => { releaseMedia(video); updatePlaybackState(); });
  video.addEventListener("error", showFallback);
  video.addEventListener("timeupdate", () => {
    timeline.value = video.duration ? String(Math.round((video.currentTime / video.duration) * 1000)) : "0";
    time.textContent = `${formatMediaTime(video.currentTime)} / ${formatMediaTime(video.duration)}`;
  });
  video.addEventListener("loadedmetadata", () => {
    time.textContent = `0:00 / ${formatMediaTime(video.duration)}`;
  });
  timeline.addEventListener("input", () => {
    if (video.duration) video.currentTime = (Number(timeline.value) / 1000) * video.duration;
  });
  volume.addEventListener("input", () => {
    video.volume = Number(volume.value);
    video.muted = video.volume === 0;
    mute.textContent = video.muted ? "×" : "♪";
  });
  mute.addEventListener("click", () => {
    video.muted = !video.muted;
    mute.textContent = video.muted ? "×" : "♪";
    mute.setAttribute("aria-label", video.muted ? "取消静音" : "静音");
  });
  pip.addEventListener("click", async () => {
    if (!pipSupported) return;
    try {
      if (document.pictureInPictureElement === video) await document.exitPictureInPicture();
      else await video.requestPictureInPicture();
    } catch (_) {}
  });
  video.addEventListener("enterpictureinpicture", () => {
    pip.textContent = "▣";
    pip.setAttribute("aria-label", "退出画中画");
    pip.title = "退出画中画";
  });
  video.addEventListener("leavepictureinpicture", () => {
    pip.setAttribute("aria-label", "画中画");
    pip.title = "画中画";
  });
  fullscreen.addEventListener("click", () => {
    if (document.fullscreenElement) void document.exitFullscreen();
    else if (player.requestFullscreen) void player.requestFullscreen();
  });
  video.src = attachment.url;
  bindMediaContextMenu(player, attachment, message);
  return player;
}

function attachmentBadge(attachment, kind) {
  const extension = fileExtension(attachment.name).toUpperCase();
  if (kind === "document") return extension || "DOC";
  if (kind === "audio") return "♫";
  if (kind === "video") return "▶";
  if (kind === "image") return "▧";
  return "↥";
}

function createAudioPlayer(attachment, message) {
  const player = document.createElement("div");
  player.className = "message-audio";
  const audio = document.createElement("audio");
  audio.preload = "metadata";
  audio.controls = false;
  if (!mediaCanPlay(audio, attachment)) {
    return createFileCard(attachment, message, "当前浏览器不支持在线播放");
  }

  const summary = document.createElement("div");
  summary.className = "audio-summary";
  const icon = document.createElement("span");
  icon.className = "audio-file-icon";
  icon.textContent = "♫";
  icon.setAttribute("aria-hidden", "true");
  const name = document.createElement("span");
  name.className = "audio-file-name";
  name.textContent = attachment.name;
  name.title = attachment.name;
  const format = document.createElement("span");
  format.className = "audio-file-format";
  format.textContent = fileExtension(attachment.name).toUpperCase() || "音频";
  const download = document.createElement("button");
  download.type = "button";
  download.className = "audio-download";
  download.textContent = "↓";
  download.setAttribute("aria-label", `下载音频${attachment.name}`);
  download.title = "下载";
  download.addEventListener("click", () => openAttachmentDownload(attachment));
  summary.append(icon, name, format, download, createMediaMoreButton(attachment, message));

  const controls = document.createElement("div");
  controls.className = "audio-controls";
  const play = document.createElement("button");
  play.type = "button";
  play.className = "audio-play";
  play.textContent = "▶";
  play.setAttribute("aria-label", "播放");
  play.title = "播放";
  const timeline = document.createElement("input");
  timeline.className = "audio-timeline";
  timeline.type = "range";
  timeline.min = "0";
  timeline.max = "1000";
  timeline.value = "0";
  timeline.setAttribute("aria-label", "播放进度");
  const time = document.createElement("span");
  time.className = "audio-time";
  time.textContent = "0:00 / 0:00";
  const mute = document.createElement("button");
  mute.type = "button";
  mute.className = "audio-mute";
  mute.textContent = "♪";
  mute.setAttribute("aria-label", "静音");
  mute.title = "静音";
  const volume = document.createElement("input");
  volume.className = "audio-volume";
  volume.type = "range";
  volume.min = "0";
  volume.max = "1";
  volume.step = "0.05";
  volume.value = "1";
  volume.setAttribute("aria-label", "音量");
  const rate = createPlaybackRateControl(audio);
  controls.append(play, timeline, time, mute, volume, rate);
  player.append(summary, audio, controls);

  const showFallback = () => {
    if (player.parentNode) player.replaceWith(createFileCard(attachment, message, "当前浏览器不支持在线播放"));
  };
  const updatePlaybackState = () => {
    const playing = !audio.paused && !audio.ended;
    play.textContent = playing ? "Ⅱ" : "▶";
    play.setAttribute("aria-label", playing ? "暂停" : "播放");
    play.title = playing ? "暂停" : "播放";
  };
  const togglePlayback = async () => {
    if (audio.paused || audio.ended) {
      try { await audio.play(); } catch (_) {}
    } else {
      audio.pause();
    }
  };
  play.addEventListener("click", () => { void togglePlayback(); });
  audio.addEventListener("play", () => { activateMedia(audio); updatePlaybackState(); });
  audio.addEventListener("pause", () => { releaseMedia(audio); updatePlaybackState(); });
  audio.addEventListener("ended", () => { releaseMedia(audio); updatePlaybackState(); });
  audio.addEventListener("error", showFallback);
  audio.addEventListener("timeupdate", () => {
    timeline.value = audio.duration ? String(Math.round((audio.currentTime / audio.duration) * 1000)) : "0";
    time.textContent = `${formatMediaTime(audio.currentTime)} / ${formatMediaTime(audio.duration)}`;
  });
  audio.addEventListener("loadedmetadata", () => {
    time.textContent = `0:00 / ${formatMediaTime(audio.duration)}`;
  });
  timeline.addEventListener("input", () => {
    if (audio.duration) audio.currentTime = (Number(timeline.value) / 1000) * audio.duration;
  });
  volume.addEventListener("input", () => {
    audio.volume = Number(volume.value);
    audio.muted = audio.volume === 0;
    mute.textContent = audio.muted ? "×" : "♪";
  });
  mute.addEventListener("click", () => {
    audio.muted = !audio.muted;
    mute.textContent = audio.muted ? "×" : "♪";
    mute.setAttribute("aria-label", audio.muted ? "取消静音" : "静音");
  });
  audio.src = attachment.url;
  bindMediaContextMenu(player, attachment, message);
  return player;
}

function createFileCard(attachment, message, note = "") {
  const card = document.createElement("div");
  card.className = "message-file";
  const kind = normalizedAttachmentKind(attachment);
  const icon = document.createElement("span");
  icon.className = "message-file-icon";
  if (kind === "document") icon.classList.add("document-badge");
  icon.textContent = attachmentBadge(attachment, kind);
  icon.setAttribute("aria-hidden", "true");
  const name = document.createElement("span");
  name.className = "message-file-name";
  name.textContent = attachment.name;
  name.title = attachment.name;
  const size = document.createElement("span");
  size.className = "message-file-size";
  size.textContent = note ? `${formatFileSize(attachment.size)} · ${note}` : formatFileSize(attachment.size);
  const actions = document.createElement("span");
  actions.className = "message-file-actions";
  if (kind === "image" || kind === "document") {
    const preview = document.createElement("button");
    preview.type = "button";
    preview.textContent = kind === "document" ? "↗" : "⌕";
    preview.setAttribute("aria-label", `${kind === "document" ? "打开文档" : "查看图片"}${attachment.name}`);
    preview.title = kind === "document" ? "打开" : "查看";
    preview.addEventListener("click", () => previewAttachment(attachment));
    actions.append(preview);
  }
  const download = document.createElement("button");
  download.type = "button";
  download.textContent = "↓";
  download.setAttribute("aria-label", `下载文件${attachment.name}`);
  download.title = "下载";
  download.addEventListener("click", () => openAttachmentDownload(attachment));
  actions.append(download, createMediaMoreButton(attachment, message));
  card.append(icon, name, size, actions);
  bindMediaContextMenu(card, attachment, message);
  return card;
}

function createMessageAttachment(attachment, message) {
  const kind = normalizedAttachmentKind(attachment);
  const inlineImage = attachment.inline === true
    || (attachment.inline == null && attachment.size <= MAX_STICKER_SIZE);
  if (kind === "image" && inlineImage) return createInlineImage(attachment, message);
  if (kind === "video") return createVideoPlayer(attachment, message);
  if (kind === "audio") return createAudioPlayer(attachment, message);
  return createFileCard(attachment, message);
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

    const content = document.createElement("div");
    content.className = "message-content";
    const friendColor = state.friendColors.get(message.from);
    if (message.recalled) {
      const recalled = document.createElement("p");
      recalled.className = "message-recalled";
      recalled.textContent = "消息已撤回";
      content.append(recalled);
    } else if (message.text) {
      const text = document.createElement("p");
      text.className = "text";
      if (!message.system && message.from !== state.currentUser && friendColor) {
        text.dataset.friendColor = friendColor;
      }
      appendFormattedText(text, message.text);
      content.append(text);
    }

    const sticker = stickerById(message.sticker);
    if (sticker) {
      const stickerEl = document.createElement("span");
      stickerEl.className = "message-sticker";
      stickerEl.textContent = sticker.visual;
      stickerEl.setAttribute("role", "img");
      stickerEl.setAttribute("aria-label", sticker.label);
      content.append(stickerEl);
    }

    if (!message.recalled && message.stickerAttachment) {
      const stickerItem = document.createElement("span");
      stickerItem.className = "message-sticker-item";
      const stickerOpen = document.createElement("button");
      stickerOpen.type = "button";
      stickerOpen.className = "message-sticker-open";
      stickerOpen.setAttribute("aria-label", `查看表情包${message.stickerAttachment.name}`);
      const stickerImage = document.createElement("img");
      stickerImage.className = "message-sticker-image";
      stickerImage.src = message.stickerAttachment.url;
      stickerImage.alt = message.stickerAttachment.name;
      stickerImage.loading = "lazy";
      stickerImage.decoding = "async";
      stickerOpen.append(stickerImage);
      stickerOpen.addEventListener("click", () => openImageViewer(message.stickerAttachment));
      stickerItem.append(
        stickerOpen,
        createMediaMoreButton(message.stickerAttachment, message)
      );
      bindMediaContextMenu(stickerItem, message.stickerAttachment, message);
      content.append(stickerItem);
    }

    if (!message.recalled && message.attachments?.length) {
      const attachments = document.createElement("div");
      attachments.className = "message-attachments";
      message.attachments.forEach((attachment) => attachments.append(createMessageAttachment(attachment, message)));
      content.append(attachments);
    }

    if (message.delivery === "queued") {
      const delivery = document.createElement("span");
      delivery.className = "message-delivery";
      delivery.textContent = "待送达";
      content.append(delivery);
    }

    row.append(createMessageMeta(message, canMention), content);
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

function setComposerSending(sending) {
  composerSending = sending;
  sendMessageEl.disabled = sending;
  contentPickerToggleEl.disabled = sending;
  renderAttachmentDrafts();
}

async function uploadAttachmentDraft(draft) {
  if (draft.status === "ready" && draft.attachmentId) return draft.attachmentId;
  if (draft.attachmentId) {
    try {
      await apiRequest(`/api/attachments/${draft.attachmentId}`, { method: "DELETE" });
    } catch (_) {}
    draft.attachmentId = "";
  }
  draft.status = "uploading";
  draft.error = "";
  renderAttachmentDrafts();
  try {
    const contentType = attachmentContentType(draft.file);
    const presigned = await apiRequest("/api/attachments/presign", {
      method: "POST",
      body: { name: draft.file.name, size: draft.file.size, contentType }
    });
    draft.attachmentId = presigned.attachmentId;
    const uploadResponse = await fetch(presigned.uploadUrl, {
      method: "PUT",
      headers: presigned.headers,
      body: draft.file
    });
    if (!uploadResponse.ok) throw new Error(`文件上传失败（${uploadResponse.status}）`);
    await apiRequest("/api/attachments/complete", {
      method: "POST", body: { attachmentId: draft.attachmentId }
    });
    draft.status = "ready";
    renderAttachmentDrafts();
    return draft.attachmentId;
  } catch (error) {
    draft.status = "error";
    draft.error = error.message || "上传失败";
    renderAttachmentDrafts();
    throw error;
  }
}

function setStickerPanelStatus(message = "", error = false) {
  stickerPanelEl.querySelector(".sticker-panel-status")?.remove();
  if (!message) return;
  const status = document.createElement("span");
  status.className = "picker-panel-status sticker-panel-status";
  status.classList.toggle("error", error);
  status.textContent = message;
  status.setAttribute("role", "status");
  stickerPanelEl.append(status);
}

async function addStickerFile(file) {
  if (!previewableImageType(file.type)) {
    setStickerPanelStatus("表情包仅支持 GIF、JPEG、PNG 和 WebP", true);
    return false;
  }
  if (file.size <= 0 || file.size > MAX_STICKER_SIZE) {
    setStickerPanelStatus("表情包文件不能超过10 MB", true);
    return false;
  }
  const draft = { file, status: "selected", attachmentId: "", error: "" };
  setStickerPanelStatus("正在上传...");
  try {
    const attachmentId = await uploadAttachmentDraft(draft);
    const sticker = await apiRequest("/api/stickers", {
      method: "POST", body: { attachmentId }
    });
    if (!state.stickers.some((item) => item.id === sticker.id)) state.stickers.push(sticker);
    renderContentPickerOptions();
    setPickerTab("sticker");
    setStickerPanelStatus("已添加");
    return true;
  } catch (error) {
    if (draft.attachmentId) {
      try { await apiRequest(`/api/attachments/${draft.attachmentId}`, { method: "DELETE" }); } catch (_) {}
    }
    renderContentPickerOptions();
    setPickerTab("sticker");
    setStickerPanelStatus(error.message || "添加失败", true);
    return false;
  }
}

async function addStickerFiles(files) {
  const selected = Array.from(files);
  if (selected.length === 0) return;
  setPickerTab("sticker");
  if (!state.uploadsEnabled || !backendEnabled) {
    setStickerPanelStatus("已选择图片，但服务器尚未配置文件存储，暂时无法上传", true);
    return;
  }
  const remaining = Math.max(0, 120 - state.stickers.length);
  if (remaining === 0) {
    setStickerPanelStatus("最多收藏120个表情包", true);
    return;
  }
  const accepted = selected.slice(0, remaining);
  let added = 0;
  for (let index = 0; index < accepted.length; index += 1) {
    setStickerPanelStatus(`正在添加 ${index + 1} / ${accepted.length}...`);
    if (await addStickerFile(accepted[index])) added += 1;
  }
  if (selected.length > accepted.length) {
    setStickerPanelStatus(`已添加${added}张，表情包总数不能超过120张`, true);
    return;
  }
  setStickerPanelStatus(
    added === accepted.length ? `已添加${added}张` : `成功添加${added} / ${accepted.length}张`,
    added !== accepted.length
  );
}

async function favoriteSticker(attachment) {
  try {
    const sticker = await apiRequest("/api/stickers/favorite", {
      method: "POST", body: { attachmentId: attachment.id }
    });
    if (!state.stickers.some((item) => item.id === sticker.id)) state.stickers.push(sticker);
    renderContentPickerOptions();
    setComposerStatus("已收藏为表情包");
  } catch (error) {
    setComposerStatus(error.message || "收藏失败");
  }
}

async function removeSticker(attachment) {
  try {
    await apiRequest(`/api/stickers/${attachment.id}`, { method: "DELETE" });
    state.stickers = state.stickers.filter((item) => item.id !== attachment.id);
    renderContentPickerOptions();
    setPickerTab("sticker");
  } catch (error) {
    setStickerPanelStatus(error.message || "移除失败", true);
  }
}

function messageDestination() {
  const member = currentMember();
  const group = currentGroup();
  return {
    conversationId: state.currentConversation,
    member,
    scope: member ? "private" : "group",
    to: member?.name || group?.id || PUBLIC_GROUP_ID
  };
}

function messageRequestID() {
  if (typeof crypto.randomUUID === "function") return crypto.randomUUID();
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function sendSocketCommand(command) {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    inputEl.placeholder = "正在重新连接...";
    throw new Error("连接已断开，请稍后重试");
  }
  const requestId = messageRequestID();
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      settlePendingMessage(requestId, new Error("操作确认超时，请检查连接后重试"));
    }, 15000);
    pendingMessageSends.set(requestId, { resolve, reject, timer });
    try {
      socket.send(JSON.stringify({ ...command, requestId }));
    } catch (error) {
      settlePendingMessage(requestId, error);
    }
  });
}

function settlePendingMessage(requestID, error = null) {
  if (!requestID) return;
  const pending = pendingMessageSends.get(requestID);
  if (!pending) return;
  window.clearTimeout(pending.timer);
  pendingMessageSends.delete(requestID);
  if (error) pending.reject(error);
  else pending.resolve();
}

function rejectPendingMessages(message) {
  [...pendingMessageSends.keys()].forEach((requestID) => {
    settlePendingMessage(requestID, new Error(message));
  });
}

function sendPreparedMessage(content, destination) {
  if (backendEnabled) {
    return sendSocketCommand({
      type: "message", scope: destination.scope, to: destination.to, ...content
    });
  }

  state.conversations[destination.conversationId].push({
    from: state.currentUser,
    text: content.text || "",
    sticker: content.sticker || "",
    attachments: [],
    sentAt: new Date().toISOString(),
    delivery: destination.member && !destination.member.online ? "queued" : "sent"
  });
  if (destination.member) state.friends.add(destination.member.name);
  renderMessages();
  renderConversationNavigation();
  window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" });
  return Promise.resolve();
}

function clearSentAttachmentDrafts(conversationId) {
  const sentDrafts = state.attachmentDrafts.filter((draft) => draft.conversationId === conversationId);
  sentDrafts.forEach((draft) => {
    if (draft.previewUrl) URL.revokeObjectURL(draft.previewUrl);
  });
  state.attachmentDrafts = state.attachmentDrafts.filter((draft) => draft.conversationId !== conversationId);
  renderAttachmentDrafts();
}

function clearLocalAttachmentDrafts() {
  state.attachmentDrafts.forEach((draft) => {
    if (draft.previewUrl) URL.revokeObjectURL(draft.previewUrl);
  });
  state.attachmentDrafts = [];
  renderAttachmentDrafts();
}

function reconcileAttachmentDrafts() {
  const attachedIDs = new Set();
  Object.values(state.conversations).forEach((messages) => {
    messages.forEach((message) => {
      message.attachments?.forEach((attachment) => attachedIDs.add(attachment.id));
    });
  });
  state.attachmentDrafts = state.attachmentDrafts.filter((draft) => {
    if (!draft.attachmentId || !attachedIDs.has(draft.attachmentId)) return true;
    if (draft.previewUrl) URL.revokeObjectURL(draft.previewUrl);
    return false;
  });
  renderAttachmentDrafts();
}

async function sendCurrentMessage() {
  if (state.currentConversation === "self" || isGroupCreateConversation() || isGroupSettingsConversation()) return;
  if (composerSending) return;
  const text = inputEl.value.trim();
  const drafts = currentAttachmentDrafts();
  if (!text && drafts.length === 0) return;
  if (drafts.length > 0 && (!backendEnabled || !state.uploadsEnabled)) {
    setComposerStatus("服务器尚未配置文件上传");
    return;
  }

  const destination = messageDestination();
  setComposerStatus();
  setComposerSending(true);
  try {
    const attachmentIds = await Promise.all(drafts.map(uploadAttachmentDraft));
    await sendPreparedMessage({ text, attachmentIds }, destination);
    inputEl.value = "";
    clearSentAttachmentDrafts(destination.conversationId);
    resizeInput();
  } catch (error) {
    setComposerStatus(error.message || "发送失败");
  } finally {
    setComposerSending(false);
    inputEl.focus();
  }
}

async function sendSticker(stickerID) {
  if (composerSending) return;
  const sticker = stickerById(stickerID);
  if (!sticker || state.currentConversation === "self"
    || isGroupCreateConversation() || isGroupSettingsConversation()) return;
  const destination = messageDestination();
  setComposerSending(true);
  try {
    await sendPreparedMessage({ text: "", sticker: stickerID }, destination);
    setContentPicker(false);
    setComposerStatus();
  } catch (error) {
    setComposerStatus(error.message || "发送失败");
  } finally {
    setComposerSending(false);
    inputEl.focus();
  }
}

async function sendStickerAttachment(sticker) {
  if (composerSending || !backendEnabled) return;
  if (state.currentConversation === "self"
    || isGroupCreateConversation() || isGroupSettingsConversation()) return;
  const destination = messageDestination();
  setComposerSending(true);
  try {
    await sendPreparedMessage({ text: "", stickerAttachmentId: sticker.id }, destination);
    setContentPicker(false);
    setComposerStatus();
  } catch (error) {
    setComposerStatus(error.message || "发送失败");
  } finally {
    setComposerSending(false);
    inputEl.focus();
  }
}

async function recallMessage(message) {
  if (!backendEnabled || !message?.id || !message.canRecall) return;
  try {
    await sendSocketCommand({ type: "recall", messageId: message.id });
  } catch (error) {
    setComposerStatus(error.message || "撤回失败");
  }
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
  if (shouldMarkConversationUnread(conversationId)) {
    markConversationUnread(conversationId);
  }
  renderConversationNavigation();
  if (state.currentConversation === conversationId) renderMessages();
}

function hideMediaMenu() {
  mediaMenuEl.hidden = true;
  mediaMenuContext = null;
}

function showMediaMenu(x, y, context) {
  hideFriendMenu();
  hideGroupMenu();
  mediaMenuContext = context;
  const { attachment, message, collectedSticker } = context;
  const kind = normalizedAttachmentKind(attachment);
  const isCollected = state.stickers.some((sticker) => sticker.id === attachment.id);
  const availability = {
    preview: kind === "image" || kind === "document",
    collect: backendEnabled && kind === "image" && !isCollected && !collectedSticker,
    download: backendEnabled,
    forward: backendEnabled && Boolean(message?.attachments?.some((item) => item.id === attachment.id)),
    recall: backendEnabled && Boolean(message?.canRecall),
    "remove-sticker": backendEnabled && Boolean(collectedSticker)
  };
  mediaMenuEl.querySelectorAll("[data-media-action]").forEach((button) => {
    button.hidden = !availability[button.dataset.mediaAction];
    if (button.dataset.mediaAction === "preview") button.textContent = kind === "document" ? "打开" : "查看";
  });
  mediaMenuEl.hidden = false;
  const rect = mediaMenuEl.getBoundingClientRect();
  mediaMenuEl.style.left = `${Math.max(8, Math.min(x - rect.width, window.innerWidth - rect.width - 8))}px`;
  mediaMenuEl.style.top = `${Math.max(8, Math.min(y, window.innerHeight - rect.height - 8))}px`;
}

function imageAttachmentsForCurrentConversation(extraAttachment = null) {
  const images = [];
  const seen = new Set();
  const add = (attachment) => {
    if (!attachment || normalizedAttachmentKind(attachment) !== "image" || seen.has(attachment.id)) return;
    seen.add(attachment.id);
    images.push(attachment);
  };
  currentMessages().forEach((message) => {
    if (message.recalled) return;
    add(message.stickerAttachment);
    message.attachments?.forEach(add);
  });
  add(extraAttachment);
  return images;
}

function applyImageViewerTransform() {
  imageViewerImageEl.style.transform = `translate(${imageViewerOffsetX}px, ${imageViewerOffsetY}px) scale(${imageViewerScale})`;
  imageZoomResetEl.textContent = `${Math.round(imageViewerScale * 100)}%`;
}

function resetImageViewerTransform() {
  imageViewerScale = 1;
  imageViewerOffsetX = 0;
  imageViewerOffsetY = 0;
  applyImageViewerTransform();
}

function renderImageViewer() {
  const attachment = imageViewerItems[imageViewerIndex];
  if (!attachment) {
    closeImageViewer();
    return;
  }
  imageViewerImageEl.src = attachment.url;
  imageViewerImageEl.alt = attachment.name;
  imageViewerCaptionEl.textContent = `${attachment.name} · ${formatFileSize(attachment.size)}`;
  imageViewerCountEl.textContent = `${imageViewerIndex + 1} / ${imageViewerItems.length}`;
  imageViewerPreviousEl.hidden = imageViewerItems.length < 2;
  imageViewerNextEl.hidden = imageViewerItems.length < 2;
  resetImageViewerTransform();
  const previous = imageViewerItems[(imageViewerIndex - 1 + imageViewerItems.length) % imageViewerItems.length];
  const next = imageViewerItems[(imageViewerIndex + 1) % imageViewerItems.length];
  [previous, next].forEach((item) => {
    if (!item || item.id === attachment.id) return;
    const preload = new Image();
    preload.src = item.url;
  });
}

function openImageViewer(attachment) {
  imageViewerItems = imageAttachmentsForCurrentConversation(attachment);
  imageViewerIndex = Math.max(0, imageViewerItems.findIndex((item) => item.id === attachment.id));
  imageViewerEl.hidden = false;
  document.body.classList.add("modal-open");
  renderImageViewer();
  imageViewerCloseEl.focus();
}

function closeImageViewer() {
  imageViewerEl.hidden = true;
  imageViewerImageEl.removeAttribute("src");
  imageViewerItems = [];
  document.body.classList.remove("modal-open");
}

function moveImageViewer(step) {
  if (imageViewerItems.length < 2) return;
  imageViewerIndex = (imageViewerIndex + step + imageViewerItems.length) % imageViewerItems.length;
  renderImageViewer();
}

function setImageViewerScale(nextScale) {
  imageViewerScale = Math.max(0.25, Math.min(5, nextScale));
  if (imageViewerScale <= 1) {
    imageViewerOffsetX = 0;
    imageViewerOffsetY = 0;
  }
  applyImageViewerTransform();
}

function destinationForConversation(conversationId) {
  if (isGroupConversation(conversationId)) {
    return {
      conversationId, member: null, scope: "group", to: groupIdForConversation(conversationId)
    };
  }
  const name = friendNameForConversation(conversationId);
  return { conversationId, member: memberByName(name), scope: "private", to: name };
}

function closeForwardDialog() {
  forwardDialogEl.hidden = true;
  forwardAttachment = null;
  forwardTargetsEl.replaceChildren();
  setFormStatus(forwardStatusEl, "");
  document.body.classList.remove("modal-open");
}

function forwardTargets() {
  const targets = [];
  [...state.groups.values()]
    .sort((left, right) => left.name.localeCompare(right.name, "zh-CN"))
    .forEach((group) => targets.push({
      conversationId: groupConversationIdFor(group.id), label: `# ${group.name}`,
      detail: `${group.members.length} 位成员`
    }));
  sortedMembers(members).forEach((member) => {
    targets.push({
      conversationId: conversationIdFor(member.name), label: `@${member.name}`,
      detail: member.online ? "在线" : "离线"
    });
  });
  return targets;
}

function openForwardDialog(attachment) {
  forwardAttachment = attachment;
  forwardTargetsEl.replaceChildren();
  forwardTargets().forEach((target) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "forward-target";
    const label = document.createElement("strong");
    label.textContent = target.label;
    const detail = document.createElement("span");
    detail.textContent = target.detail;
    button.append(label, detail);
    button.addEventListener("click", async () => {
      if (!forwardAttachment) return;
      forwardTargetsEl.querySelectorAll("button").forEach((item) => { item.disabled = true; });
      setFormStatus(forwardStatusEl, "正在转发...");
      forwardStatusEl.hidden = false;
      try {
        await sendPreparedMessage(
          { text: "", forwardAttachmentId: forwardAttachment.id },
          destinationForConversation(target.conversationId)
        );
        closeForwardDialog();
        setComposerStatus(`已转发到${target.label}`);
      } catch (error) {
        setFormStatus(forwardStatusEl, error.message || "转发失败", true);
        forwardTargetsEl.querySelectorAll("button").forEach((item) => { item.disabled = false; });
      }
    });
    forwardTargetsEl.append(button);
  });
  forwardDialogEl.hidden = false;
  document.body.classList.add("modal-open");
  setFormStatus(forwardStatusEl, "");
  forwardStatusEl.hidden = true;
  forwardCloseEl.focus();
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
  rejectPendingMessages("连接已断开，请稍后重试");
  if (socket) {
    socket.onclose = null;
    socket.close();
    socket = null;
  }
}

function showAuthUI(message = "") {
  backendAuthenticated = false;
  closeSocket();
  clearLocalAttachmentDrafts();
  setContentPicker(false);
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
  state.uploadsEnabled = Boolean(payload.uploadsEnabled);
  state.stickers = payload.stickers || [];
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
  reconcileAttachmentDrafts();

  showMessageTimeEl.checked = payload.self.settings.showMessageTime;
  fullMessageTimeEl.checked = payload.self.settings.fullMessageTime;
  parseLatexEl.checked = payload.self.settings.parseLatex;
  schemeSelectorEl.value = payload.self.settings.theme || "dune";
  document.body.dataset.theme = schemeSelectorEl.value;
  profileNameEl.value = state.currentUser;
  profileSignatureInputEl.value = state.profile.signature;
  updateUploadControls();
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
    settlePendingMessage(event.requestId);
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
      && shouldMarkConversationUnread(event.conversation)) {
      markConversationUnread(event.conversation);
    }
    if (isNewMessage && isIncomingGroupMessage
      && shouldMarkConversationUnread(event.conversation)) {
      markConversationUnread(event.conversation);
    }
    if (state.currentConversation === event.conversation) renderMessages();
    renderConversationNavigation();
    return;
  }

  if (event.type === "message_recalled") {
    settlePendingMessage(event.requestId);
    const messages = state.conversations[event.conversation] || [];
    const message = messages.find((candidate) => candidate.id === event.messageId);
    if (message) {
      message.recalled = true;
      message.canRecall = false;
      message.text = "";
      message.sticker = "";
      message.stickerAttachment = null;
      message.attachments = [];
    }
    if (state.currentConversation === event.conversation) renderMessages();
    return;
  }

  if (event.type === "error") {
    const error = new Error(event.message || "发送失败");
    if (event.requestId) settlePendingMessage(event.requestId, error);
    else setComposerStatus(error.message);
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
    rejectPendingMessages("连接已断开，请稍后重试");
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
  void sendCurrentMessage();
});

contentPickerToggleEl.addEventListener("click", (event) => {
  event.stopPropagation();
  setContentPicker(contentPickerEl.hidden);
});

emojiPanelEl.addEventListener("scroll", () => {
  const nearBottom = emojiPanelEl.scrollTop + emojiPanelEl.clientHeight
    >= emojiPanelEl.scrollHeight - 96;
  if (nearBottom && renderedEmojiCount < EMOJI_OPTIONS.length) appendEmojiOptions();
});

pickerTabEls.forEach((tab) => {
  tab.addEventListener("click", () => {
    setPickerTab(tab.dataset.pickerTab, tab.dataset.pickerTab === "file");
  });
});

chooseFilesEl.addEventListener("click", () => {
  if (state.uploadsEnabled) attachmentFileInputEl.click();
});

attachmentFileInputEl.addEventListener("change", () => {
  if (attachmentFileInputEl.files?.length) addAttachmentFiles(attachmentFileInputEl.files);
  attachmentFileInputEl.value = "";
  setContentPicker(false);
  inputEl.focus();
});

stickerFileInputEl.addEventListener("change", () => {
  const files = stickerFileInputEl.files ? Array.from(stickerFileInputEl.files) : [];
  stickerFileInputEl.value = "";
  if (files.length) void addStickerFiles(files);
});

mediaMenuEl.addEventListener("click", (event) => {
  const button = event.target.closest("[data-media-action]");
  if (!button || !mediaMenuContext) return;
  const context = mediaMenuContext;
  const action = button.dataset.mediaAction;
  hideMediaMenu();
  if (action === "preview") previewAttachment(context.attachment);
  else if (action === "collect") void favoriteSticker(context.attachment);
  else if (action === "download") openAttachmentDownload(context.attachment);
  else if (action === "forward") openForwardDialog(context.attachment);
  else if (action === "recall") void recallMessage(context.message);
  else if (action === "remove-sticker") void removeSticker(context.attachment);
});

forwardCloseEl.addEventListener("click", closeForwardDialog);
forwardDialogEl.addEventListener("click", (event) => {
  if (event.target === forwardDialogEl) closeForwardDialog();
});

imageViewerCloseEl.addEventListener("click", closeImageViewer);
imageViewerPreviousEl.addEventListener("click", () => moveImageViewer(-1));
imageViewerNextEl.addEventListener("click", () => moveImageViewer(1));
imageZoomOutEl.addEventListener("click", () => setImageViewerScale(imageViewerScale / 1.25));
imageZoomInEl.addEventListener("click", () => setImageViewerScale(imageViewerScale * 1.25));
imageZoomResetEl.addEventListener("click", resetImageViewerTransform);
imageViewerDownloadEl.addEventListener("click", () => {
  const attachment = imageViewerItems[imageViewerIndex];
  if (attachment) openAttachmentDownload(attachment);
});
imageViewerStageEl.addEventListener("wheel", (event) => {
  event.preventDefault();
  setImageViewerScale(imageViewerScale * (event.deltaY < 0 ? 1.15 : 1 / 1.15));
}, { passive: false });
imageViewerImageEl.addEventListener("dblclick", resetImageViewerTransform);

let imageDragPointer = null;
let imageDragStartX = 0;
let imageDragStartY = 0;
let imageDragOriginX = 0;
let imageDragOriginY = 0;
imageViewerImageEl.addEventListener("pointerdown", (event) => {
  if (imageViewerScale <= 1) return;
  imageDragPointer = event.pointerId;
  imageDragStartX = event.clientX;
  imageDragStartY = event.clientY;
  imageDragOriginX = imageViewerOffsetX;
  imageDragOriginY = imageViewerOffsetY;
  imageViewerImageEl.setPointerCapture(event.pointerId);
  imageViewerImageEl.classList.add("dragging");
});
imageViewerImageEl.addEventListener("pointermove", (event) => {
  if (imageDragPointer !== event.pointerId) return;
  imageViewerOffsetX = imageDragOriginX + event.clientX - imageDragStartX;
  imageViewerOffsetY = imageDragOriginY + event.clientY - imageDragStartY;
  applyImageViewerTransform();
});
const stopImageDrag = (event) => {
  if (imageDragPointer !== event.pointerId) return;
  imageDragPointer = null;
  imageViewerImageEl.classList.remove("dragging");
};
imageViewerImageEl.addEventListener("pointerup", stopImageDrag);
imageViewerImageEl.addEventListener("pointercancel", stopImageDrag);

let messageInputComposing = false;

inputEl.addEventListener("compositionstart", () => {
  messageInputComposing = true;
});

inputEl.addEventListener("compositionend", () => {
  messageInputComposing = false;
});

inputEl.addEventListener("compositioncancel", () => {
  messageInputComposing = false;
});

inputEl.addEventListener("keydown", (event) => {
  if (messageInputComposing || event.isComposing || event.keyCode === 229) return;
  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault();
    void sendCurrentMessage();
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
          fullMessageTime: fullMessageTimeEl.checked,
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
  localStorage.setItem("whisper-full-message-time", String(fullMessageTimeEl.checked));
  localStorage.setItem("whisper-parse-latex", String(parseLatexEl.checked));
  localStorage.setItem("whisper-theme", schemeSelectorEl.value);
}

showMessageTimeEl.addEventListener("change", () => {
  persistSettings();
  renderMessages();
});

fullMessageTimeEl.addEventListener("change", () => {
  persistSettings();
  renderMessages();
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
  if (!mediaMenuEl.hidden && !mediaMenuEl.contains(event.target)) hideMediaMenu();
  if (!contentPickerEl.hidden && !contentPickerEl.contains(event.target)
    && event.target !== contentPickerToggleEl) setContentPicker(false);
});

document.addEventListener("keydown", (event) => {
  if (!imageViewerEl.hidden) {
    if (event.key === "ArrowLeft") moveImageViewer(-1);
    else if (event.key === "ArrowRight") moveImageViewer(1);
    else if (event.key === "+" || event.key === "=") setImageViewerScale(imageViewerScale * 1.25);
    else if (event.key === "-") setImageViewerScale(imageViewerScale / 1.25);
    else if (event.key === "0") resetImageViewerTransform();
    else if (event.key === "Escape") closeImageViewer();
    if (["ArrowLeft", "ArrowRight", "+", "=", "-", "0", "Escape"].includes(event.key)) {
      event.preventDefault();
    }
    return;
  }
  if (event.key === "Escape") {
    if (!forwardDialogEl.hidden) closeForwardDialog();
    hideFriendMenu();
    hideGroupMenu();
    hideMediaMenu();
    setContentPicker(false);
    setSidebar(false);
  }
});

document.addEventListener("visibilitychange", () => {
  setPageAttentionActive(!document.hidden);
});

window.addEventListener("focus", () => setPageAttentionActive(true));
window.addEventListener("blur", () => setPageAttentionActive(false));
window.addEventListener("pageshow", () => setPageAttentionActive(true));
window.addEventListener("pagehide", () => setPageAttentionActive(false));
document.documentElement.addEventListener("pointerenter", () => setPageAttentionActive(true));
document.documentElement.addEventListener("pointerleave", () => setPageAttentionActive(false));
document.addEventListener("pointerdown", () => setPageAttentionActive(true));
document.addEventListener("touchstart", () => setPageAttentionActive(true), { passive: true });
document.addEventListener("wheel", () => setPageAttentionActive(true), { passive: true });
document.addEventListener("keydown", (event) => {
  const switchingTabs = event.key === "Tab" && (event.ctrlKey || event.metaKey);
  setPageAttentionActive(!switchingTabs);
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
fullMessageTimeEl.checked = localStorage.getItem("whisper-full-message-time") === "true";
parseLatexEl.checked = localStorage.getItem("whisper-parse-latex") !== "false";
renderContentPickerOptions();
updateUploadControls();
setSidebar(pinSidebarEl.checked);
setAuthMode("login");
if (typeof ResizeObserver !== "undefined") {
  const composerObserver = new ResizeObserver(() => {
    document.documentElement.style.setProperty("--composer-height", `${formEl.offsetHeight}px`);
  });
  composerObserver.observe(formEl);
}
startApplication();
