package enums

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobalSettings(t *testing.T) {
	initGlobalSettingNames()
	initGlobalSettings()

	require.Len(t, GlobalSettingName, int(NumOfGlobalSettings))
	require.Len(t, GlobalSettingInfo, int(NumOfGlobalSettings))

	for key := range GlobalSettingInfo {
		found := false
		for _, name := range GlobalSettingName {
			if key == name {
				found = true
				break
			}
		}
		require.True(t, found)
	}

	for _, name := range GlobalSettingName {
		_, ok := GlobalSettingInfo[name]
		require.True(t, ok)
	}
}
