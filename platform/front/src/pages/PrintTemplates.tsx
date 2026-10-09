import React, { useState, useRef } from 'react';
import { Plus, FileText, Search, Settings, Trash2, RefreshCw, Eye, Upload, Link2, Unlink, Download, Printer } from 'lucide-react';
import { authFetch } from '../components/shared';
import UniverSheet from '../components/UniverSheet';
import pako from 'pako';
import UniverImportExport from '@mertdeveci55/univer-import-export';

interface PageResource {
  key: string;
  name: string;
  type: string;
}

interface ResourceWithButtons {
  id: number;
  resource_key: string;
  name: string;
  page_resources: PageResource[];
}

interface PrintTemplate {
  id: number;
  name: string;
  menu_route: string;
  button_key: string;
  button_name: string;
  template: string;
  excel_filename?: string;
  created_at: string;
}

interface PrintTemplateFilters {
  name: string;
}

interface PrintTemplatesProps {
  printTemplates: PrintTemplate[];
  setPrintTemplates: (templates: PrintTemplate[]) => void;
}

const PrintTemplates: React.FC<PrintTemplatesProps> = ({
  printTemplates,
  setPrintTemplates
}) => {
  const [filters, setFilters] = useState<PrintTemplateFilters>({ name: '' });
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [form, setForm] = useState({
    name: '',
    menu_route: '',
    button_key: '',
    button_name: '',
    template: ''
  });
  const [showPreviewModal, setShowPreviewModal] = useState(false);
  const [previewSheetData, setPreviewSheetData] = useState<any>(null);
  const [previewTemplateId, setPreviewTemplateId] = useState<number | null>(null);
  const [importingTemplateId, setImportingTemplateId] = useState<number | null>(null);
  const excelInputRef = useRef<HTMLInputElement>(null);
  const univerRef = useRef<any>(null);

  // Toast提示状态
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 1000);
  };

  // 绑定弹框状态
  const [showBindModal, setShowBindModal] = useState(false);
  const [bindingTemplateId, setBindingTemplateId] = useState<number | null>(null);
  const [bindingTemplateName, setBindingTemplateName] = useState('');
  const [resourcesWithButtons, setResourcesWithButtons] = useState<ResourceWithButtons[]>([]);
  const [bindLoading, setBindLoading] = useState(false);

  // 测试打印弹框状态
  const [showTestPrintModal, setShowTestPrintModal] = useState(false);
  const [testPrintTemplateId, setTestPrintTemplateId] = useState<number | null>(null);
  const [testPrintTemplateName, setTestPrintTemplateName] = useState('');
  const [testPrintData, setTestPrintData] = useState('');

  const fetchData = async () => {
    const res = await authFetch('/api/platform/print-templates');
    const result = await res.json();
    setPrintTemplates(result.data || []);
  };

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await fetchData();
    } finally {
      setIsRefreshing(false);
    }
  };

  const handleOpenModal = (template?: PrintTemplate) => {
    if (template) {
      setEditingId(template.id);
      setForm({
        name: template.name,
        menu_route: template.menu_route || '',
        button_key: template.button_key || '',
        button_name: template.button_name || '',
        template: template.template || ''
      });
    } else {
      setEditingId(null);
      setForm({ name: '', menu_route: '', button_key: '', button_name: '', template: '' });
    }
    setShowModal(true);
  };

  const handleSubmit = async () => {
    if (!form.name) {
      alert('请输入模板名称');
      return;
    }

    const existingTemplate = printTemplates.find(t => t.id !== editingId && t.name === form.name);
    if (existingTemplate) {
      alert(`模板名称 "${form.name}" 已存在，请使用其他名称`);
      return;
    }

    try {
      let res;
      if (editingId) {
        res = await authFetch(`/api/platform/print-templates/${editingId}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(form)
        });
      } else {
        res = await authFetch('/api/platform/print-templates', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(form)
        });
      }
      const result = await res.json();
      if (!res.ok || result.error) {
        alert(result.error || '操作失败');
        return;
      }
      setShowModal(false);
      await fetchData();
    } catch (err) {
      alert('操作失败');
    }
  };

  const handleDelete = async () => {
    if (!deletingId) return;
    try {
      await authFetch(`/api/platform/print-templates/${deletingId}`, {
        method: 'DELETE'
      });
      setDeletingId(null);
      await fetchData();
    } catch (err) {
      alert('删除失败');
    }
  };

  // 预览模板 - 解压数据并用Univer显示
  const handlePreview = (templateId: number, preview: string) => {
    setPreviewTemplateId(templateId);

    if (!preview) {
      alert('模板数据为空');
      return;
    }

    try {
      let workbookData: any;

      // 检查数据格式
      if (preview.startsWith('GZIP:')) {
        const base64 = preview.substring(5);
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        const decompressed = pako.inflate(bytes, { to: 'string' });
        workbookData = JSON.parse(decompressed);
      } else if (preview.startsWith('UNIVER:')) {
        // 新格式：Univer workbook数据
        const base64 = preview.substring(7);
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        const decompressed = pako.inflate(bytes, { to: 'string' });
        workbookData = JSON.parse(decompressed);
      } else {
        // 直接JSON
        workbookData = JSON.parse(preview);
      }

      setPreviewSheetData(workbookData);
      setShowPreviewModal(true);
    } catch (err) {
      alert('模板数据解析失败');
    }
  };

  const handleDownloadExcel = async (templateId: number, filename: string) => {
    try {
      const res = await authFetch(`/api/platform/print-templates/${templateId}/download-excel`);
      if (!res.ok) {
        const result = await res.json();
        alert(result.error || '下载失败');
        return;
      }
      const blob = await res.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      alert('下载失败');
    }
  };

  const handleImportExcel = (templateId: number) => {
    setImportingTemplateId(templateId);
    excelInputRef.current?.click();
  };

  const handleExcelFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !importingTemplateId) return;

    try {
      const arrayBuffer = await file.arrayBuffer();
      const header = new Uint8Array(arrayBuffer.slice(0, 8));
      const isOLE2 = header[0] === 0xD0 && header[1] === 0xCF && header[2] === 0x11 && header[3] === 0xE0;

      if (isOLE2) {
        alert('该Excel文件是加密或旧格式(.xls)文件。\n请：\n1. 用Excel打开文件\n2. 取消密码保护\n3. 另存为.xlsx格式\n4. 重新导入');
        setImportingTemplateId(null);
        if (excelInputRef.current) excelInputRef.current.value = '';
        return;
      }

      // 使用 UniverImportExport.transformExcelToUniver 直接转换为Univer格式
      UniverImportExport.transformExcelToUniver(file, (workbookData: any) => {
        if (!workbookData) {
          alert('Excel解析失败');
          setImportingTemplateId(null);
          if (excelInputRef.current) excelInputRef.current.value = '';
          return;
        }

        // gzip压缩Univer workbook数据
        const workbookJson = JSON.stringify(workbookData);
        const compressed = pako.gzip(workbookJson);
        const base64 = btoa(String.fromCharCode(...compressed));
        const compressedData = 'UNIVER:' + base64;

        // 上传到后端
        const formData = new FormData();
        formData.append('file', file);
        formData.append('html', compressedData);

        authFetch(`/api/platform/print-templates/${importingTemplateId}/import-excel`, {
          method: 'POST',
          body: formData
        }).then(res => res.json())
          .then(result => {
            if (result.error) {
              alert(result.error || '导入Excel失败');
              return;
            }
            showToast('导入成功', 'success');
            fetchData();
          })
          .catch(err => {
            alert('上传失败');
          })
          .finally(() => {
            setImportingTemplateId(null);
            if (excelInputRef.current) excelInputRef.current.value = '';
          });
      }, (err: Error) => {
        alert('Excel解析失败: ' + err.message);
        setImportingTemplateId(null);
        if (excelInputRef.current) excelInputRef.current.value = '';
      });
    } catch (err: any) {
      alert('导入Excel失败: ' + (err.message || '未知错误'));
      setImportingTemplateId(null);
      if (excelInputRef.current) excelInputRef.current.value = '';
    }
  };

  const handleTestPrint = (templateId: number) => {
    const template = printTemplates.find(t => t.id === templateId);
    if (!template) return;
    setTestPrintTemplateId(templateId);
    setTestPrintTemplateName(template.name);
    setTestPrintData('{\n  "订单编号": "2024001",\n  "客户名称": "测试客户",\n  "订单金额": "10000"\n}');
    setShowTestPrintModal(true);
  };

  const handleGenerateExcel = async () => {
    if (!testPrintTemplateId) return;
    try {
      const data = JSON.parse(testPrintData);
      const res = await authFetch(`/api/platform/print-templates/${testPrintTemplateId}/generate-excel?timestamp=${new Date().toISOString()}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      });
      if (!res.ok) {
        const result = await res.json();
        alert(result.error || '生成失败');
        return;
      }
      const blob = await res.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `print_${testPrintTemplateName}_${new Date().toISOString().slice(0,10)}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      setShowTestPrintModal(false);
    } catch (err) {
      alert('JSON格式错误或生成失败');
    }
  };

  const handleBindButton = async (templateId: number) => {
    const template = printTemplates.find(t => t.id === templateId);
    if (!template) return;

    setBindingTemplateId(templateId);
    setBindingTemplateName(template.name);
    setBindLoading(true);
    setShowBindModal(true);

    try {
      const res = await authFetch('/api/platform/resources/page-buttons');
      const result = await res.json();
      if (result.data) {
        const parsed = result.data.map((r: any) => ({
          ...r,
          page_resources: typeof r.page_resources === 'string' ? JSON.parse(r.page_resources) : r.page_resources
        }));
        setResourcesWithButtons(parsed);
      }
    } catch (err) {
      // 获取资源失败
    } finally {
      setBindLoading(false);
    }
  };

  const handleSelectButton = async (resourceId: number, button: PageResource) => {
    if (!bindingTemplateId) return;

    const template = printTemplates.find(t => t.id === bindingTemplateId);
    if (!template) return;

    const existingTemplate = printTemplates.find(t => t.id !== bindingTemplateId && t.button_key === button.key);
    if (existingTemplate) {
      alert(`按钮标识 "${button.key}" 已被模板 "${existingTemplate.name}" 绑定，请先解绑后再使用`);
      return;
    }

    try {
      await authFetch(`/api/platform/print-templates/${bindingTemplateId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: template.name,
          menu_route: resourcesWithButtons.find(r => r.id === resourceId)?.resource_key,
          button_key: button.key,
          button_name: button.name
        })
      });
      setShowBindModal(false);
      setBindingTemplateId(null);
      setResourcesWithButtons([]);
      await fetchData();
    } catch (err) {
      alert('绑定失败');
    }
  };

  // 保存模板修改
  const handleSaveTemplate = async () => {
    if (!univerRef.current || !previewTemplateId) return;

    const workbookData = univerRef.current.getRawData();
    if (!workbookData) {
      showToast('获取模板数据失败', 'error');
      return;
    }

    try {
      // gzip压缩Univer workbook数据
      const workbookJson = JSON.stringify(workbookData);
      const compressed = pako.gzip(workbookJson);
      const base64 = btoa(String.fromCharCode(...compressed));

      const res = await authFetch(`/api/platform/print-templates/${previewTemplateId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ template: 'UNIVER:' + base64 })
      });
      const result = await res.json();

      if (res.ok) {
        showToast('保存成功', 'success');
        fetchData();
      } else {
        showToast(result.error || '保存失败', 'error');
      }
    } catch (err) {
      showToast('保存失败', 'error');
    }
  };

  const filteredTemplates = printTemplates.filter(t =>
    !filters.name || t.name.toLowerCase().includes(filters.name.toLowerCase())
  );

  return (
    <div className="erp-legacy-page flex-1 flex flex-col min-h-0 space-y-8 animate-in fade-in duration-500 py-4 md:py-8 !w-full !max-w-none m-0 !p-0 relative">
      {/* 加载遮罩 */}
      {isRefreshing && (
        <div className="absolute inset-0 bg-white/80 z-50 flex items-center justify-center backdrop-blur-sm">
          <div className="flex items-center gap-3 text-zinc-600 bg-white px-6 py-3 rounded-xl shadow-lg border border-zinc-200">
            <RefreshCw className="w-5 h-5 animate-spin" />
            <span className="font-medium">数据加载中...</span>
          </div>
        </div>
      )}

      {/* 头部 */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 px-4 md:px-8">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-zinc-900 flex items-center gap-2">
            <FileText className="w-8 h-8" />
            打印模板
          </h2>
          <p className="text-zinc-500">管理打印模板配置</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              placeholder="搜索模板名称..."
              value={filters.name}
              onChange={(e) => setFilters({ ...filters, name: e.target.value })}
              className="pl-9 pr-4 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 w-64"
            />
          </div>
          <button
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="p-2 hover:bg-zinc-100 rounded-lg text-zinc-500 transition-colors disabled:opacity-50"
            title="刷新数据"
          >
            <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={() => handleOpenModal()}
            className="bg-zinc-900 text-white px-4 py-2 rounded-xl text-sm font-medium flex items-center gap-2 hover:bg-zinc-800 transition-colors"
          >
            <Plus className="w-4 h-4" />
            新建模板
          </button>
        </div>
      </div>

      {/* 表格 */}
      <div className="flex-1 min-h-0 bg-white rounded-none border-y border-zinc-200 overflow-auto">
        <table className="w-full text-left text-sm">
          <thead className="bg-zinc-50 border-b border-zinc-200">
            <tr>
              <th className="px-6 py-4 font-semibold text-zinc-500">模板名称</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">菜单路由</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">按钮标识</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">按钮名称</th>
              <th className="px-6 py-4 font-semibold text-zinc-500">Excel文件</th>
              <th className="px-6 py-4 font-semibold text-zinc-500 text-center">模板操作</th>
              <th className="px-6 py-4 font-semibold text-zinc-500 text-left">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {filteredTemplates.map((template) => (
              <tr key={template.id} className="hover:bg-zinc-50 transition-colors">
                <td className="px-6 py-4">
                  <span className="font-bold text-zinc-900">{template.name}</span>
                </td>
                <td className="px-6 py-4 text-zinc-600">{template.menu_route || '-'}</td>
                <td className="px-6 py-4">
                  {template.button_key ? (
                    <code className="bg-zinc-100 px-2 py-1 rounded text-xs font-mono text-zinc-600">
                      {template.button_key}
                    </code>
                  ) : '-'}
                </td>
                <td className="px-6 py-4">
                  <div className="flex items-center gap-2">
                    <span className="text-zinc-600">{template.button_name || '-'}</span>
                    <button
                      onClick={() => handleBindButton(template.id)}
                      className="inline-flex items-center gap-1 px-2 py-1 bg-zinc-50 border border-zinc-200 text-zinc-500 rounded text-xs hover:bg-zinc-100 transition-all"
                      title="绑定按钮"
                    >
                      绑定
                    </button>
                  </div>
                </td>
                <td className="px-6 py-4">
                  {template.excel_filename ? (
                    <span className="text-zinc-600 text-sm">{template.excel_filename}</span>
                  ) : (
                    <span className="text-zinc-400 text-sm">-</span>
                  )}
                </td>
                <td className="px-6 py-4 text-center">
                  <div className="flex items-center justify-center gap-2">
                    {template.template && (
                      <button
                        onClick={() => handlePreview(template.id, template.template)}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-zinc-50 border border-zinc-200 text-zinc-600 rounded-lg font-bold text-xs hover:bg-zinc-100 transition-all shadow-sm"
                      >
                        <Eye className="w-3.5 h-3.5" />
                        预览
                      </button>
                    )}
                    {template.excel_filename && (
                      <button
                        onClick={() => handleDownloadExcel(template.id, template.excel_filename || 'template.xlsx')}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-blue-50 border border-blue-200 text-blue-600 rounded-lg font-bold text-xs hover:bg-blue-100 transition-all shadow-sm"
                        title="下载原始Excel"
                      >
                        <Download className="w-3.5 h-3.5" />
                        下载
                      </button>
                    )}
                    <button
                      onClick={() => handleImportExcel(template.id)}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-emerald-50 border border-emerald-200 text-emerald-600 rounded-lg font-bold text-xs hover:bg-emerald-100 transition-all shadow-sm"
                      title="导入模板文件"
                    >
                      <Upload className="w-3.5 h-3.5" />
                      导入
                    </button>
                    {template.excel_filename && (
                      <button
                        onClick={() => handleTestPrint(template.id)}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-purple-50 border border-purple-200 text-purple-600 rounded-lg font-bold text-xs hover:bg-purple-100 transition-all shadow-sm"
                        title="测试打印"
                      >
                        <Printer className="w-3.5 h-3.5" />
                        打印
                      </button>
                    )}
                  </div>
                </td>
                <td className="px-6 py-4 text-left">
                  <div className="flex justify-start gap-2">
                    <button
                      onClick={() => handleOpenModal(template)}
                      className="p-2 hover:bg-white border hover:border-zinc-200 rounded-lg text-zinc-600 hover:text-zinc-900 transition-all font-bold text-xs flex items-center gap-1.5 shadow-sm"
                    >
                      <Settings className="w-3.5 h-3.5" />
                      修改
                    </button>
                    <button
                      onClick={() => setDeletingId(template.id)}
                      className="p-2 hover:bg-rose-50 border border-transparent hover:border-rose-100 rounded-lg text-zinc-400 hover:text-rose-600 transition-all"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {filteredTemplates.length === 0 && (
              <tr>
                <td colSpan={7} className="px-6 py-12 text-center text-zinc-400">
                  暂无模板，点击右上角新建
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* 新建/修改弹窗 */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg">
            <div className="p-6 border-b border-zinc-200">
              <h3 className="text-xl font-bold text-zinc-900">
                {editingId ? '修改模板' : '新建模板'}
              </h3>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-bold text-zinc-600 mb-2">模板名称 <span className="text-rose-500">*</span></label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  className="w-full px-4 py-2 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900"
                  placeholder="请输入模板名称"
                />
              </div>
            </div>
            <div className="p-6 border-t border-zinc-200 flex justify-end gap-3">
              <button
                onClick={() => setShowModal(false)}
                className="px-4 py-2 text-zinc-600 hover:text-zinc-900 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleSubmit}
                className="bg-zinc-900 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-zinc-800 transition-colors"
              >
                {editingId ? '保存' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 删除确认弹窗 */}
      {deletingId && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-sm">
            <div className="p-6">
              <h3 className="text-xl font-bold text-zinc-900 mb-2">确认删除</h3>
              <p className="text-zinc-500">删除后无法恢复，确定要删除该模板吗？</p>
            </div>
            <div className="p-6 border-t border-zinc-200 flex justify-end gap-3">
              <button
                onClick={() => setDeletingId(null)}
                className="px-4 py-2 text-zinc-600 hover:text-zinc-900 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleDelete}
                className="bg-rose-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-rose-700 transition-colors"
              >
                删除
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 预览弹窗 - 使用Univer */}
      {showPreviewModal && previewSheetData && (
        <div className="fixed inset-0 bg-black/50 z-[100] flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-6xl h-[85vh] overflow-hidden">
            <div className="p-4 border-b border-zinc-200 flex justify-between items-center">
              <h3 className="text-xl font-bold text-zinc-900">模板预览</h3>
              <div className="flex items-center gap-3">
                <button
                  onClick={handleSaveTemplate}
                  className="bg-emerald-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-emerald-700 transition-colors"
                >
                  保存修改
                </button>
                <button
                  onClick={() => {
                    setShowPreviewModal(false);
                    setPreviewTemplateId(null);
                    setPreviewSheetData(null);
                  }}
                  className="text-zinc-400 hover:text-zinc-600"
                >
                  ✕
                </button>
              </div>
            </div>
            <div className="h-[calc(85vh-60px)] w-full">
              <UniverSheet
                ref={univerRef}
                data={previewSheetData}
                height="100%"
              />
            </div>
          </div>
        </div>
      )}

      {/* 隐藏的Excel文件输入框 */}
      <input
        type="file"
        ref={excelInputRef}
        onChange={handleExcelFileChange}
        accept=".xlsx,.xls"
        className="hidden"
      />

      {/* 绑定按钮弹框 */}
      {showBindModal && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-3xl mx-4 max-h-[80vh] flex flex-col">
            <div className="px-6 py-4 border-b border-zinc-200 flex items-center justify-between">
              <h3 className="text-lg font-bold">绑定按钮 - {bindingTemplateName}</h3>
              <button
                onClick={() => { setShowBindModal(false); setBindingTemplateId(null); setResourcesWithButtons([]); }}
                className="text-zinc-400 hover:text-zinc-600"
              >
                ✕
              </button>
            </div>

            {bindLoading ? (
              <div className="flex-1 flex items-center justify-center">
                <RefreshCw className="w-6 h-6 animate-spin text-zinc-400" />
              </div>
            ) : (
              <div className="flex-1 overflow-auto p-6">
                <div className="grid grid-cols-2 gap-6 h-full min-h-[400px]">
                  {/* 左侧：当前绑定信息 */}
                  <div className="flex flex-col border border-zinc-200 rounded-xl overflow-hidden">
                    <div className="px-4 py-3 bg-zinc-50 border-b border-zinc-200">
                      <h4 className="text-sm font-bold text-zinc-700">当前绑定</h4>
                    </div>
                    <div className="flex-1 overflow-auto p-3">
                      {(() => {
                        const template = printTemplates.find(t => t.id === bindingTemplateId);
                        if (!template?.button_key) {
                          return <p className="text-sm text-zinc-400 text-center py-8">暂未绑定按钮</p>;
                        }
                        return (
                          <div className="flex items-center justify-between p-3 bg-zinc-50 rounded-lg">
                            <div>
                              <div className="font-medium text-sm">{template.button_name}</div>
                              <div className="text-xs text-zinc-400 mt-1">
                                <span className="font-mono">{template.button_key}</span>
                                <span className="mx-2">|</span>
                                <span>{template.menu_route}</span>
                              </div>
                            </div>
                            <button
                              onClick={async () => {
                                const template = printTemplates.find(t => t.id === bindingTemplateId);
                                if (!template) return;
                                try {
                                  await authFetch(`/api/platform/print-templates/${bindingTemplateId}`, {
                                    method: 'PATCH',
                                    headers: { 'Content-Type': 'application/json' },
                                    body: JSON.stringify({
                                      name: template.name,
                                      menu_route: '',
                                      button_key: '',
                                      button_name: ''
                                    })
                                  });
                                  await fetchData();
                                  setShowBindModal(false);
                                  setBindingTemplateId(null);
                                  setResourcesWithButtons([]);
                                } catch (err) {
                                  alert('解绑失败');
                                }
                              }}
                              className="p-1.5 hover:bg-red-100 rounded text-red-500 hover:text-red-700 transition-colors"
                              title="解绑"
                            >
                              <Unlink className="w-4 h-4" />
                            </button>
                          </div>
                        );
                      })()}
                    </div>
                  </div>

                  {/* 右侧：可绑定按钮 */}
                  <div className="flex flex-col border border-zinc-200 rounded-xl overflow-hidden">
                    <div className="px-4 py-3 bg-emerald-50 border-b border-zinc-200">
                      <h4 className="text-sm font-bold text-emerald-700">可绑定按钮</h4>
                    </div>
                    <div className="flex-1 overflow-auto p-3">
                      {resourcesWithButtons.length === 0 ? (
                        <p className="text-sm text-zinc-400 text-center py-8">暂无可绑定的按钮资源</p>
                      ) : (
                        <div className="space-y-3">
                          {resourcesWithButtons.map((resource) => (
                            <div key={resource.id} className="border border-zinc-200 rounded-lg overflow-hidden">
                              <div className="px-3 py-2 bg-emerald-50 border-b border-zinc-200">
                                <span className="font-bold text-xs text-emerald-700">{resource.name}</span>
                              </div>
                              <div className="p-2 space-y-1">
                                {resource.page_resources?.map((button) => {
                                  const boundTemplate = printTemplates.find(t => t.id !== bindingTemplateId && t.button_key === button.key);
                                  const isBound = !!boundTemplate;
                                  const isCurrentBound = printTemplates.find(t => t.id === bindingTemplateId)?.button_key === button.key;

                                  return (
                                    <div key={button.key} className={`flex items-center justify-between p-2 rounded transition-colors ${
                                      isCurrentBound ? 'bg-emerald-100' : isBound ? 'bg-zinc-100' : 'bg-white hover:bg-emerald-50'
                                    }`}>
                                      <div className="flex items-center gap-2">
                                        <span className={`font-medium text-sm ${isBound && !isCurrentBound ? 'text-zinc-400' : ''}`}>{button.name}</span>
                                        {isBound && !isCurrentBound && (
                                          <span className="text-xs text-zinc-400">(已绑定: {boundTemplate.name})</span>
                                        )}
                                        {isCurrentBound && (
                                          <span className="text-xs text-emerald-600">(当前绑定)</span>
                                        )}
                                      </div>
                                      {!isBound || isCurrentBound ? (
                                        <button
                                          onClick={() => handleSelectButton(resource.id, button)}
                                          className="p-1.5 hover:bg-emerald-100 rounded text-emerald-600 hover:text-emerald-800 transition-colors"
                                          title={isCurrentBound ? "重新绑定" : "绑定"}
                                        >
                                          <Link2 className="w-4 h-4" />
                                        </button>
                                      ) : (
                                        <span className="text-xs text-zinc-400 p-1.5">不可用</span>
                                      )}
                                    </div>
                                  );
                                })}
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div className="px-6 py-4 border-t border-zinc-200">
              <button
                onClick={() => { setShowBindModal(false); setBindingTemplateId(null); setResourcesWithButtons([]); }}
                className="w-full px-4 py-2 border border-zinc-200 rounded-xl text-sm font-medium hover:bg-zinc-50 transition-colors"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 测试打印弹框 */}
      {showTestPrintModal && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg">
            <div className="p-6 border-b border-zinc-200">
              <h3 className="text-xl font-bold text-zinc-900">测试打印 - {testPrintTemplateName}</h3>
              <p className="text-zinc-500 text-sm mt-1">输入JSON数据替换模板变量，生成Excel文件</p>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-bold text-zinc-600 mb-2">变量数据 (JSON格式)</label>
                <textarea
                  value={testPrintData}
                  onChange={(e) => setTestPrintData(e.target.value)}
                  className="w-full px-4 py-3 border border-zinc-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900 font-mono"
                  rows={8}
                  placeholder={'{"key1": "value1", "key2": "value2"}'}
                />
              </div>
              <div className="text-xs text-zinc-400">
                提示：模板中的 {"{{"}订单编号{"}}"} 将被替换为对应值
              </div>
            </div>
            <div className="p-6 border-t border-zinc-200 flex justify-end gap-3">
              <button
                onClick={() => setShowTestPrintModal(false)}
                className="px-4 py-2 text-zinc-600 hover:text-zinc-900 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleGenerateExcel}
                className="bg-purple-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-purple-700 transition-colors flex items-center gap-2"
              >
                <Printer className="w-4 h-4" />
                生成Excel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Toast提示 */}
      {toast && (
        <div
          className={`fixed inset-0 z-[200] flex items-center justify-center bg-black/30 animate-in fade-in duration-200`}
        >
          <div
            className={`px-8 py-4 rounded-xl shadow-xl flex items-center gap-3 animate-in zoom-in duration-300 ${
              toast.type === 'success'
                ? 'bg-emerald-500 text-white'
                : 'bg-red-500 text-white'
            }`}
          >
            {toast.type === 'success' ? (
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            ) : (
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            )}
            <span className="font-medium text-lg">{toast.message}</span>
          </div>
        </div>
      )}
    </div>
  );
};

export default PrintTemplates;