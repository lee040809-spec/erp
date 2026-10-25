import { useCallback, useState } from "react";
import { flowRequest } from "../../services/productionFlow";
import { useAsyncResource } from "../../hooks/useAsyncResource";
interface Runtime {
  source: string;
  company: string;
  corp_id: number;
  database_ready: boolean;
  build_time: string;
  version: string;
  backup_status: string;
}
export default function RuntimeInfo() {
  const [open, setOpen] = useState(false);
  const request = useCallback(
    () => flowRequest<Runtime>("/system/context"),
    [],
  );
  const { data, error, loading, reload } = useAsyncResource(
    request,
    null as Runtime | null,
    true,
  );
  return (
    <div className="mx-4 mb-2 text-xs text-slate-300">
      <button
        type="button"
        className="text-left w-full hover:text-white"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        {data
          ? `${data.source} · ${data.database_ready ? "连接正常" : "连接异常"}`
          : "运行环境与服务状态"}{" "}
        {open ? "收起" : "查看"}
      </button>
      {open && (
        <div className="mt-2 rounded-lg bg-white/5 p-2 leading-5 break-words">
          {loading ? (
            "正在检查服务…"
          ) : error ? (
            <p role="alert">{error}</p>
          ) : (
            data && (
              <>
                <p>{data.source}</p>
                <p>
                  {data.company} · 公司 {data.corp_id}
                </p>
                <p>
                  数据库：
                  {data.database_ready ? "连接正常" : "连接异常，请稍后重试"}
                </p>
                <p>
                  {data.version} · 构建 {data.build_time}
                </p>
                <p>{data.backup_status}</p>
              </>
            )
          )}
          <button
            type="button"
            className="mt-1 underline"
            disabled={loading}
            onClick={reload}
          >
            重新检查
          </button>
        </div>
      )}
    </div>
  );
}
