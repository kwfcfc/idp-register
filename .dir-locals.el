((prog-mode . ((indent-tabs-mode . nil)
         (tab-width . 4)
         (standard-indent . 4)
         (require-final-newline . t)
         ;; (eval . (progn
         ;;           (set-buffer-file-coding-system 'utf-8-unix)
         ;;           (add-hook 'before-save-hook
         ;;                     #'delete-trailing-whitespace
         ;;                     nil
         ;;                     t)))
         ))

 ;; JavaScript
 (js-mode . ((js-indent-level . 4)))
 (js-ts-mode . ((js-indent-level . 4)))

 ;; TypeScript
 (typescript-mode . ((typescript-indent-level . 4)))
 (typescript-ts-mode . ((typescript-ts-mode-indent-offset . 4)))
 (tsx-ts-mode . ((typescript-ts-mode-indent-offset . 4)))

 ;; Svelte
 (svelte-mode . ((svelte-basic-offset . 4)))

 ;; agent-shell
 (nil . ((eval . (setq-local agent-shell-anthropic-claude-environment
                             (agent-shell-make-environment-variables :inherit-env t)))
         (eval . (setq-local agent-shell-openai-environment
                             (agent-shell-make-environment-variables :inherit-env t)))))
 )
