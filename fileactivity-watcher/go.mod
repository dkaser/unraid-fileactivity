module github.com/dkaser/unraid-fileactivity/fileactivity-watcher

go 1.24

require (
	github.com/containernetworking/plugins v1.7.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/sirupsen/logrus v1.9.3
	gopkg.in/ini.v1 v1.67.0
)

require golang.org/x/sys v0.33.0 // indirect

replace github.com/fsnotify/fsnotify => github.com/dkaser/fsnotify v1.9.3
