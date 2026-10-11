import type { FlowData } from '../types/productionFlow';
export type WorkflowStep = 'process' | 'completion' | 'delivery' | 'batches' | 'outsourcing' | 'records';
export function nextPartAction(data: FlowData, priceReady?: boolean): {
    step: WorkflowStep;
    title: string;
    detail: string;
} {
    const { flow } = data, item = flow.item;
    const ordered = Number(item.quantity) || 0, delivered = Number(item.delivered_quantity) || 0, qualified = Number(item.qualified_quantity) || 0;
    const remaining = Math.max(0, Number(item.production_quantity || ordered) - Number(item.scrap_quantity || 0) - qualified);
    if (ordered > 0 && delivered >= ordered)
        return { step: 'records', title: '订单数量已交齐', detail: remaining > 0 ? '仍有投产余量；如需完成备用件，可继续办理加工和合格确认。' : '可查看交货记录、送货单及异常处理结果。' };
    if (Math.min(qualified, ordered) > delivered)
        return { step: 'delivery', title: '已有合格件，可以登记交货', detail: '支持先交已合格的部分，剩余零件可继续加工；请填写本次实际交货数量和日期。' };
    const processes = flow.processes.filter(p => p.name !== '送货');
    const ready = flow.tracking ? Number(flow.available_completion) > 0 : processes.length > 0 && processes.every(p => p.status === 'completed') && remaining > 0;
    if (ready)
        return { step: 'completion', title: priceReady === false ? '加工已完成，请核对单价后确认合格' : '可确认本次合格数量', detail: priceReady === false ? '尚未录入有效单价，确认前需补录真实单价；不会按零产值入账。' : '工序完工和合格确认是两项记录。确认合格后自动计入对应月份产值，再登记交货。' };
    if (!processes.length)
        return { step: 'process', title: '先设置加工工序', detail: '在“修改 / 添加 / 删除工序”中设置工艺，再安排日期或登记开工。' };
    return { step: 'process', title: '继续安排或登记加工', detail: flow.tracking ? '逐道登记本次开工、完工或报废数量；前道部分完工后，后道即可安排开工。' : '选中工序，安排计划日期或点击“现在开工 / 现在完工”；只记录实际发生的加工。' };
}
// The flow endpoint contains the original item; confirmed totals come from the business ledger.
export function withBusinessQuantities(data: FlowData, totals?: Partial<Record<'quantity' | 'production_quantity' | 'scrap_quantity' | 'qualified_quantity' | 'delivered_quantity', number | string>>): FlowData {
    if (!totals)
        return data;
    const item = { ...data.flow.item };
    for (const key of ['quantity', 'production_quantity', 'scrap_quantity', 'qualified_quantity', 'delivered_quantity'] as const) {
        if (totals[key] !== undefined && totals[key] !== null) {
            const value = Number(totals[key]);
            if (Number.isFinite(value))
                item[key] = value;
        }
    }
    return { ...data, flow: { ...data.flow, item } };
}
