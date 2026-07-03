// Package menubar embeds menu bar icon templates.
package menubar

import _ "embed"

//go:embed iconTemplate.png
var IconIdle []byte

//go:embed iconTemplate@2x.png
var IconIdle2x []byte

//go:embed iconAlertTemplate.png
var IconAlert []byte

//go:embed iconAlertTemplate@2x.png
var IconAlert2x []byte

//go:embed iconOfflineTemplate.png
var IconOffline []byte

//go:embed iconOfflineTemplate@2x.png
var IconOffline2x []byte
