module github.com/dkaser/unraid-fileactivity/fileactivity-watcher

go 1.24

require (
	github.com/containernetworking/plugins v0.8.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/rs/zerolog v1.34.0
	gopkg.in/ini.v1 v1.67.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/fsnotify/fsnotify => github.com/dkaser/fsnotify v1.9.4
