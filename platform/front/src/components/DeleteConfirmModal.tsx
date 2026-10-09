import React from 'react';
import { motion } from 'motion/react';
import { Trash2 } from 'lucide-react';

interface DeleteConfirmModalProps {
  show: boolean;
  title?: string;
  message?: string;
  onClose: () => void;
  onConfirm: () => void;
}

const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  show,
  title = '确认删除',
  message = '确定要删除吗？此操作不可撤销。',
  onClose,
  onConfirm
}) => {
  if (!show) return null;

  return (
    <div className="fixed inset-0 z-[140] flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        className="absolute inset-0 bg-zinc-900/60 backdrop-blur-sm"
      />
      <motion.div
        initial={{ opacity: 0, scale: 0.95, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.95, y: 20 }}
        className="relative w-full max-w-sm bg-white rounded-2xl shadow-2xl p-6"
      >
        <div className="text-center">
          <div className="w-12 h-12 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <Trash2 className="w-6 h-6 text-red-600" />
          </div>
          <h3 className="text-lg font-bold text-zinc-900 mb-2">{title}</h3>
          <p className="text-zinc-500 text-sm mb-6">{message}</p>
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-zinc-200 rounded-xl text-zinc-600 hover:bg-zinc-50 transition-colors"
            >
              取消
            </button>
            <button
              onClick={onConfirm}
              className="flex-1 px-4 py-2 bg-red-600 text-white rounded-xl hover:bg-red-700 transition-colors"
            >
              删除
            </button>
          </div>
        </div>
      </motion.div>
    </div>
  );
};

export default DeleteConfirmModal;
