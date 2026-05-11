//go:build embed

package web

import (
	"embed"
	"io/fs"
)

//go:embed embed/public
var embeddedFS embed.FS

// PublicFS is the Almanach Studio filesystem rooted at embed/public/.
// cmd/build-web builds web/dist and copies it here for embedding.
var PublicFS, _ = fs.Sub(embeddedFS, "embed/public")
