package main

/*
	fileactivity-watcher
	Copyright (C) 2025-2026 Derek Kaser

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/config"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/disks"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/docker"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/filter"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/monitor"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/version"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/writer"
)

type App struct {
	appConfig    config.ActivityConfig
	watchFolders map[string]int
}

func setup() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	debug := flag.Bool("debug", false, "sets log level to debug")
	license := flag.Bool("license", false, "prints the license information")

	flag.Parse()

	if *license {
		printLicense()
		os.Exit(0)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Log version information
	version.OutputToLog()
}

func main() {
	setup()
	log.Info().Msg("Starting file activity watcher...")

	app := NewApp()

	app.watchFolders = app.GetWatchFolders()

	app.startEventListener()

	log.Info().Msg("Watcher ready")
	<-make(chan struct{})
}

func NewApp() *App {
	appConfig := config.LoadConfig()
	if !appConfig.Enable {
		log.Info().Msg("File activity watcher is disabled, exiting")
		os.Exit(0)
	}

	return &App{
		appConfig: appConfig,
	}
}

func (a *App) GetWatchFolders() map[string]int {
	disks := disks.New(a.appConfig)

	return disks.GetWatchFolders()
}

func (a *App) startEventListener() {
	go func() {
		log.Info().Msg("Starting event listener...")

		dockerClient := docker.New()
		filter := filter.New(a.appConfig)

		activityFile, err := writer.New(a.appConfig.ActivityPath, a.appConfig.MaxRecords)
		if err != nil {
			log.Fatal().Err(err).Msg("Error creating activity file writer")
		}
		defer activityFile.Close()

		monitor := monitor.New(a.watchFolders)

		for {
			event, err := monitor.GetEvent()
			if err != nil {
				log.Error().Err(err).Msg("Error getting event")

				continue
			}

			if filter.IsExcluded(event) {
				continue
			}

			eventDetails := monitor.GetEventDetails(event)

			containerName := ""
			if eventDetails.ContainerID != "" {
				containerName = dockerClient.GetContainerNameByID(eventDetails.ContainerID)
			}

			err = activityFile.Write(
				[]string{
					time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
					event.Op,
					event.File,
					strconv.Itoa(event.PID),
					eventDetails.ProcessPath,
					containerName,
				},
			)
			if err != nil {
				log.Error().Err(err).Msg("Error writing activity record")
			}
		}
	}()
}

func printLicense() {
	licenseText := `
	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	`

	fmt.Println(licenseText)
}
