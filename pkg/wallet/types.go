package wallet

import (
	"time"
)

// Wallet represents an Ethereum wallet with address and private key
type Wallet struct {
	Address    string    `json:"address"`
	PublicKey  string    `json:"public_key"`
	PrivateKey string    `json:"private_key"`
	Mnemonic   string    `json:"mnemonic,omitempty"`
	Network    string    `json:"network,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// GenerationResult represents the result of wallet generation
type GenerationResult struct {
	Wallet   *Wallet       `json:"wallet,omitempty"`
	Attempts int64         `json:"attempts"`
	Duration time.Duration `json:"duration"`
	Error    error         `json:"error,omitempty"`
	WorkerID int           `json:"worker_id,omitempty"`
}

// GenerationCriteria defines the criteria for wallet generation
type GenerationCriteria struct {
	Network       string `json:"network"`
	Prefix        string `json:"prefix"`
	Suffix        string `json:"suffix"`
	IsChecksum    bool   `json:"is_checksum"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
	UseMnemonic   bool   `json:"use_mnemonic,omitempty"`
	MaxAttempts   int64  `json:"max_attempts,omitempty"`
}

// GenerationRequest represents a request for wallet generation
type GenerationRequest struct {
	Criteria     GenerationCriteria `json:"criteria"`
	Count        int                `json:"count"`
	ShowProgress bool               `json:"show_progress"`
	Timeout      time.Duration      `json:"timeout,omitempty"`
}

// GenerationStats holds statistics about wallet generation
type GenerationStats struct {
	Pattern         string        `json:"pattern"`
	Difficulty      float64       `json:"difficulty"`
	Probability50   int64         `json:"probability_50"`
	CurrentAttempts int64         `json:"current_attempts"`
	Speed           float64       `json:"speed"`
	Probability     float64       `json:"probability"`
	EstimatedTime   time.Duration `json:"estimated_time"`
	StartTime       time.Time     `json:"start_time"`
	LastUpdate      time.Time     `json:"last_update"`
	IsChecksum      bool          `json:"is_checksum"`
}

// BenchmarkResult holds benchmark statistics
type BenchmarkResult struct {
	Engine                 string          `json:"engine,omitempty"`
	RequestedEngine        string          `json:"requested_engine,omitempty"`
	FallbackReason         string          `json:"fallback_reason,omitempty"`
	DeviceName             string          `json:"device_name,omitempty"`
	Network                string          `json:"network,omitempty"`
	Pattern                string          `json:"pattern,omitempty"`
	BatchSize              int             `json:"batch_size,omitempty"`
	IsSynthetic            bool            `json:"synthetic,omitempty"`
	IsHybrid               bool            `json:"hybrid,omitempty"`
	Matches                int64           `json:"matches"`
	TotalAttempts          int64           `json:"total_attempts"`
	TotalDuration          time.Duration   `json:"total_duration"`
	AverageSpeed           float64         `json:"average_speed"`
	MinSpeed               float64         `json:"min_speed"`
	MaxSpeed               float64         `json:"max_speed"`
	SpeedSamples           []float64       `json:"speed_samples"`
	DurationSamples        []time.Duration `json:"duration_samples"`
	SingleThreadSpeed      float64         `json:"single_thread_speed"`
	ThreadCount            int             `json:"thread_count"`
	ScalabilityEfficiency  float64         `json:"scalability_efficiency"`
	ThreadBalanceScore     float64         `json:"thread_balance_score"`
	ThreadUtilization      float64         `json:"thread_utilization"`
	SpeedupVsSingleThread  float64         `json:"speedup_vs_single_thread"`
	AmdahlsLawLimit        float64         `json:"amdahls_law_limit"`
	EntropyDuration        time.Duration   `json:"entropy_duration,omitempty"`
	ScalarBaseMultDuration time.Duration   `json:"scalar_base_mult_duration,omitempty"`
	HashFormatDuration     time.Duration   `json:"hash_format_duration,omitempty"`
	MatchDuration          time.Duration   `json:"match_duration,omitempty"`
	MetalBufferDuration    time.Duration   `json:"metal_buffer_duration,omitempty"`
	KernelDuration         time.Duration   `json:"kernel_duration,omitempty"`
}

// IsValid checks if a wallet is valid
func (w *Wallet) IsValid() bool {
	return w != nil &&
		len(w.Address) == 40 &&
		len(w.PrivateKey) == 64 &&
		!w.CreatedAt.IsZero()
}

// GetChecksumAddress returns the address with proper checksum formatting
func (w *Wallet) GetChecksumAddress() string {
	if w == nil {
		return ""
	}
	return "0x" + w.Address
}

// IsSuccessful checks if a generation result was successful
func (gr *GenerationResult) IsSuccessful() bool {
	return gr.Error == nil && gr.Wallet != nil && gr.Wallet.IsValid()
}

// GetPattern returns the combined pattern from criteria
func (gc *GenerationCriteria) GetPattern() string {
	return gc.Prefix + gc.Suffix
}

// GetPatternLength returns the total length of the pattern
func (gc *GenerationCriteria) GetPatternLength() int {
	return len(gc.Prefix) + len(gc.Suffix)
}

// IsEmpty checks if the criteria has any pattern requirements
func (gc *GenerationCriteria) IsEmpty() bool {
	return gc.Prefix == "" && gc.Suffix == ""
}

// Validate checks if the generation criteria is valid
func (gc *GenerationCriteria) Validate() error {
	// Pattern length validation
	patternLength := gc.GetPatternLength()
	if patternLength > 20 { // Reasonable limit to prevent extremely long generation times
		return NewValidationError("criteria_validation",
			"pattern too long (max 20 characters)")
	}

	// Hex validation
	if !isValidHex(gc.Prefix) {
		return NewValidationError("criteria_validation",
			"prefix contains invalid hex characters")
	}

	if !isValidHex(gc.Suffix) {
		return NewValidationError("criteria_validation",
			"suffix contains invalid hex characters")
	}

	if gc.CaseSensitive && !gc.IsChecksum {
		return NewValidationError("criteria_validation",
			"case-sensitive matching requires checksum validation")
	}

	if gc.CaseSensitive && gc.Network != "" && gc.Network != "ethereum" {
		return NewValidationError("criteria_validation",
			"case-sensitive matching is only supported for ethereum")
	}

	// Max attempts validation
	if gc.MaxAttempts < 0 {
		return NewValidationError("criteria_validation",
			"max attempts cannot be negative")
	}

	return nil
}

// Update updates the generation stats with new attempt count
func (gs *GenerationStats) Update(attempts int64) {
	gs.CurrentAttempts = attempts
	gs.Probability = computeProbability(gs.Difficulty, attempts) * 100

	now := time.Now()
	elapsed := now.Sub(gs.StartTime)

	if elapsed.Seconds() > 0 {
		gs.Speed = float64(attempts) / elapsed.Seconds()

		if gs.Probability50 > 0 && gs.Speed > 0 {
			remainingAttempts := gs.Probability50 - attempts
			if remainingAttempts > 0 {
				gs.EstimatedTime = time.Duration(float64(remainingAttempts)/gs.Speed) * time.Second
			} else {
				gs.EstimatedTime = 0
			}
		}
	}

	gs.LastUpdate = now
}

// Helper functions

// isValidHex checks if a string contains only valid hex characters
func isValidHex(hex string) bool {
	if len(hex) == 0 {
		return true
	}
	for _, char := range hex {
		if (char < '0' || char > '9') &&
			(char < 'a' || char > 'f') &&
			(char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

// computeProbability calculates the probability of finding an address after N attempts
func computeProbability(difficulty float64, attempts int64) float64 {
	if difficulty <= 0 {
		return 0
	}
	return 1 - pow(1-1/difficulty, float64(attempts))
}

// pow is a simple power function to avoid importing math
func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// Import the validation error from errors package
// This would normally be imported, but for this example we'll define it locally
type ValidationError struct {
	Operation string
	Message   string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(operation, message string) *ValidationError {
	return &ValidationError{
		Operation: operation,
		Message:   message,
	}
}
