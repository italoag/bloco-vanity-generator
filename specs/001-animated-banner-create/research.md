# Research: Animated ASCII Banner Experience

## 1. Harmonica Integration Strategy
- **Decision**: Use Charmbracelet Harmonica to interpolate frames for the existing `internal/tui/logo.go` ASCII art, rendering via the TUI renderer.
- **Rationale**: Harmonica provides easing utilities and frame composers purpose-built for terminal animations, reducing the need for custom timing logic.
- **Alternatives Considered**:
  - Manual ticker-based animation: rejected due to added maintenance and easing complexity.
  - Other Charmbracelet libraries (e.g., Lip Gloss): useful for styling but lack animation orchestration.

## 2. Motion Reduction Detection
- **Decision**: Default to static banner when environment variables such as `NO_COLOR`, `BLOCO_DISABLE_ANIMATION`, or `CLICOLOR=0` signal reduced motion, while allowing an explicit `--enable-animation` flag to override.
- **Rationale**: Aligns with accessibility expectations and constitution Principle III (non-blocking, responsive UI).
- **Alternatives Considered**:
  - Always animate unless `--no-animation` flag set: rejected because it ignores platform standards for reduced motion preferences.
  - Rely solely on terminal capability detection: insufficient for honoring user accessibility choices.

## 3. Performance & Cancellation Safeguards
- **Decision**: Run animation updates on a dedicated goroutine with context propagation, throttled to ≤30 FPS, and include benchmark comparisons to ensure ≤5% CPU overhead.
- **Rationale**: Keeps wallet generation responsive (Principle III) and ensures the animation can stop instantly during shutdown.
- **Alternatives Considered**:
  - Higher FPS (60): rejected for unnecessary CPU cost.
  - Shared goroutine with progress updates: risks blocking statistics channels and complicates cancellation.
