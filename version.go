// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package gl

import (
	"github.com/maloquacious/semver"
)

func Version() semver.Version {
	return semver.Version{
		Major:      0,
		Minor:      1,
		Patch:      0,
		PreRelease: "beta",
		Build:      semver.Commit(),
	}
}
