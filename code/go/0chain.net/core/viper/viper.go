package viper

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type (
	// Viper wraps spf13 viper configuration registry.
	Viper struct {
		viper *viper.Viper
		mutex sync.RWMutex
	}
)

// New returns constructed viper instance.
func New() *Viper {
	return &Viper{viper: viper.New()}
}

func (v *Viper) Instance() *viper.Viper {
	return v.viper
}

// AddConfigPath wraps viper's method.
func (v *Viper) AddConfigPath(in string) {
	v.viper.AddConfigPath(in)
}

// AddRemoteProvider wraps viper's method.
func (v *Viper) AddRemoteProvider(provider, endpoint, path string) error {
	return v.viper.AddRemoteProvider(provider, endpoint, path)
}

// AddSecureRemoteProvider wraps viper's method.
func (v *Viper) AddSecureRemoteProvider(provider, endpoint, path, secret string) error {
	return v.viper.AddSecureRemoteProvider(provider, endpoint, path, secret)
}

// AllKeys wraps viper's method.
func (v *Viper) AllKeys() []string {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.AllKeys()
}

// AllSettings wraps viper's method.
func (v *Viper) AllSettings() map[string]interface{} {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.AllSettings()
}

// BindPFlags wraps viper's method.
func (v *Viper) BindPFlags(flags *pflag.FlagSet) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.BindPFlags(flags)
}

// BindPFlag wraps viper's method.
func (v *Viper) BindPFlag(key string, flag *pflag.Flag) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.BindPFlag(key, flag)
}

// BindFlagValues wraps viper's method.
func (v *Viper) BindFlagValues(flags viper.FlagValueSet) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.BindFlagValues(flags)
}

// BindFlagValue wraps viper's method.
func (v *Viper) BindFlagValue(key string, flag viper.FlagValue) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.BindFlagValue(key, flag)
}

// BindEnv wraps viper's method.
func (v *Viper) BindEnv(in ...string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.BindEnv(in...)
}

// Debug wraps viper's method.
func (v *Viper) Debug() {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	v.viper.Debug()
}

// Get wraps viper's method.
func (v *Viper) Get(key string) interface{} {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.Get(key)
}

// GetBool returns the value associated with the key as a boolean.
func (v *Viper) GetBool(key string) bool {
	return cast.ToBool(v.Get(key))
}

// GetDuration returns the value associated with the key as a duration.
func (v *Viper) GetDuration(key string) time.Duration {
	return cast.ToDuration(v.Get(key))
}

// GetFloat64 returns the value associated with the key as a float64.
func (v *Viper) GetFloat64(key string) float64 {
	return cast.ToFloat64(v.Get(key))
}

// GetInt returns the value associated with the key as an integer.
func (v *Viper) GetInt(key string) int {
	return cast.ToInt(v.Get(key))
}

// GetInt32 returns the value associated with the key as an integer.
func (v *Viper) GetInt32(key string) int32 {
	return cast.ToInt32(v.Get(key))
}

// GetInt64 returns the value associated with the key as an integer.
func (v *Viper) GetInt64(key string) int64 {
	return cast.ToInt64(v.Get(key))
}

// GetIntSlice returns the value associated with the key as a slice of int values.
func (v *Viper) GetIntSlice(key string) []int {
	return cast.ToIntSlice(v.Get(key))
}

// GetSizeInBytes wraps viper's method.
func (v *Viper) GetSizeInBytes(key string) uint {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.GetSizeInBytes(key)
}

// GetString returns the value associated with the key as a string.
func (v *Viper) GetString(key string) string {
	return cast.ToString(v.Get(key))
}

// GetStringMap returns the value associated with the key as a map of interfaces.
func (v *Viper) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(v.Get(key))
}

// GetStringMapString returns the value associated with the key as a map of strings.
func (v *Viper) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(v.Get(key))
}

// GetStringMapString returns the value associated with the key as a map of strings.
func (v *Viper) GetStringMapInt(key string) map[string]int {
	return cast.ToStringMapInt(v.Get(key))
}

// GetStringMapStringSlice returns the value associated with the key as a map to a slice of strings.
func (v *Viper) GetStringMapStringSlice(key string) map[string][]string {
	return cast.ToStringMapStringSlice(v.Get(key))
}

// GetStringSlice returns the value associated with the key as a slice of strings.
func (v *Viper) GetStringSlice(key string) []string {
	return cast.ToStringSlice(v.Get(key))
}

// GetTime returns the value associated with the key as time.
func (v *Viper) GetTime(key string) time.Time {
	return cast.ToTime(v.Get(key))
}

// GetUint returns the value associated with the key as an unsigned integer.
func (v *Viper) GetUint(key string) uint {
	return cast.ToUint(v.Get(key))
}

// GetUint32 returns the value associated with the key as an unsigned integer.
func (v *Viper) GetUint32(key string) uint32 {
	return cast.ToUint32(v.Get(key))
}

// GetUint64 returns the value associated with the key as an unsigned integer.
func (v *Viper) GetUint64(key string) uint64 {
	return cast.ToUint64(v.Get(key))
}

// InConfig wraps viper's method.
func (v *Viper) InConfig(key string) bool {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.InConfig(key)
}

// IsSet wraps viper's method.
func (v *Viper) IsSet(key string) bool {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.IsSet(key)
}

// MergeConfig wraps viper's method.
func (v *Viper) MergeConfig(in io.Reader) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.MergeConfig(in)
}

// MergeConfigMap wraps viper's method.
func (v *Viper) MergeConfigMap(cfg map[string]interface{}) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.MergeConfigMap(cfg)
}

// MergeInConfig wraps viper's method.
func (v *Viper) MergeInConfig() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.MergeInConfig()
}

// ReadConfig wraps viper's method.
func (v *Viper) ReadConfig(in io.Reader) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.ReadConfig(in)
}

// ReadConfigFile wraps viper's method.
func (v *Viper) ReadConfigFile(path string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	blob, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	v.viper.SetConfigType(filepath.Ext(path)[1:])

	return v.viper.ReadConfig(bytes.NewReader(blob))
}

// ReadRemoteConfig wraps viper's method.
func (v *Viper) ReadRemoteConfig() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.ReadRemoteConfig()
}

// RegisterAlias wraps viper's method.
func (v *Viper) RegisterAlias(alias string, key string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.viper.RegisterAlias(alias, key)
}

// SafeWriteConfig wraps viper's method.
func (v *Viper) SafeWriteConfig() error {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.SafeWriteConfig()
}

// SafeWriteConfigAs wraps viper's method.
func (v *Viper) SafeWriteConfigAs(filename string) error {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.SafeWriteConfigAs(filename)
}

// Set wraps viper's method.
func (v *Viper) Set(key string, val interface{}) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.viper.Set(key, val)
}

// SetDefault wraps viper's method.
func (v *Viper) SetDefault(key string, val interface{}) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.viper.SetDefault(key, val)
}

// Sub wraps viper's method.
func (v *Viper) Sub(key string) *Viper {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	sub := v.viper.Sub(key)
	if sub != nil {
		return &Viper{viper: sub}
	}

	return nil
}

// Unmarshal wraps viper's method.
func (v *Viper) Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.Unmarshal(rawVal, opts...)
}

// UnmarshalExact wraps viper's method.
func (v *Viper) UnmarshalExact(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.UnmarshalExact(rawVal, opts...)
}

// UnmarshalKey wraps viper's method.
func (v *Viper) UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.viper.UnmarshalKey(key, rawVal, opts...)
}

// WatchConfig wraps viper's method.
func (v *Viper) WatchConfig() {
	v.viper.WatchConfig()
}

// WatchRemoteConfig wraps viper's method.
func (v *Viper) WatchRemoteConfig() error {
	return v.viper.WatchRemoteConfig()
}

// WatchRemoteConfigOnChannel wraps viper's method.
func (v *Viper) WatchRemoteConfigOnChannel() error {
	return v.viper.WatchRemoteConfigOnChannel()
}

// WriteConfig wraps viper's method.
func (v *Viper) WriteConfig() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.WriteConfig()
}

// WriteConfigFile wraps viper's method.
func (v *Viper) WriteConfigFile(filename string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.viper.WriteConfigAs(filename)
}

// SetConfigType sets the type of the configuration returned by the
// remote source, e.g. "json".
func (v *Viper) SetConfigType(in string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.viper.SetConfigType(in)
}
