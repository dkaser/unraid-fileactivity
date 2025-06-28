module github.com/dkaser/unraid-fileactivity/fileactivity-watcher

go 1.24

require (
	github.com/containernetworking/plugins v0.9.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/sirupsen/logrus v1.9.3
	gopkg.in/ini.v1 v1.67.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)

replace github.com/fsnotify/fsnotify => github.com/dkaser/fsnotify v1.9.4
