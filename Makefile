COMPLETIONS_DIR := $(HOME)/.snowctl/completions

.PHONY: completion-bash completion-zsh completion-fish completion-powershell completions

completion-bash:
	@mkdir -p $(COMPLETIONS_DIR)
	@go run ./cmd/snowctl completion bash > $(COMPLETIONS_DIR)/snowctl.bash
	@echo "Bash completion written to $(COMPLETIONS_DIR)/snowctl.bash"

completion-zsh:
	@mkdir -p $(COMPLETIONS_DIR)
	@go run ./cmd/snowctl completion zsh > $(COMPLETIONS_DIR)/_snowctl
	@echo "Zsh completion written to $(COMPLETIONS_DIR)/_snowctl"

completion-fish:
	@mkdir -p $(COMPLETIONS_DIR)
	@go run ./cmd/snowctl completion fish > $(COMPLETIONS_DIR)/snowctl.fish
	@echo "Fish completion written to $(COMPLETIONS_DIR)/snowctl.fish"

completion-powershell:
	@mkdir -p $(COMPLETIONS_DIR)
	@go run ./cmd/snowctl completion powershell > $(COMPLETIONS_DIR)/snowctl.ps1
	@echo "PowerShell completion written to $(COMPLETIONS_DIR)/snowctl.ps1"

completions: completion-bash completion-zsh completion-fish completion-powershell
