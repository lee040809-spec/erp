import React from 'react';
import { motion } from 'motion/react';
import { AlertCircle } from 'lucide-react';

interface ErrorModalProps {
  show: boolean;
  title?: string;
  message?: string;
  onClose: () => void;
}

const ErrorModal: React.FC<ErrorModalProps> = ({
  show,
  title = '操作失败',
  message = '发生错误，请稍后重试。',
  onClose
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
            <AlertCircle className="w-6 h-6 text-red-600" />
          </div>
          <h3 className="text-lg font-bold text-zinc-900 mb-2">{title}</h3>
          <p className="text-zinc-500 text-sm mb-6">{message}</p>
          <button
            onClick={onClose}
            className="w-full px-4 py-2 bg-zinc-900 text-white rounded-xl hover:bg-zinc-800 transition-colors"
          >
            确定
          </button>
        </div>
      </motion.div>
    </div>
  );
};

export default ErrorModal;