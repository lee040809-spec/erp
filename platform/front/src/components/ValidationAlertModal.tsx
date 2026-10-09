import React from 'react';
import { motion } from 'motion/react';
import { AlertCircle } from 'lucide-react';

interface ValidationAlertModalProps {
  show: string | null;
  onClose: () => void;
  title?: string;
  buttonText?: string;
}

const ValidationAlertModal: React.FC<ValidationAlertModalProps> = ({
  show,
  onClose,
  title = '提示',
  buttonText = '我知道了'
}) => {
  if (!show) return null;

  return (
    <div className="fixed inset-0 z-[700] flex items-center justify-center p-4">
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
        className="relative w-full max-w-sm bg-white rounded-3xl shadow-2xl border border-zinc-200 p-8 overflow-hidden"
      >
        <div className="flex flex-col items-center text-center space-y-4">
          <div className="w-16 h-16 bg-amber-50 rounded-2xl flex items-center justify-center">
            <AlertCircle className="w-8 h-8 text-amber-500" />
          </div>
          <div className="space-y-2">
            <h3 className="text-xl font-bold text-zinc-900">{title}</h3>
            <p className="text-sm text-zinc-500 font-medium leading-relaxed">{show}</p>
          </div>
          <button
            onClick={onClose}
            className="w-full mt-4 px-6 py-3 bg-zinc-900 text-white rounded-2xl font-bold hover:bg-zinc-800 transition-colors shadow-lg shadow-zinc-200"
          >
            {buttonText}
          </button>
        </div>
      </motion.div>
    </div>
  );
};

export default ValidationAlertModal;
