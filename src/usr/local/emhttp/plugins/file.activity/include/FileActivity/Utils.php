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
    public static function make_option(bool|string $selected, string $value, string $text, string $extra = ""): string
    {
        if (is_string($selected)) {
            $selected = $selected === $value;
        }

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
        $filename  = basename(is_string($_SERVER['PHP_SELF']) ? $_SERVER['PHP_SELF'] : "");
        file_put_contents("/var/log/fileactivity.log", "{$timestamp} {$filename}: {$message}" . PHP_EOL, FILE_APPEND);
    }

    /**
    * @param array<mixed> $content
    */
    private static function send_usage(string $url, array $content): int
    {
        if (empty($url)) {
            throw new \InvalidArgumentException("URL cannot be empty");
        }

        $body = json_encode($content);

        if ( ! $body) {
            throw new \InvalidArgumentException("Failed to encode JSON");
        }

        $token = self::download_url($url . '?connect');

        $c = curl_init();
        curl_setopt($c, CURLOPT_URL, $url);

        $headers = [
            'Content-Type: application/json',
            'Authorization: Bearer ' . $token
        ];

        curl_setopt($c, CURLOPT_POST, true);
        curl_setopt($c, CURLOPT_POSTFIELDS, $body);
        curl_setopt($c, CURLOPT_HTTPHEADER, $headers);
        curl_setopt($c, CURLOPT_USERAGENT, 'plugin-metrics/1.0.0');

        curl_exec($c);
        if ( ! curl_errno($c)) {
            $info = curl_getinfo($c);
            return $info['http_code'];
        }
        return -1;
    }

    public static function sendUsageData(): void
    {
        $endpoint = "https://plugin-usage.edacerton.win/";

        if ( ! defined(__NAMESPACE__ . "\PLUGIN_NAME")) {
            throw new \RuntimeException("PLUGIN_NAME not defined");
        }

        if ( ! defined(__NAMESPACE__ . "\PLUGIN_ROOT")) {
            throw new \RuntimeException("PLUGIN_ROOT not defined");
        }

        $usage_cfg = parse_ini_file("/boot/config/plugins/" . PLUGIN_NAME . "/usage.cfg", false, INI_SCANNER_RAW) ?: array();
        if (($usage_cfg['usage_allowed'] ?? "yes") != "yes") {
            Utils::logmsg("Usage data not allowed, skipping.");
            return;
        }

        $var     = parse_ini_file('/usr/local/emhttp/state/var.ini');
        $version = parse_ini_file(PLUGIN_ROOT . '/version.ini');

        if ( ! $var || ! $version) {
            Utils::logmsg("Could not retrieve system data, skipping usage data.");
            return;
        }

        $content = array(
            'clientId'       => hash("crc32b", $var['flashGUID']),
            'plugin'         => PLUGIN_NAME,
            'plugin_version' => $version['VERSION'],
            'unraid_version' => $var['version'],
        );

        $attempts = 0;
        $delay    = rand(0, 300);
        do {
            Utils::logmsg("Waiting for {$delay} seconds before sending usage data.");
            sleep($delay);
            $delay += 300;
            $attempts++;

            $result = self::send_usage($endpoint, $content);
            Utils::logmsg("Usage data sent.");
        } while (($result != '200') && ($attempts < 3));

        if ($result != '200') {
            Utils::logmsg("Error occurred while transmitting usage data.");
        }
    }

    public static function download_url(string $url): string
    {
        if (empty($url)) {
            throw new \InvalidArgumentException("URL cannot be empty");
        }

        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_FRESH_CONNECT, true);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_CONNECTTIMEOUT, 15);
        curl_setopt($ch, CURLOPT_TIMEOUT, 45);
        curl_setopt($ch, CURLOPT_ENCODING, "");
        curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
        curl_setopt($ch, CURLOPT_FAILONERROR, true);
        curl_setopt($ch, CURLOPT_USERAGENT, 'plugin-metrics/1.0.0');
        $out = curl_exec($ch) ?: false;
        curl_close($ch);
        return strval($out);
    }

    /**
    * @param array<mixed> $args
    */
    public static function run_task(string $functionName, array $args = array()): void
    {
        try {
            $reflectionMethod = new \ReflectionMethod($functionName);
            $reflectionMethod->invokeArgs(null, $args);
        } catch (\Throwable $e) {
            Utils::logmsg("Caught exception in {$functionName} : " . $e->getMessage());
        }
    }
}
