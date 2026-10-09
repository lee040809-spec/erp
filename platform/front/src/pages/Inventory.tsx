import React from 'react';
import { Plus, Image as ImageIcon } from 'lucide-react';
import { Material, Remnant } from '../types';

interface InventoryProps {
  materials: Material[];
  remnants: Remnant[];
}

const Inventory: React.FC<InventoryProps> = ({ materials, remnants }) => {
  return (
    <div className="erp-legacy-page flex-1 overflow-y-auto space-y-8 py-4 md:py-8 !w-full !max-w-none m-0 !p-0">
      <div className="flex items-center justify-between px-4 md:px-8">
        <h2 className="text-2xl font-bold">仓库与余料</h2>
        <button type="button" disabled title="采购入库与领退料流程尚未接入，当前为库存资料展示" className="opacity-50 cursor-not-allowed bg-zinc-900 text-white px-4 py-2 rounded-xl text-sm font-medium flex items-center gap-2">
          <Plus className="w-4 h-4" />
          入库材料（待接入）
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* Materials List */}
        <div className="space-y-4">
          <h3 className="font-bold text-zinc-500 uppercase tracking-wider text-xs">常规材料库</h3>
          <div className="bg-white rounded-2xl border border-zinc-200 divide-y divide-zinc-100">
            {materials.map((m) => (
              <div key={m.id} className="p-4 flex items-center justify-between">
                <div>
                  <p className="font-bold">{m.name}</p>
                  <p className="text-xs text-zinc-500">规格: {m.spec}</p>
                </div>
                <div className="text-right">
                  <p className="font-mono font-bold">{m.quantity} {m.unit}</p>
                  <p className="text-[10px] text-zinc-400 uppercase">{m.quantity > 0 ? "现有库存" : "暂无库存"}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Remnants Grid */}
        <div className="space-y-4">
          <h3 className="font-bold text-zinc-500 uppercase tracking-wider text-xs">余料回收 (可复用)</h3>
          <div className="grid grid-cols-2 gap-4">
            {remnants.map((r) => (
              <div key={r.id} className="bg-white rounded-2xl border border-zinc-200 overflow-hidden group cursor-pointer">
                <div className="aspect-square bg-zinc-100 flex items-center justify-center relative">
                  {r.photo_data ? (
                    <img src={r.photo_data} alt="余料" className="w-full h-full object-cover" />
                  ) : (
                    <ImageIcon className="w-8 h-8 text-zinc-300" />
                  )}
                  <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
                    <Plus className="w-6 h-6" />
                  </div>
                </div>
                <div className="p-3">
                  <p className="text-sm font-bold truncate">{r.material_name}</p>
                  <p className="text-xs text-zinc-500">{r.dimensions}</p>
                </div>
              </div>
            ))}
            <button type="button" disabled title="余料登记流程尚未接入" className="opacity-50 cursor-not-allowed aspect-square border-2 border-dashed border-zinc-200 rounded-2xl flex flex-col items-center justify-start gap-2 text-zinc-400 hover:border-zinc-400 hover:text-zinc-600 transition-all">
              <Plus className="w-6 h-6" />
              <span className="text-xs font-medium">新增余料（待接入）</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Inventory;
