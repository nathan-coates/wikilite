//go:build plugins

package plugin

import _ "embed"

//go:embed jspkgs.js
var jsLibraries string
