import React from 'react';
import { motion } from 'motion/react';
import { X, Download } from 'lucide-react';

interface DrawingViewerModalProps {
  show: string | null;
  onClose: () => void;
}

const DrawingViewerModal: React.FC<DrawingViewerModalProps> = ({ show, onClose }) => {
  if (!show) return null;

  return (
    <div className="fixed inset-0 z-[120] flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        className="absolute inset-0 bg-zinc-900/90 backdrop-blur-sm"
      />
      <motion.div
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.9 }}
        className="relative max-w-4xl w-full bg-white rounded-3xl overflow-hidden shadow-2xl"
      >
        <div className="p-4 border-b border-zinc-100 flex items-center justify-between">
          <h3 className="font-bold">图纸预览</h3>
          <div className="flex gap-2">
            <button className="p-2 hover:bg-zinc-100 rounded-lg text-zinc-500">
              <Download className="w-5 h-5" />
            </button>
            <button onClick={onClose} className="p-2 hover:bg-zinc-100 rounded-lg">
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>
        <div className="p-4 bg-zinc-100 flex items-center justify-center min-h-[400px]">
          <img src={show} alt="Drawing" className="max-w-full max-h-[70vh] object-contain shadow-lg" />
        </div>
      </motion.div>
    </div>
  );
};

export default DrawingViewerModal;
