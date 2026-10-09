export type WorkbenchView = 'all' | 'unassigned' | 'today' | 'overdue' | 'blocked' | 'outsourced';

export interface WorkbenchState {
    view: WorkbenchView;
    search: string;
    query: string;
    page: number;
    size: number;
    density: 'comfortable' | 'compact';
    chartMode?: boolean;
}

export interface WorkbenchOrderTarget {
    orderId: number;
    itemId: number;
    orderNumber: string;
    partName: string;
    sourceTab?: 'dashboard' | 'production_dashboard';
    state: WorkbenchState;
}

export const workbenchViewDescriptions: Record<WorkbenchView, string> = {
    all: '按零件下单日期从新到旧排列；未填时沿用订单日期。点击“查看订单”定位对应订单与零件。',
    unassigned: '生产安排尚未填写负责人的零件。',
    today: '计划完工日期为今天的零件；未排计划时优先按未交完批次交期，其次按零件交期判断。',
    overdue: '尚未交完的最早批次交期，或未设批次时的零件交期，早于今天的任务。',
    blocked: '生产安排填写了受阻原因的零件。问题解决后，在任务上点击“解除”或清空受阻原因。',
    outsourced: '已启用数量跟踪时按未回厂、待返工或返工在外数量筛选；旧零件未启用时沿用外协生产工序未完成状态。'
};
