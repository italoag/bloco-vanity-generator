# Data Model: Animated ASCII Banner Experience

## Entities

### AnimatedBannerSequence
- **Description**: Defines the generated frames and timing configuration for the ASCII logo animation.
- **Fields**:
  - `frames []string`: Rendered ASCII frames with consistent width.
  - `frameDuration time.Duration`: Duration per frame (capped to 33ms for 30 FPS).
  - `easingProfile string`: Named easing function applied (e.g., "ease-in-out").
  - `loop bool`: Whether the animation repeats indefinitely.
- **Relationships**: Consumed by the banner renderer (TUI layer only).

### BannerPreferences
- **Description**: Captures runtime preferences controlling animation behavior.
- **Fields**:
  - `enabled bool`: Final resolved state after flags/env evaluation.
  - `source string`: Reason for the resolved state (flag, env, default).
  - `maxCpuOverhead float64`: Upper bound target (0.05).
- **Relationships**: Derived from CLI flags (`internal/cli`) and environment detection utilities.

### AnimationContext
- **Description**: Bundles dependencies needed to drive the animation safely.
- **Fields**:
  - `ctx context.Context`: Propagated cancellation token.
  - `frameTicker *time.Ticker`: Governs frame cadence.
  - `output io.Writer`: Destination writer for TUI banner region.
  - `metricsCollector MetricsRecorder`: Optional hook to record animation performance.
- **Relationships**: Owned by the TUI manager; ensures graceful shutdown alongside worker pool.
