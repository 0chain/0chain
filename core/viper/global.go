package viper

import (
	"io"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	// SupportedExts describes the supported configuration formats
	// for the wrapper to avoid importing the spf13/viper package.
	// NOTE: This is a copy of spf13/viper.SupportedExts,
	// with the exception of the "ini" format, since its marshaller
	// does not have a thread-safe implementation.
	SupportedExts = []string{"dotenv", "env", "hcl", "json", "properties", "props", "prop", "toml", "yaml", "yml"}

	// vi stores the Viper global instance.
	vi = &Viper{viper: viper.GetViper()}
)

// AddConfigPath wraps viper's method.
func AddConfigPath(in string) {
	vi.AddConfigPath(in)
}

// AddRemoteProvider wraps viper's method.
func AddRemoteProvider(provider, endpoint, path string) error {
	return vi.AddRemoteProvider(provider, endpoint, path)
}

// AddSecureRemoteProvider wraps viper's method.
func AddSecureRemoteProvider(provider, endpoint, path, secret string) error {
	return vi.AddSecureRemoteProvider(provider, endpoint, path, secret)
}

// AllKeys wraps viper's method.
func AllKeys() []string {
	return vi.AllKeys()
}

// AllSettings wraps viper's method.
func AllSettings() map[string]interface{} {
	return vi.AllSettings()
}

// BindPFlags wraps viper's method.
func BindPFlags(flags *pflag.FlagSet) error {
	return vi.BindPFlags(flags)
}

// BindPFlag wraps viper's method.
func BindPFlag(key string, flag *pflag.Flag) error {
	return vi.BindPFlag(key, flag)
}

// BindFlagValues wraps viper's method.
func BindFlagValues(flags viper.FlagValueSet) error {
	return vi.BindFlagValues(flags)
}

// BindFlagValue wraps viper's method.
func BindFlagValue(key string, flag viper.FlagValue) error {
	return vi.BindFlagValue(key, flag)
}

// BindEnv wraps viper's method.
func BindEnv(in ...string) error {
	return vi.BindEnv(in...)
}

// Debug wraps viper's method.
func Debug() {
	vi.Debug()
}

// Get wraps viper's method.
func Get(key string) interface{} {
	return vi.Get(key)
}

// GetBool returns the value associated with the key as a boolean.
func GetBool(key string) bool {
	return cast.ToBool(vi.Get(key))
}

// GetDuration returns the value associated with the key as a duration.
func GetDuration(key string) time.Duration {
	return cast.ToDuration(vi.Get(key))
}

// GetFloat64 returns the value associated with the key as a float64.
func GetFloat64(key string) float64 {
	return cast.ToFloat64(vi.Get(key))
}

// GetInt returns the value associated with the key as an integer.
func GetInt(key string) int {
	return cast.ToInt(vi.Get(key))
}

// GetInt32 returns the value associated with the key as an integer.
func GetInt32(key string) int32 {
	return cast.ToInt32(vi.Get(key))
}

// GetInt64 returns the value associated with the key as an integer.
func GetInt64(key string) int64 {
	return cast.ToInt64(vi.Get(key))
}

// GetIntSlice returns the value associated with the key as a slice of int values.
func GetIntSlice(key string) []int {
	return cast.ToIntSlice(vi.Get(key))
}

// GetSizeInBytes wraps viper's method.
func GetSizeInBytes(key string) uint {
	return vi.GetSizeInBytes(key)
}

// GetString returns the value associated with the key as a string.
func GetString(key string) string {
	return cast.ToString(vi.Get(key))
}

// GetStringMap returns the value associated with the key as a map of interfaces.
func GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(vi.Get(key))
}

// GetStringMapString returns the value associated with the key as a map of strings.
func GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(vi.Get(key))
}

// GetStringMapStringSlice returns the value associated with the key as a map to a slice of strings.
func GetStringMapStringSlice(key string) map[string][]string {
	return cast.ToStringMapStringSlice(vi.Get(key))
}

// GetStringSlice returns the value associated with the key as a slice of strings.
func GetStringSlice(key string) []string {
	return cast.ToStringSlice(vi.Get(key))
}

// GetTime returns the value associated with the key as time.
func GetTime(key string) time.Time {
	return cast.ToTime(vi.Get(key))
}

// GetUint returns the value associated with the key as an unsigned integer.
func GetUint(key string) uint {
	return cast.ToUint(vi.Get(key))
}

// GetUint32 returns the value associated with the key as an unsigned integer.
func GetUint32(key string) uint32 {
	return cast.ToUint32(vi.Get(key))
}

// GetUint64 returns the value associated with the key as an unsigned integer.
func GetUint64(key string) uint64 {
	return cast.ToUint64(vi.Get(key))
}

// GetViper returns the global viper instance.
func GetViper() *Viper {
	return vi
}

// InConfig wraps viper's method.
func InConfig(key string) bool {
	return vi.InConfig(key)
}

// IsSet wraps viper's method.
func IsSet(key string) bool {
	return vi.IsSet(key)
}

// MergeConfig wraps viper's method.
func MergeConfig(in io.Reader) error {
	return vi.MergeConfig(in)
}

// MergeConfigMap wraps viper's method.
func MergeConfigMap(cfg map[string]interface{}) error {
	return vi.MergeConfigMap(cfg)
}

// MergeInConfig wraps viper's method.
func MergeInConfig() error {
	return vi.MergeInConfig()
}

// ReadConfig wraps viper's method.
func ReadConfig(in io.Reader) error {
	return vi.ReadConfig(in)
}

// ReadConfigFile wraps viper's method.
func ReadConfigFile(path string) error {
	return vi.ReadConfigFile(path)
}

// ReadRemoteConfig wraps viper's method.
func ReadRemoteConfig() error {
	return vi.ReadRemoteConfig()
}

// RegisterAlias wraps viper's method.
func RegisterAlias(alias string, key string) {
	vi.RegisterAlias(alias, key)
}

// SafeWriteConfig wraps viper's method.
func SafeWriteConfig() error {
	return vi.SafeWriteConfig()
}

// SafeWriteConfigAs wraps viper's method.
func SafeWriteConfigAs(filename string) error {
	return vi.SafeWriteConfigAs(filename)
}

// Set wraps viper's method.
func Set(key string, val interface{}) {
	vi.Set(key, val)
}

// SetDefault wraps viper's method.
func SetDefault(key string, val interface{}) {
	vi.SetDefault(key, val)
}

// Sub wraps viper's method.
func Sub(key string) *Viper {
	return vi.Sub(key)
}

// Unmarshal wraps viper's method.
func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return vi.Unmarshal(rawVal, opts...)
}

// UnmarshalExact wraps viper's method.
func UnmarshalExact(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return vi.UnmarshalExact(rawVal, opts...)
}

// UnmarshalKey wraps viper's method.
func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return vi.UnmarshalKey(key, rawVal, opts...)
}

// WatchConfig wraps viper's method.
func WatchConfig() {
	vi.WatchConfig()
}

// WatchRemoteConfig wraps viper's method.
func WatchRemoteConfig() error {
	return vi.WatchRemoteConfig()
}

// WatchRemoteConfigOnChannel wraps viper's method.
func WatchRemoteConfigOnChannel() error {
	return vi.WatchRemoteConfigOnChannel()
}

// WriteConfig wraps viper's method.
func WriteConfig() error {
	return vi.WriteConfig()
}

// WriteConfigFile wraps viper's method.
func WriteConfigFile(filename string) error {
	return vi.WriteConfigFile(filename)
}
