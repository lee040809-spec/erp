// Browser preferences and drafts never contain tokens or share data across users/companies.
export function browserScope(): string | null {
  try {
    if (typeof localStorage === "undefined") return null;
    const user = JSON.parse(localStorage.getItem("auth_user") || "null");
    return user?.id != null && user?.corp_id != null
      ? `${user.corp_id}:${user.id}`
      : null;
  } catch {
    return null;
  }
}
export function scopedKey(name: string) {
  const scope = browserScope();
  return scope ? `erp:v1:${scope}:${name}` : null;
}
export function readScoped<T>(name: string, fallback: T): T {
  try {
    const key = scopedKey(name);
    return key
      ? (JSON.parse(localStorage.getItem(key) || "null") ?? fallback)
      : fallback;
  } catch {
    return fallback;
  }
}
export function writeScoped(name: string, value: unknown): boolean {
  try {
    const key = scopedKey(name);
    if (!key) return false;
    localStorage.setItem(key, JSON.stringify(value));
    return true;
  } catch {
    return false;
  }
}
