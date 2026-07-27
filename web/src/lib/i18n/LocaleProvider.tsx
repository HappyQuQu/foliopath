import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import {
  readLocalePreference,
  writeLocalePreference,
  type LocalePreference,
} from "../storage/preferences";

const zhCN = {
  "account.account": "账户",
  "account.appearance": "外观",
  "account.description": "管理当前管理员会话与界面外观。",
  "account.eyebrow": "偏好",
  "account.language": "语言",
  "account.languageDescription": "默认跟随浏览器语言，也可以在此切换。",
  "account.logout": "退出登录",
  "account.logoutFailed": "暂时无法退出，请稍后重试。",
  "account.theme": "主题",
  "account.themeDescription": "默认跟随系统，也可以从页面右上角切换。",
  "account.title": "通用设置",
  "account.username": "用户名：",
  "auth.confirmPassword": "确认密码",
  "auth.create": "创建账户",
  "auth.displayName": "显示名称",
  "auth.firstUse": "首次使用",
  "auth.invalidCredentials": "用户名或密码不正确。",
  "auth.invalidInput": "请检查输入内容后重试。",
  "auth.login": "登录",
  "auth.loginIntro": "使用管理员账户继续访问您的媒体库。",
  "auth.loginTitle": "登录 FolioPath",
  "auth.originInvalid": "当前页面来源未通过安全检查，请从 FolioPath 正式地址访问。",
  "auth.password": "密码",
  "auth.rateLimited": "尝试次数过多，请稍后再试。",
  "auth.sessionExpired": "为了保护您的媒体库，会话已过期。请重新登录。",
  "auth.setupInProgress": "另一项初始化正在进行，请稍后重试。",
  "auth.setupIntro": "此账户用于管理媒体库、扫描任务与系统设置。",
  "auth.setupTitle": "创建管理员账户",
  "auth.unknownFailure": "操作没有完成，请稍后重试。",
  "auth.unavailable": "暂时无法完成操作，请稍后重试。",
  "auth.username": "用户名",
  "common.closeNotice": "关闭提示",
  "common.closeToast": "关闭通知",
  "common.retry": "重新检查",
  "common.skipToMain": "跳到主要内容",
  "common.toastRegion": "通知",
  "common.readOnlyFooter": "您的原始媒体始终保持只读",
  "error.confirmingSecurity": "正在确认安全状态…",
  "error.reload": "重新载入",
  "error.renderBody": "您的媒体没有被修改。请重新载入界面；如果问题持续，请检查服务状态。",
  "error.renderTitle": "界面暂时无法显示",
  "error.pageFailed": "页面暂时无法载入。",
  "error.serviceFailed": "FolioPath 暂时无法响应。原始媒体没有被修改。",
  "error.serviceOffline":
    "无法连接 FolioPath 服务。原始媒体未被修改，请确认服务正在运行后重新检查。",
  "readiness.applicationData":
    "应用数据目录不可用。请检查数据卷挂载与写入权限后重新检查。",
  "readiness.database": "应用数据库暂时不可用。原始媒体未被修改，请稍后重新检查。",
  "readiness.migration":
    "数据库升级没有完成。原始媒体未被修改，请检查服务日志后重新检查。",
  "readiness.shuttingDown": "FolioPath 正在安全停止。请等待服务重新启动后再检查。",
  "theme.toDark": "切换到深色主题",
  "theme.toLight": "切换到浅色主题",
  "unavailable.default":
    "FolioPath 暂时无法完成启动，系统已安全停止。媒体目录没有被修改。",
  "unavailable.eyebrow": "服务暂不可用",
  "unavailable.noDiagnostics": "没有显示内部路径或诊断信息",
  "unavailable.readOnly": "原始媒体保持只读",
  "unavailable.safety": "安全状态",
  "unavailable.title": "FolioPath 无法完成启动",
  "validation.confirmPassword": "两次输入的密码不一致。",
  "validation.displayName": "请输入显示名称。",
  "validation.displayNameLength": "显示名称不能超过 128 个字符。",
  "validation.password": "请输入密码。",
  "validation.passwordLength": "密码至少需要 12 个字符。",
  "validation.username": "请输入用户名。",
  "validation.usernameLength": "用户名不能超过 64 个字符。",
} as const;

export type MessageKey = keyof typeof zhCN;
export type Locale = LocalePreference;

const en: Record<MessageKey, string> = {
  "account.account": "Account",
  "account.appearance": "Appearance",
  "account.description": "Manage the current administrator session and interface appearance.",
  "account.eyebrow": "Preferences",
  "account.language": "Language",
  "account.languageDescription": "Uses your browser language by default, or choose one here.",
  "account.logout": "Log out",
  "account.logoutFailed": "Unable to log out right now. Please try again.",
  "account.theme": "Theme",
  "account.themeDescription": "Follows your system by default, with a page-level override.",
  "account.title": "General settings",
  "account.username": "Username: ",
  "auth.confirmPassword": "Confirm password",
  "auth.create": "Create account",
  "auth.displayName": "Display name",
  "auth.firstUse": "First use",
  "auth.invalidCredentials": "The username or password is incorrect.",
  "auth.invalidInput": "Check the entered information and try again.",
  "auth.login": "Log in",
  "auth.loginIntro": "Use the administrator account to continue to your media libraries.",
  "auth.loginTitle": "Log in to FolioPath",
  "auth.originInvalid": "This page did not pass the security check. Use the official FolioPath address.",
  "auth.password": "Password",
  "auth.rateLimited": "Too many attempts. Please try again later.",
  "auth.sessionExpired": "Your session expired to protect your media libraries. Please log in again.",
  "auth.setupInProgress": "Another setup is in progress. Please try again shortly.",
  "auth.setupIntro": "This account manages media libraries, scan jobs, and system settings.",
  "auth.setupTitle": "Create administrator account",
  "auth.unknownFailure": "The operation did not complete. Please try again.",
  "auth.unavailable": "Unable to complete the operation right now. Please try again.",
  "auth.username": "Username",
  "common.closeNotice": "Dismiss notice",
  "common.closeToast": "Dismiss notification",
  "common.retry": "Check again",
  "common.skipToMain": "Skip to main content",
  "common.toastRegion": "Notifications",
  "common.readOnlyFooter": "Your original media always remains read-only",
  "error.confirmingSecurity": "Confirming security status…",
  "error.reload": "Reload",
  "error.renderBody":
    "Your media was not modified. Reload the interface and check the service status if the problem continues.",
  "error.renderTitle": "The interface could not be displayed",
  "error.pageFailed": "The page could not be loaded.",
  "error.serviceFailed": "FolioPath is temporarily unavailable. Original media was not modified.",
  "error.serviceOffline":
    "Unable to reach the FolioPath service. Original media was not modified. Confirm the service is running and check again.",
  "readiness.applicationData":
    "Application data is unavailable. Check the data volume mount and write permissions, then check again.",
  "readiness.database":
    "The application database is temporarily unavailable. Original media was not modified. Check again shortly.",
  "readiness.migration":
    "The database upgrade did not complete. Original media was not modified. Check the service logs, then check again.",
  "readiness.shuttingDown":
    "FolioPath is shutting down safely. Wait for the service to restart, then check again.",
  "theme.toDark": "Switch to dark theme",
  "theme.toLight": "Switch to light theme",
  "unavailable.default":
    "FolioPath could not finish starting and stopped safely. The media directory was not modified.",
  "unavailable.eyebrow": "Service unavailable",
  "unavailable.noDiagnostics": "Internal paths and diagnostics are not displayed",
  "unavailable.readOnly": "Original media remains read-only",
  "unavailable.safety": "Safe state",
  "unavailable.title": "FolioPath could not finish starting",
  "validation.confirmPassword": "The passwords do not match.",
  "validation.displayName": "Enter a display name.",
  "validation.displayNameLength": "The display name must be 128 characters or fewer.",
  "validation.password": "Enter a password.",
  "validation.passwordLength": "The password must contain at least 12 characters.",
  "validation.username": "Enter a username.",
  "validation.usernameLength": "The username must be 64 characters or fewer.",
};

interface LocaleContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: MessageKey) => string;
}

const LocaleContext = createContext<LocaleContextValue>({
  locale: "zh-CN",
  setLocale: () => undefined,
  t: (key) => zhCN[key],
});

export function detectBrowserLocale(): Locale {
  const candidates = navigator.languages.length > 0 ? navigator.languages : [navigator.language];
  return candidates.some((locale) => locale.toLowerCase().startsWith("zh")) ? "zh-CN" : "en";
}

export function applyInitialLocale(): Locale {
  const locale = readLocalePreference() ?? detectBrowserLocale();
  document.documentElement.lang = locale;
  return locale;
}

export function translate(locale: Locale, key: MessageKey): string {
  return locale === "zh-CN" ? zhCN[key] : en[key];
}

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(applyInitialLocale);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const value = useMemo<LocaleContextValue>(
    () => ({
      locale,
      setLocale: (nextLocale) => {
        writeLocalePreference(nextLocale);
        setLocaleState(nextLocale);
      },
      t: (key) => translate(locale, key),
    }),
    [locale],
  );

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale(): LocaleContextValue {
  return useContext(LocaleContext);
}
