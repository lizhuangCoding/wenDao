import { motion, AnimatePresence } from 'framer-motion';
import { useTranslation } from 'react-i18next';

interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  onConfirm: () => void;
  onCancel: () => void;
  isDanger?: boolean;
}

export const ConfirmModal = ({
  isOpen,
  title,
  message,
  confirmText,
  cancelText,
  onConfirm,
  onCancel,
  isDanger = false,
}: ConfirmModalProps) => {
  const { t } = useTranslation();

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onCancel}
            className="fixed inset-0 bg-neutral-900/40 backdrop-blur-sm z-[999]"
          />

          {/* Modal */}
          <div className="fixed inset-0 flex items-center justify-center z-[1000] p-6">
            <motion.div
              initial={{ opacity: 0, scale: 0.9, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.9, y: 20 }}
              className="w-full max-w-sm bg-white dark:bg-neutral-800 rounded-[32px] shadow-elevated overflow-hidden border border-neutral-100 dark:border-neutral-700"
            >
              <div className="p-8">
                <h3 className="text-xl font-serif font-black text-neutral-900 dark:text-neutral-100 mb-3">
                  {title}
                </h3>
                <p className="text-sm text-neutral-500 dark:text-neutral-400 font-medium leading-relaxed mb-8">
                  {message}
                </p>

                <div className="flex gap-3">
                  <button
                    onClick={onCancel}
                    className="flex-1 px-6 py-3 rounded-2xl text-[10px] font-black tracking-widest uppercase transition-all border border-neutral-100 dark:border-neutral-700 text-neutral-400 hover:bg-neutral-50 dark:hover:bg-neutral-700"
                  >
                    {cancelText || t('common.cancel')}
                  </button>
                  <button
                    onClick={onConfirm}
                    className={`flex-1 px-6 py-3 rounded-2xl text-[10px] font-black tracking-widest uppercase transition-all shadow-soft active:scale-95 text-white ${
                      isDanger
                        ? 'bg-red-500 hover:bg-red-600 shadow-red-200 dark:shadow-none'
                        : 'bg-neutral-900 dark:bg-primary-600 hover:bg-primary-600'
                    }`}
                  >
                    {confirmText || t('common.confirm')}
                  </button>
                </div>
              </div>
            </motion.div>
          </div>
        </>
      )}
    </AnimatePresence>
  );
};
