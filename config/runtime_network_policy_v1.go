package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const QCRuntimeNetworkPolicyVersionV1 = "QC_RUNTIME_NETWORK_POLICY_V1"

type EffectiveDifficultyInfoV1 struct {
	Profile         NetworkProfileV1 `json:"profile"`
	BaseBits        int              `json:"base_bits"`
	DynamicBits     int              `json:"dynamic_bits"`
	EffectiveBits   int              `json:"effective_bits"`
	OverrideApplied bool             `json:"override_applied"`
	OverrideSource  string           `json:"override_source,omitempty"`
}

// ValidateRuntimeNetworkProfileV1 validates the complete runtime
// boundary before node, API, P2P, or mining services are started.
func ValidateRuntimeNetworkProfileV1(c *Config) error {
	if c == nil {
		return fmt.Errorf(
			"%s: config cannot be nil",
			QCRuntimeNetworkPolicyVersionV1,
		)
	}

	profile, err := ResolveNetworkProfileV1(c.ChainFile)
	if err != nil {
		return fmt.Errorf(
			"%s: resolve profile: %w",
			QCRuntimeNetworkPolicyVersionV1,
			err,
		)
	}

	gotChain := strings.ToLower(
		filepath.Base(strings.TrimSpace(c.ChainFile)),
	)
	wantChain := strings.ToLower(
		filepath.Base(profile.ExpectedChainFile),
	)

	if gotChain != wantChain {
		return fmt.Errorf(
			"%s: profile %s requires chain file %s, got %s",
			QCRuntimeNetworkPolicyVersionV1,
			profile.Profile,
			profile.ExpectedChainFile,
			c.ChainFile,
		)
	}

	gotHTTP := normalizeRuntimePortV1(c.HTTPPort)
	wantHTTP := normalizeRuntimePortV1(
		profile.ExpectedHTTPPort,
	)

	if gotHTTP != wantHTTP {
		return fmt.Errorf(
			"%s: profile %s requires HTTP port %s, got %s",
			QCRuntimeNetworkPolicyVersionV1,
			profile.Profile,
			wantHTTP,
			gotHTTP,
		)
	}

	gotP2P := normalizeRuntimePortV1(c.P2PPort)
	wantP2P := normalizeRuntimePortV1(
		profile.ExpectedP2PPort,
	)

	if gotP2P != wantP2P {
		return fmt.Errorf(
			"%s: profile %s requires P2P port %s, got %s",
			QCRuntimeNetworkPolicyVersionV1,
			profile.Profile,
			wantP2P,
			gotP2P,
		)
	}

	if c.DefaultDifficultyBits !=
		profile.CanonicalDefaultDifficultyBits {
		return fmt.Errorf(
			"%s: profile %s requires base difficulty %d, got %d",
			QCRuntimeNetworkPolicyVersionV1,
			profile.Profile,
			profile.CanonicalDefaultDifficultyBits,
			c.DefaultDifficultyBits,
		)
	}

	if err := validateRuntimeNetworkMagicV1(profile); err != nil {
		return err
	}

	if _, err := ResolveEffectiveHTTPPortV1(c); err != nil {
		return err
	}

	if _, err := ResolveEffectiveDifficultyBitsV1(
		c,
		c.DefaultDifficultyBits,
	); err != nil {
		return err
	}

	return nil
}

// ResolveEffectiveDifficultyBitsV1 is the single policy for combining
// canonical base difficulty, dynamic chain adjustment, and the temporary
// local-release fixed override.
func ResolveEffectiveDifficultyBitsV1(
	c *Config,
	dynamicBits int,
) (EffectiveDifficultyInfoV1, error) {
	if c == nil {
		return EffectiveDifficultyInfoV1{}, fmt.Errorf(
			"%s: config cannot be nil",
			QCRuntimeNetworkPolicyVersionV1,
		)
	}

	profile, err := ResolveNetworkProfileV1(c.ChainFile)
	if err != nil {
		return EffectiveDifficultyInfoV1{}, fmt.Errorf(
			"%s: resolve profile: %w",
			QCRuntimeNetworkPolicyVersionV1,
			err,
		)
	}

	if dynamicBits == 0 {
		dynamicBits = c.DefaultDifficultyBits
	}

	if dynamicBits <= 0 || dynamicBits > 255 {
		return EffectiveDifficultyInfoV1{}, fmt.Errorf(
			"%s: dynamic difficulty must be 1..255, got %d",
			QCRuntimeNetworkPolicyVersionV1,
			dynamicBits,
		)
	}

	info := EffectiveDifficultyInfoV1{
		Profile:       profile.Profile,
		BaseBits:      c.DefaultDifficultyBits,
		DynamicBits:   dynamicBits,
		EffectiveBits: dynamicBits,
	}

	rawOverride := strings.TrimSpace(
		os.Getenv("QC_DIFFICULTY_BITS"),
	)

	if rawOverride == "" {
		return info, nil
	}

	overrideBits, err := strconv.Atoi(rawOverride)
	if err != nil || overrideBits <= 0 || overrideBits > 30 {
		return EffectiveDifficultyInfoV1{}, fmt.Errorf(
			"%s: invalid QC_DIFFICULTY_BITS=%q; expected 1..30",
			QCRuntimeNetworkPolicyVersionV1,
			rawOverride,
		)
	}

	if !profile.AllowsFixedDifficultyOverride {
		return EffectiveDifficultyInfoV1{}, fmt.Errorf(
			"%s: profile %s forbids QC_DIFFICULTY_BITS override",
			QCRuntimeNetworkPolicyVersionV1,
			profile.Profile,
		)
	}

	info.EffectiveBits = overrideBits
	info.OverrideApplied = true
	info.OverrideSource = "QC_DIFFICULTY_BITS"

	return info, nil
}

// ResolveEffectiveHTTPPortV1 makes the canonical config port authoritative.
// The legacy HTTP_PORT value may match it for compatibility, but may not
// silently redirect the process to another port.
func ResolveEffectiveHTTPPortV1(
	c *Config,
) (string, error) {
	if c == nil {
		return "", fmt.Errorf(
			"%s: config cannot be nil",
			QCRuntimeNetworkPolicyVersionV1,
		)
	}

	canonical := normalizeRuntimePortV1(c.HTTPPort)
	if canonical == "" {
		return "", fmt.Errorf(
			"%s: canonical HTTP port is empty",
			QCRuntimeNetworkPolicyVersionV1,
		)
	}

	legacy := normalizeRuntimePortV1(
		os.Getenv("HTTP_PORT"),
	)

	if legacy != "" && legacy != canonical {
		return "", fmt.Errorf(
			"%s: legacy HTTP_PORT=%s conflicts with canonical HTTP port %s",
			QCRuntimeNetworkPolicyVersionV1,
			legacy,
			canonical,
		)
	}

	return canonical, nil
}

func validateRuntimeNetworkMagicV1(
	profile NetworkProfileInfoV1,
) error {
	magic := strings.TrimSpace(
		os.Getenv("QC_NET_MAGIC"),
	)

	if profile.IsTestnet {
		if magic != QCTestnetMagicV1 {
			return fmt.Errorf(
				"%s: profile %s requires QC_NET_MAGIC=%s, got %q",
				QCRuntimeNetworkPolicyVersionV1,
				profile.Profile,
				QCTestnetMagicV1,
				magic,
			)
		}

		return nil
	}

	if magic != "" && magic != QCMainnetMagicV1 {
		return fmt.Errorf(
			"%s: profile %s requires empty QC_NET_MAGIC or %s, got %q",
			QCRuntimeNetworkPolicyVersionV1,
			profile.Profile,
			QCMainnetMagicV1,
			magic,
		)
	}

	return nil
}

func normalizeRuntimePortV1(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, ":") {
		return value
	}

	return ":" + value
}
