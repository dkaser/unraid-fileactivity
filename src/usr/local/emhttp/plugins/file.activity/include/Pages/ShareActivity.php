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
?>

<table class="tablesorter shift ups">
<thead><tr><th><?= $tr->tr("share_activity"); ?></th></tr></thead>
</table>

<br>
<?php
echo ($resize ?? false) ? "<pre class='up' style='display:none'>" : "<pre class='up'>";

$display_events = ((isset($file_activity_cfg['DISPLAY_EVENTS'])) && ($file_activity_cfg['DISPLAY_EVENTS'])) ? $file_activity_cfg['DISPLAY_EVENTS'] : "25";
$filesactive    = "";

// Share activity.
$shares = glob("/mnt/user/*") ?: array();
natcasesort($shares);
foreach ($shares as $share) {
    $share_dev = basename($share) . "/";
    $files     = shell_exec("cat /var/log/file.activity.log 2>/dev/null | grep  -P '/mnt/disk\d+/{$share_dev}' | tail -n " . escapeshellarg($display_events));
    if ($files) {
        $filesactive .= "<strong>** " . $share . " **</strong>\n";
        $filesactive .= $files;
        $filesactive .= "\n";
    }
}

echo $filesactive;
echo "</pre>";
?>

<script>
<?if ($resize):?>
$(function() {
  $('pre.up').css('height',Math.max(window.innerHeight-400,370)).show();
});
<?endif;?>
</script>

<div style="position:relative;float:left;text-align:right;margin-bottom:24px">
	<input type="button" value="<?= $tr->tr("refresh"); ?>" title="<?= $tr->tr("refresh_page"); ?>" onclick="refresh()">
</div>
<div style="position:relative;float:left;text-align:right;margin-bottom:24px">
	<form name="clear log" method="POST" action="/update.php" target="progressFrame">
		<input type="hidden" name="#command" value="/plugins/file.activity/scripts/rc.file.activity">
		<input type="hidden" name="#arg[1]" value="clear">
		<input type="submit" value="<?= $tr->tr("clear"); ?>" title="<?= $tr->tr("clear_log"); ?>">
	</form>
</div>
<div style="position:relative;float:left;text-align:right;margin-bottom:24px">
	<input type="button" value="<?= $tr->tr("done"); ?>" onclick="done()">
</div>
