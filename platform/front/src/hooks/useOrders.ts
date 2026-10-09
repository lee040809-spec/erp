import React, { useCallback, useRef } from 'react';
import { Order, OrderProcess } from '../types';
import { saveProcessStatus, ProcessSaveNotice } from '../services/processStatus';
import { formatDateForInput } from '../components/shared';

export function useOrders(
  orders: Order[],
  setOrders: React.Dispatch<React.SetStateAction<Order[]>>,
  fetchData: () => void | Promise<void>,
  notify: (notice: ProcessSaveNotice) => void
) {
  // 获取订单最大交期
  const getOrderMaxDueDate = useCallback((order: Order): string => {
    let max = order.due_date || '';
    if (order.items && order.items.length > 0) {
      const dates = order.items.map(i => i.due_date).filter(Boolean);
      if (dates.length > 0) {
        max = dates.sort().reverse()[0];
      }
    }
    return max;
  }, []);

  const pendingProcesses = useRef(new Set<number>());
  const handleProcessClick = useCallback(async (orderId: number, itemId: number, processId: number, currentStatus: string) => {
    if (pendingProcesses.current.has(processId)) return;
    const nextStatus: OrderProcess['status'] = currentStatus === 'pending' ? 'processing' : currentStatus === 'processing' ? 'completed' : 'pending';
    pendingProcesses.current.add(processId);
    notify({kind: 'saving', message: '正在保存工序状态…'});
    try {
      const result = await saveProcessStatus(itemId, processId, nextStatus);
      setOrders(previous => previous.map(order => Number(order.id) !== Number(orderId) ? order : {
        ...order, status: result.order_status,
        items: (order.items || []).map(item => Number(item.id) !== Number(itemId) ? item : {
          ...item, status: result.item_status,
          processes: (item.processes || []).map(process => Number(process.id) !== Number(processId) ? process : {...process, status: nextStatus})
        })
      }));
      await fetchData();
      notify({kind: 'success', message: '工序状态已保存。合格完工数量请通过完工确认入账。'});
    } catch (error) {
      notify({kind: 'error', message: (error instanceof Error ? error.message : '保存失败') + '；本次状态未更新，请重试。'});
    } finally { pendingProcesses.current.delete(processId); }
  }, [setOrders, fetchData, notify]);

  // 编辑订单
  const editOrder = useCallback((order: Order, setNewOrder: (order: Partial<Order>) => void, setShowOrderModal: (show: boolean) => void) => {
    const orderStartDate = order.start_date || (order.items && order.items.length > 0 ? order.items[0].start_date : null);
    const orderDueDate = order.due_date || (order.items && order.items.length > 0 ? order.items[0].due_date : null);

    setNewOrder({
      id: order.id,
      order_number: order.order_number,
      order_name: order.order_name,
      customer_id: order.customer_id,
      customer_name: order.customer_name,
      customer_short_name: order.customer_short_name,
      contact_name: order.contact_name,
      priority: order.priority,
      start_date: formatDateForInput(orderStartDate),
      due_date: formatDateForInput(orderDueDate),
      total_amount: order.total_amount,
      notes: order.notes,
      items: (order.items || []).map(item => ({
        ...item,
        id: item.id,
        order_id: item.order_id,
        part_number: item.part_number || '',
        scrap_quantity: item.scrap_quantity || 0,
        delivered_quantity: item.delivered_quantity || 0,
        tool_cost: item.tool_cost || 0,
        fixture_cost: item.fixture_cost || 0,
        material_cost: item.material_cost || 0,
        other_cost: item.other_cost || 0,
        item_notes: item.item_notes || '',
        start_date: formatDateForInput(item.start_date || orderStartDate),
        due_date: formatDateForInput(item.due_date || orderDueDate),
        completion_date: formatDateForInput(item.completion_date),
        processes: (item.processes || []).map(p => ({
          id: p.id,
          order_item_id: p.order_item_id,
          name: p.name,
          is_outsourced: p.is_outsourced,
          outsourcing_fee: p.outsourcing_fee,
          status: p.status,
          sort_order: p.sort_order,
          planned_start_at: p.planned_start_at,
          planned_end_at: p.planned_end_at,
          actual_start_at: p.actual_start_at,
          actual_end_at: p.actual_end_at
        }))
      }))
    });
    setShowOrderModal(true);
  }, []);

  // 重置并打开新建订单弹框
  const resetAndOpenModal = useCallback((
    setNewOrder: (order: Partial<Order>) => void,
    setShowOrderModal: (show: boolean) => void
  ) => {
    const today = new Date();
    const dateStr = today.getFullYear().toString() +
                    (today.getMonth() + 1).toString().padStart(2, '0') +
                    today.getDate().toString().padStart(2, '0');
    const prefix = `YHS-${dateStr}-`;

    const todayOrders = orders.filter(o => o.order_number?.startsWith(prefix));
    let nextSuffix = 1;
    if (todayOrders.length > 0) {
      const suffixes = todayOrders.map(o => {
        const parts = o.order_number!.split('-');
        const lastPart = parts[parts.length - 1];
        return parseInt(lastPart) || 0;
      });
      nextSuffix = Math.max(...suffixes) + 1;
    }
    const generatedOrderNumber = `${prefix}${nextSuffix.toString().padStart(3, '0')}`;

    setNewOrder({
      priority: 'medium',
      status: 'pending',
      start_date: '',
      due_date: '',
      order_number: generatedOrderNumber,
      items: [{ part_name: '', quantity: 1, unit_price: 0, processes: [] }]
    } as Partial<Order>);
    setShowOrderModal(true);
  }, [orders]);

  return {
    getOrderMaxDueDate,
    handleProcessClick,
    editOrder,
    resetAndOpenModal
  };
}