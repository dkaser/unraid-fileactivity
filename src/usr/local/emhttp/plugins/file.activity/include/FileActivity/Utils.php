<?php

namespace FileActivity;

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

class Utils
{
    public static function make_option(bool $selected, string $value, string $text, string $extra = ""): string
    {
        return "<option value='{$value}'" . ($selected ? " selected" : "") . (strlen($extra) ? " {$extra}" : "") . ">{$text}</option>";
    }

    /**
     * @return array<string, string>
     */
    public static function parse_plugin_cfg(string $plugin): array
    {
        $default = "/usr/local/emhttp/plugins/{$plugin}/default.cfg";
        $user    = "/boot/config/plugins/{$plugin}/{$plugin}.cfg";

        $cfg_default = parse_ini_file($default, false, INI_SCANNER_RAW) ?: array();
        $cfg_user    = parse_ini_file($user, false, INI_SCANNER_RAW) ?: array();
        return array_replace_recursive($cfg_default, $cfg_user);
    }

    /**
     * @return array<string>
     */
    public static function run_command(string $command, bool $alwaysShow = false, bool $show = true): array
    {
        $output = array();
        $retval = null;
        if ($show) {
            self::logmsg("exec: {$command}");
        }
        exec("{$command} 2>&1", $output, $retval);

        if (($retval != 0) || $alwaysShow) {
            self::logmsg("Command returned {$retval}" . PHP_EOL . implode(PHP_EOL, $output));
        }

        return $output;
    }

    public static function logmsg(string $message): void
    {
        $timestamp = date('Y/m/d H:i:s');
        $filename  = basename($_SERVER['PHP_SELF']);
        file_put_contents("/var/log/fileactivity.log", "{$timestamp} {$filename}: {$message}" . PHP_EOL, FILE_APPEND);
    }
}
