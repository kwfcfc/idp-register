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

 ;; agent-shell, use direnv's exec to inherit/load flake's settings
 (nil
  . ((agent-shell-anthropic-claude-acp-command
      . ("direnv" "exec" "." "claude-agent-acp"))
     (agent-shell-openai-codex-acp-command
      . ("direnv" "exec" "." "codex-acp"))
     ;; OpenAI API key: decrypt on demand from sops (age recipient = ssh key,
     ;; decrypted with ~/.ssh/id_ed25519). The key only ever lives inside the
     ;; Emacs process and is never exported to the environment, so other
     ;; direnv consumers in this project never see it.
     (eval . (when (fboundp 'agent-shell-openai-make-authentication)
               (setq-local
                agent-shell-openai-authentication
                (agent-shell-openai-make-authentication
                 :api-key
                 (lambda ()
                   (let ((f (expand-file-name
                             "secrets.enc.yaml"
                             (locate-dominating-file
                              default-directory ".dir-locals.el"))))
                     (string-trim
                      (shell-command-to-string
                       (format "sops -d --extract '[\"OPENAI_API_KEY\"]' %s"
                               (shell-quote-argument f))))))))))))
 )
