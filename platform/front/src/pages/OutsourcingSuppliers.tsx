import { useCallback, useState } from "react";
import type {
  OutsourcingSupplier,
  SupplierPage,
} from "../types/outsourcingOrder";
import { flowRequest } from "../services/productionFlow";
import { useAsyncResource } from "../hooks/useAsyncResource";
import { useDraftGuard } from "../hooks/useDraftGuard";
import SupplierEditor from "../components/SupplierEditor";

const initial: SupplierPage = {
  items: [],
  page: 1,
  has_more: false,
  can_manage: false,
};
export default function OutsourcingSuppliers({
  embedded = false,
}: {
  embedded?: boolean;
}) {
  const [page, setPage] = useState(1),
    [search, setSearch] = useState(""),
    [query, setQuery] = useState("");
  const [editing, setEditing] = useState<
    OutsourcingSupplier | null | undefined
  >(null);
  const [saving, setSaving] = useState(false),
    [message, setMessage] = useState("");
  const [editorRevision, setEditorRevision] = useState(0);
  const request = useCallback(
    () =>
      flowRequest<SupplierPage>(
        `/outsourcing-suppliers?page=${page}&pageSize=30&search=${encodeURIComponent(query)}`,
      ),
    [page, query],
  );
  const { data, loading, error, reload } = useAsyncResource(request, initial);
  const editorSession = `${editing?.id || "new"}:${editorRevision}`;
  const guard = useDraftGuard(editing !== null, editorSession, saving);
  const edit = (supplier?: OutsourcingSupplier) => {
    if (!guard.confirmDiscard()) return;
    guard.reset();
    setEditing(supplier);
    setEditorRevision(value => value + 1);
    setMessage("");
  };
  const clear = () => {
    setSearch("");
    if (!query && page === 1) void reload();
    else {
      setQuery("");
      setPage(1);
    }
  };
  return (
    <section
      className={embedded ? "space-y-4" : "erp-page space-y-4 p-4 md:p-6"}
    >
      <header className="flex flex-wrap justify-between items-center gap-3">
        <div>
          <h2 className="text-xl font-bold text-slate-900">外协供应商管理</h2>
          <p className="text-xs text-slate-500 mt-1">
            维护联系人和加工能力；停用后保留历史订单、回厂与对账记录。
          </p>
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            className="erp-btn"
            disabled={loading || saving}
            onClick={() => void reload()}
          >
            刷新供应商
          </button>
          {data.can_manage && (
            <button
              type="button"
              className="erp-btn erp-primary"
              disabled={saving}
              onClick={() => edit()}
            >
              新增供应商
            </button>
          )}
        </div>
      </header>
      <form
        className="flex flex-wrap gap-2"
        role="search"
        aria-label="供应商查询"
        onSubmit={(e) => {
          e.preventDefault();
          const next = search.trim();
          setSearch(next);
          if (page === 1 && next === query) void reload();
          else {
            setPage(1);
            setQuery(next);
          }
        }}
      >
        <input
          className="erp-input flex-1 min-w-0"
          aria-label="搜索供应商"
          placeholder="供应商 / 联系人 / 电话 / 加工能力"
          value={search}
          disabled={saving}
          onChange={(e) => setSearch(e.target.value)}
        />
        <button className="erp-btn" disabled={loading || saving}>
          查询
        </button>
        {(query || search) && (
          <button
            type="button"
            className="erp-btn"
            disabled={saving}
            onClick={clear}
          >
            清除
          </button>
        )}
      </form>
      {message && (
        <p role="status" className="text-emerald-700 text-sm">
          {message}
        </p>
      )}
      {error && (
        <div
          role="alert"
          className="erp-card p-3 bg-amber-50 text-amber-800 text-sm"
        >
          <p>供应商读取失败：{error}</p>
          {message && (
            <p className="mt-1">
              资料已经保存，请勿重复提交；重新读取清单即可。
            </p>
          )}
          <button
            type="button"
            className="erp-btn mt-2"
            disabled={saving || loading}
            onClick={() => void reload()}
          >
            重新读取供应商
          </button>
        </div>
      )}
      {editing !== null && (
        <SupplierEditor
          key={editorSession}
          supplier={editing || undefined}
          onDirty={guard.markChanged}
          onBusy={setSaving}
          onCancel={() => {
            if (guard.confirmDiscard()) setEditing(null);
          }}
          onSaved={async () => {
            guard.reset();
            setEditing(null);
            setMessage("供应商资料已保存");
            await reload();
          }}
        />
      )}
      {loading ? (
        <p role="status" className="py-6 text-sm text-slate-500">
          正在读取供应商…
        </p>
      ) : (
        !error && (
          <>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
              {data.items.map((supplier) => (
                <article key={supplier.id} className="erp-card p-3 space-y-2">
                  <div className="flex justify-between gap-2">
                    <strong className="text-sm break-words">
                      {supplier.name}
                    </strong>
                    <span
                      className={`text-xs shrink-0 ${supplier.active ? "text-emerald-600" : "text-slate-400"}`}
                    >
                      {supplier.active ? "启用" : "停用"}
                    </span>
                  </div>
                  <p className="text-xs text-slate-600 break-words">
                    {supplier.contact_name || "未填联系人"} ·{" "}
                    {supplier.phone || "未填电话"}
                  </p>
                  <p className="text-xs text-slate-500 break-words">
                    加工能力：{supplier.capabilities || "未填写"}
                  </p>
                  {supplier.address && (
                    <p className="text-xs text-slate-500 break-words">
                      {supplier.address}
                    </p>
                  )}
                  {supplier.notes && (
                    <p className="text-xs text-slate-500 break-words">
                      {supplier.notes}
                    </p>
                  )}
                  {data.can_manage && (
                    <button
                      type="button"
                      className="erp-btn"
                      disabled={saving}
                      onClick={() => edit(supplier)}
                    >
                      编辑 / 启停
                    </button>
                  )}
                </article>
              ))}
            </div>
            {!data.items.length && (
              <p role="status" className="text-sm text-slate-500">
                {query
                  ? "没有找到符合条件的供应商，可清除关键词重新查询。"
                  : "暂无供应商资料。"}
              </p>
            )}
          </>
        )
      )}
      <div className="flex items-center gap-3 text-xs">
        <button
          type="button"
          className="erp-btn"
          disabled={page <= 1 || loading || saving || !!error}
          onClick={() => setPage((p) => p - 1)}
        >
          上一页
        </button>
        <span>第 {page} 页</span>
        <button
          type="button"
          className="erp-btn"
          disabled={!data.has_more || loading || saving || !!error}
          onClick={() => setPage((p) => p + 1)}
        >
          下一页
        </button>
      </div>
    </section>
  );
}
