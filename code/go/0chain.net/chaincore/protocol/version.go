package protocol

import (
	"github.com/blang/semver/v4"
)

// LatestSupportProtoVersion indicates the latest protocol version this build could support
var LatestSupportProtoVersion = semver.MustParse("1.0.0")
