# Quickstart Validation: Animated ASCII Banner Experience

1. Build the CLI: `make build`.
2. Launch the TUI with default settings: `./bloco-eth --prefix cafe --progress`.
   - Verify the ASCII banner animates smoothly without stalling progress updates.
3. Disable animation via env var: `BLOCO_DISABLE_ANIMATION=1 ./bloco-eth --prefix cafe --progress`.
   - Confirm the banner renders statically with no animation artifacts.
4. Re-enable explicitly: `BLOCO_DISABLE_ANIMATION=1 ./bloco-eth --prefix cafe --progress --enable-animation`.
   - Expect animation to run despite the environment override.
5. Terminate the CLI (Ctrl+C) while animation is running.
   - Ensure the banner stops immediately and the process exits cleanly.
6. Run `go test -race ./internal/tui -run TestAnimatedBanner` to execute newly added tests verifying banner behavior and static fallback.
