<?php

namespace EDACerton\FileActivity;

use EDACerton\PluginUtils\Translator;
use EDACerton\PluginUtils\Utils;

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

if ( ! defined(__NAMESPACE__ . '\PLUGIN_ROOT') || ! defined(__NAMESPACE__ . '\PLUGIN_NAME')) {
    throw new \RuntimeException("Common file not loaded.");
}

$tr = $tr ?? new Translator(PLUGIN_ROOT);

// Parse the plugin config file.
$file_activity_cfg = Utils::parse_plugin_cfg('file.activity');

?>

<script>
$(function() {
	showStatus('fileactivity-watcher');
});
</script>

<table class="tablesorter shift ups">

<thead><tr><th><?= $tr->tr("plugin_settings"); ?></th></tr></thead>
</table>

<?if ($mdStarted ?? false) { ?>
<h3><?= $tr->tr("settings.monitoring"); ?></h3>

<p><?= $tr->tr("settings.description"); ?></p>

<p><?= $tr->tr("settings.note"); ?></p>

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
        <?= $tr->tr("settings.help.enable_monitoring"); ?>
    </blockquote>

    <dl>
        <dt><?= $tr->tr("enable_ssd"); ?></dt>
        <dd>
            <select name="INCLUDE_SSD" size="1">
		        <?= Utils::make_option($file_activity_cfg['INCLUDE_SSD'] == "no", "no", $tr->tr("no"));?>
		        <?= Utils::make_option($file_activity_cfg['INCLUDE_SSD'] == "yes", "yes", $tr->tr("yes"));?>
	        </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.enable_ssd"); ?>
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
        <?= $tr->tr("settings.help.enable_unassigned"); ?>
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
        <?= $tr->tr("settings.help.enable_cache"); ?>
    </blockquote>

    <dl>
        <dt><?= $tr->tr("display_events"); ?></dt>
        <dd>
            <input type="text" name="DISPLAY_EVENTS" class="narrow" maxlength="4" value="<?= htmlspecialchars($file_activity_cfg['DISPLAY_EVENTS']);?>" placeholder="25">
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.display_events"); ?>
    </blockquote>

    <dl>
        <dt>
            <input type="submit" name="#default" value="<?= $tr->tr("default"); ?>" title="<?= $tr->tr("settings.apply_defaults"); ?>">
        </dt>
        <dd>
            <input type="submit" name="#apply" value="<?= $tr->tr("apply"); ?>"><input type="button" value="<?= $tr->tr("done"); ?>" onclick="done()">
        </dd>
</form>
</div>
<?} else { ?>
	<p><?= $tr->tr("array_stopped"); ?></p>
<?}?>
