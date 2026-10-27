// Package registry hands the plugin and locale values defined in package dayjs to the plugin and
// locale packages without exporting them from dayjs itself.
package registry

// Plugins maps a Day.js plugin name to its *dayjs.Plugin.
var Plugins = map[string]any{}

// Locales maps a Day.js locale name to its *dayjs.Object.
var Locales = map[string]any{}
