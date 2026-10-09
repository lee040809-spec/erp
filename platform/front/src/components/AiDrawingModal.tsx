import React from 'react';
import { motion } from 'motion/react';
import { X, Sparkles, Image as ImageIcon, CheckCircle2 } from 'lucide-react';

interface AiDrawingModalProps {
  show: boolean;
  onClose: () => void;
  aiPrompt: string;
  setAiPrompt: (prompt: string) => void;
  isGenerating: boolean;
  generatedImage: string | null;
  onGenerate: () => void;
  onUseImage: (imageData: string) => void;
}

const AiDrawingModal: React.FC<AiDrawingModalProps> = ({
  show,
  onClose,
  aiPrompt,
  setAiPrompt,
  isGenerating,
  generatedImage,
  onGenerate,
  onUseImage
}) => {
  if (!show) return null;

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        className="absolute inset-0 bg-zinc-900/80 backdrop-blur-md"
      />
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.95 }}
        className="relative w-full max-w-2xl bg-white rounded-3xl shadow-2xl overflow-hidden"
      >
        <div className="p-6 border-b border-zinc-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-zinc-900" />
            <h3 className="text-xl font-bold">AI 图纸助手</h3>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-zinc-100 rounded-full">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-8 space-y-6">
          <div className="space-y-2">
            <label className="text-sm font-bold text-zinc-500 uppercase">描述零件特征</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={aiPrompt}
                onChange={(e) => setAiPrompt(e.target.value)}
                placeholder="例如：一个带4个孔的铝合金法兰盘，直径100mm..."
                className="flex-1 px-4 py-3 bg-zinc-50 border border-zinc-200 rounded-2xl focus:ring-2 focus:ring-zinc-900 outline-none"
              />
              <button
                onClick={onGenerate}
                disabled={isGenerating || !aiPrompt}
                className="bg-zinc-900 text-white px-6 py-3 rounded-2xl font-bold disabled:opacity-50 flex items-center gap-2"
              >
                {isGenerating ? '生成中...' : '生成图纸'}
              </button>
            </div>
          </div>

          <div className="aspect-square bg-zinc-50 rounded-2xl border-2 border-dashed border-zinc-200 flex items-center justify-center overflow-hidden relative">
            {isGenerating ? (
              <div className="flex flex-col items-center gap-4">
                <div className="w-12 h-12 border-4 border-zinc-900 border-t-transparent rounded-full animate-spin" />
                <p className="text-sm font-medium text-zinc-500">正在构思图纸细节...</p>
              </div>
            ) : generatedImage ? (
              <img src={generatedImage} alt="AI Generated" className="w-full h-full object-contain" />
            ) : (
              <div className="text-center space-y-2">
                <ImageIcon className="w-12 h-12 text-zinc-200 mx-auto" />
                <p className="text-sm text-zinc-400">输入描述并点击生成</p>
              </div>
            )}
          </div>

          {generatedImage && (
            <button
              onClick={() => onUseImage(generatedImage)}
              className="w-full bg-emerald-600 text-white py-4 rounded-2xl font-bold hover:bg-emerald-700 transition-colors flex items-center justify-start gap-2"
            >
              <CheckCircle2 className="w-5 h-5" />
              使用此图纸
            </button>
          )}
        </div>
      </motion.div>
    </div>
  );
};

export default AiDrawingModal;
