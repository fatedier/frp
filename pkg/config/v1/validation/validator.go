package validation

import (
	"fmt"

	"github.com/fatedier/frp/pkg/policy/security"
)

// ConfigValidator holds the context dependencies for configuration validation.
type ConfigValidator struct {
	unsafeFeatures *security.UnsafeFeatures
}

// NewConfigValidator creates a new ConfigValidator instance.
func NewConfigValidator(unsafeFeatures *security.UnsafeFeatures) *ConfigValidator {
	return &ConfigValidator{
		unsafeFeatures: unsafeFeatures,
	}
}

// ValidateUnsafeFeature checks if a specific unsafe feature is enabled.
func (v *ConfigValidator) ValidateUnsafeFeature(feature string) error {
	if !v.unsafeFeatures.IsEnabled(feature) {
		return fmt.Errorf("unsafe feature %q is not enabled. "+
			"To enable it, ensure it is allowed in the configuration or command line flags", feature)
	}
	return nil
}
