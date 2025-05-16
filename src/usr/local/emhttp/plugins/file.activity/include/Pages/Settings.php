<?php

namespace FileActivity;

/*
    Copyright (C) 2017-2025, Dan Landon
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

$docroot = $docroot ?? $_SERVER['DOCUMENT_ROOT'] ?: '/usr/local/emhttp';
require_once "{$docroot}/plugins/file.activity/include/common.php";

$tr = $tr ?? new Translator();

// Define our plugin name.
define('FILE_ACTIVITY_PLUGIN', 'file.activity');

// Define the docroot path.
if ( ! defined('DOCROOT')) {
    define('DOCROOT', $_SERVER['DOCUMENT_ROOT'] ?: '/usr/local/emhttp');
}

// Parse the plugin config file.
$file_activity_cfg = Utils::parse_plugin_cfg(FILE_ACTIVITY_PLUGIN);
?>

<script>
$(function() {
	showStatus('pid','file.activity');
});
</script>

<table class="tablesorter shift ups">

<thead><tr><th><?= $tr->tr("plugin_settings"); ?></th></tr></thead>
</table>

<?if ($mdStarted ?? false) { ?>
<h3>File Activity Monitoring</h3>

<p>File open, read, write, and modify activity is monitored and logged on the array using inotify and is displayed by disk or share, UD disks, and cache.
You need to start the File Activity in order to log the disk activity.
File Activity is intended to be running for short periods so you can check disk activity.
A server with a lot of file activity can fill the log space.
The 'appdata', 'docker', 'syslogs', and 'system' directories (case insensitive) are excluded.</p>

<p>Note: File Activity monitoring is stopped if the array is stopped, and will restart when the array is started if it is enabled.</p>

<div>
	<form markdown="1" name="file_activity" method="POST" action="/update.php" target="progressFrame">
	<input type="hidden" name="#file" value="file.activity/file.activity.cfg">
	<input type="hidden" name="#command" value="/plugins/file.activity/scripts/rc.file.activity">
	<input type="hidden" name="#arg[1]" value="update">

    <dl>
        <dt><?= $tr->tr("enable_monitoring"); ?></dt>
        <dd>
            <select name="SERVICE" size="1">
		        <?= Utils::make_option($file_activity_cfg['SERVICE'] == "disable", "disable", $tr->tr("no"));?>
		        <?= Utils::make_option($file_activity_cfg['SERVICE'] == "enable", "enable", $tr->tr("yes"));?>
	        </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        Set to **Yes** to enable File Activity monitoring when the server is started.
    </blockquote>
	
    <dl>
        <dt><?= $tr->tr("enable_unassigned"); ?></dt>
        <dd>
            <select name="INCLUDE_UD" size="1">
                <?= Utils::make_option($file_activity_cfg['INCLUDE_UD'] == "yes", "yes", $tr->tr("yes"));?>
                <?= Utils::make_option($file_activity_cfg['INCLUDE_UD'] == "no", "no", $tr->tr("no"));?>
            </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        Set to **Yes** to enable File Activity monitoring for Unassigned Devices if the Unassigned Devices plugin is installed.
    </blockquote>

    <dl>
        <dt><?= $tr->tr("enable_cache"); ?></dt>
        <dd>
            <select name="INCLUDE_CACHE" size="1">
                <?= Utils::make_option($file_activity_cfg['INCLUDE_CACHE'] == "no", "no", $tr->tr("no"));?>
                <?= Utils::make_option($file_activity_cfg['INCLUDE_CACHE'] == "yes", "yes", $tr->tr("yes"));?>
            </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        Set to **Yes** to enable File Activity monitoring for the Cache and Pool Disks.
    </blockquote>

    <dl>
        <dt><?= $tr->tr("display_events"); ?></dt>
        <dd>
            <input type="text" name="DISPLAY_EVENTS" class="narrow" maxlength="4" value="<?= htmlspecialchars($file_activity_cfg['DISPLAY_EVENTS']);?>" placeholder="25">
        </dd>
    </dl>
    <blockquote class="inline_help">
        This is the number of file events shown on disks and shares from the File Activity log for each share or disk.
    </blockquote>

    <dl>
        <dt>
            <input type="submit" name="#default" value="<?= $tr->tr("default"); ?>" title="Load and apply default values.">
        </dt>
        <dd>
            <?if ( ! is_file("/var/run/file.activity.pid")) { ?>
                <input type="hidden" name="#command" value="/plugins/file.activity/scripts/rc.file.activity">
                <input type="hidden" name="#arg[1]" value="start">
                <input type="submit" value="<?= $tr->tr("start"); ?>" title="<?= $tr->tr("start_monitoring"); ?>">
            <?} else { ?>
                <input type="hidden" name="#command" value="/plugins/file.activity/scripts/rc.file.activity">
                <input type="hidden" name="#arg[1]" value="stop">
                <input type="submit" value="<?= $tr->tr("stop"); ?>" title="<?= $tr->tr("stop_monitoring"); ?>">
            <?}?>
            <input type="submit" name="#apply" value="<?= $tr->tr("apply"); ?>"><input type="button" value="<?= $tr->tr("done"); ?>" onclick="done()">
        </dd>
</form>
</div>
<?} else { ?>
	<p><?= $tr->tr("array_stopped"); ?></p>
<?}?>