<?php

namespace EDACerton\FileActivity;

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

class Config
{
    private bool $enable             = false;
    private bool $unassigned_devices = true;
    private bool $cache              = false;
    private bool $ssd                = false;
    private int $display_events      = 1000;
    /**
     * @var array<string>
     */
    private array $exclusions = ['(?i)appdata', '(?i)docker', '(?i)system', '(?i)syslogs'];
    private int $max_records  = 20000;

    private string $config_path = '/boot/config/plugins/file.activity/config.json';

    public function fromINIFile(): void
    {
        $file_activity_cfg = Utils::parse_plugin_cfg('file.activity');
        if (isset($file_activity_cfg['SERVICE'])) {
            $this->enable = $file_activity_cfg['SERVICE'] === 'enable';
        }
        if (isset($file_activity_cfg['INCLUDE_UD'])) {
            $this->unassigned_devices = filter_var($file_activity_cfg['INCLUDE_UD'], FILTER_VALIDATE_BOOLEAN);
        }
        if (isset($file_activity_cfg['INCLUDE_CACHE'])) {
            $this->cache = filter_var($file_activity_cfg['INCLUDE_CACHE'], FILTER_VALIDATE_BOOLEAN);
        }
        if (isset($file_activity_cfg['INCLUDE_SSD'])) {
            $this->ssd = filter_var($file_activity_cfg['INCLUDE_SSD'], FILTER_VALIDATE_BOOLEAN);
        }

        $this->display_events = isset($file_activity_cfg['DISPLAY_EVENTS']) && is_numeric($file_activity_cfg['DISPLAY_EVENTS']) ? intval($file_activity_cfg['DISPLAY_EVENTS']) : 1000;
    }

    public function __construct(bool $load_config = true)
    {
        $json = $load_config ? file_get_contents($this->config_path) : false;

        if ($json) {
            $data = json_decode($json, true);
            if (is_array($data)) {
                $this->enable             = $data['enable']             ?? $this->enable;
                $this->unassigned_devices = $data['unassigned_devices'] ?? $this->unassigned_devices;
                $this->cache              = $data['cache']              ?? $this->cache;
                $this->ssd                = $data['ssd']                ?? $this->ssd;
                $this->display_events     = isset($data['display_events']) && is_numeric($data['display_events']) ? intval($data['display_events']) : $this->display_events;
                $this->exclusions         = $data['exclusions'] ?? $this->exclusions;
                $this->max_records        = isset($data['max_records']) && is_numeric($data['max_records']) ? intval($data['max_records']) : $this->max_records;
            }
        }
    }

    public function save(): void
    {
        $config = json_encode([
            'enable'             => $this->enable,
            'unassigned_devices' => $this->unassigned_devices,
            'cache'              => $this->cache,
            'ssd'                => $this->ssd,
            'display_events'     => $this->display_events,
            'exclusions'         => $this->exclusions,
            'max_records'        => $this->max_records
        ]) ?: '{}';

        file_put_contents($this->config_path, $config);
    }

    public function isEnabled(): bool
    {
        return $this->enable;
    }
    public function isUnassignedDevicesEnabled(): bool
    {
        return $this->unassigned_devices;
    }
    public function isCacheEnabled(): bool
    {
        return $this->cache;
    }
    public function isSSDEnabled(): bool
    {
        return $this->ssd;
    }
    public function getDisplayEvents(): int
    {
        return $this->display_events;
    }
    /**
     * @return array<string>
     */
    public function getExclusions(): array
    {
        return $this->exclusions;
    }
    public function getMaxRecords(): int
    {
        return $this->max_records;
    }

    public function setEnable(bool $enable): void
    {
        $this->enable = $enable;
    }
    public function setUnassignedDevices(bool $unassigned_devices): void
    {
        $this->unassigned_devices = $unassigned_devices;
    }
    public function setCache(bool $cache): void
    {
        $this->cache = $cache;
    }
    public function setSSD(bool $ssd): void
    {
        $this->ssd = $ssd;
    }
    public function setDisplayEvents(int $display_events): void
    {
        if ($display_events <= 0) {
            throw new \InvalidArgumentException("Display events must be a positive integer.");
        }
        $this->display_events = $display_events;
    }
    /**
     * @param array<string> $exclusions
     */
    public function setExclusions(array $exclusions): void
    {
        $this->exclusions = $exclusions;
    }

    public function setMaxRecords(int $max_records): void
    {
        if ($max_records <= 0) {
            throw new \InvalidArgumentException("Max records must be a positive integer.");
        }
        $this->max_records = $max_records;
    }
}
