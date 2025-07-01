<?php

namespace EDACerton\FileActivity;

use EDACerton\PluginUtils\Utils;

/*
    Copyright (C) 2025  Derek Kaser

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

class Activity
{
    private int $display_events;

    public function __construct()
    {
        // Constructor can be used for initialization if needed.
        $config               = new Config();
        $this->display_events = $config->getDisplayEvents();
    }
    /**
     * @return array<string, list<array<string, string>>>
     */
    public function getShareActivity(): array
    {
        $result = array();

        // Share activity.
        $shares = $this->getShares();
        natcasesort($shares);
        foreach ($shares as $share) {
            $share_dev = basename($share) . "/";
            $files     = $this->getActivityEntries("/mnt/disk\d+/{$share_dev}", $this->display_events);
            if ( ! empty($files)) {
                $result[basename($share)] = $files;
            }
        }

        return $result;
    }

    /**
     * @return array<string, list<array<string, string>>>
     */
    public function getDiskActivity(): array
    {
        $result = array();

        // Share activity.
        $disks = $this->getDisks();
        foreach ($disks["array"] as $disk) {
            $dev   = basename($disk) . "/";
            $files = $this->getActivityEntries("/mnt/{$disk}/", $this->display_events);
            if ( ! empty($files)) {
                $result[$disk] = $files;
            }
        }

        $files = $this->getActivityEntries("/mnt/disks/", $this->display_events);
        if ( ! empty($files)) {
            $result["Unassigned Devices"] = $files;
        }

        foreach ($disks['pool'] as $pool) {
            $files = $this->getActivityEntries("/mnt/{$pool}/", $this->display_events);
            if ( ! empty($files)) {
                $result[ucfirst($pool)] = $files;
            }
        }

        return $result;
    }

    /**
     * @return array<string>
     */
    public function getShares(): array
    {
        $shares = parse_ini_file("/var/local/emhttp/shares.ini", true) ?: array();
        return array_keys($shares);
    }

    /**
     * @return array<string, list<string>>
     */
    public function getDisks(): array
    {
        $result = [
            "array" => [],
            "pool"  => [],
        ];

        $disks = parse_ini_file("/var/local/emhttp/disks.ini", true) ?: array();

        foreach ($disks as $disk => $info) {
            if (isset($info['type']) && $info['type'] === 'Data') {
                $result['array'][] = $disk;
            } elseif (isset($info['type']) && $info['type'] === 'Cache') {
                if (isset($info['fsType']) && ! empty($info['fsType'])) {
                    $result['pool'][] = $disk;
                }
            }
        }

        return $result;
    }

    /**
     * @return list<array<string, string>>
     */
    private function getActivityEntries(string $disk, int $display_events): array
    {
        $files      = shell_exec("cat /var/log/file.activity/data.log.1 /var/log/file.activity/data.log  2>/dev/null | grep -P " . escapeshellarg($disk) . " | tail -n " . strval($display_events));
        $filesArray = array();

        if ($files) {
            $files = explode("\n", $files);
            foreach ($files as $file) {
                if ( ! empty($file)) {
                    $fileEntry    = new ActivityEntry($file);
                    $filesArray[] = $fileEntry->toArray();
                }
            }
        }

        return $filesArray;
    }

    /**
     * @param array<string, list<array<string, string>>> $activity
     * @return list<array<string, string>>
     */
    public function flattenActivity(array $activity): array
    {
        $result = array();
        foreach ($activity as $disk => $files) {
            foreach ($files as $file) {
                $file['disk'] = $disk;
                $result[]     = $file;
            }
        }
        return $result;
    }
}
