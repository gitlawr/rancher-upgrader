package model

//ServiceUpgrade config
type ServiceUpgrade struct {
	ServiceSelector map[string]string `json:"serviceSelector,omitempty" mapstructure:"serviceSelector"`
	Tag             string            `json:"tag,omitempty" mapstructure:"tag"`
	BatchSize       int64             `json:"batchSize,omitempty" mapstructure:"batchSize"`
	IntervalMillis  int64             `json:"intervalMillis,omitempty" mapstructure:"intervalMillis"`
	StartFirst      bool              `json:"startFirst,omitempty" mapstructure:"startFirst"`
	Type            string            `json:"type,omitempty" mapstructure:"type"`
}
