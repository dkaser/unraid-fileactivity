<?php

namespace EDACerton\FileActivity;

use EDACerton\PluginUtils\Translator;

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

$tr    = $tr       ?? new Translator(PLUGIN_ROOT);
$utils = $utils ?? new Utils(PLUGIN_NAME);

$fileactivity_cfg = new Config();

?>

<script>
$(function() {
	showStatus('fileactivity-watcher');
});
</script>
<script src="/plugins/file.activity/assets/re2js.js"></script>

<table class="tablesorter shift ups">

<thead><tr><th><?= $tr->tr("plugin_settings"); ?></th></tr></thead>
</table>

<h3><?= $tr->tr("settings.monitoring"); ?></h3>

<p><?= $tr->tr("settings.description"); ?></p>

<p><?= $tr->tr("settings.re2"); ?>: <a href="https://github.com/google/re2/wiki/syntax" target="_blank"><?= $tr->tr("settings.reference"); ?></a></p>

<p><?= $tr->tr("settings.note"); ?></p>

<div>
	<form markdown="1" id="file_activity" method="POST" action="/plugins/file.activity/data.php/config">

    <dl>
        <dt><?= $tr->tr("enable_monitoring"); ?></dt>
        <dd>
            <select name="enable" size="1" class="narrow">
		        <?= Utils::make_option( ! $fileactivity_cfg->isEnabled(), "no", $tr->tr("no"));?>
		        <?= Utils::make_option($fileactivity_cfg->isEnabled(), "yes", $tr->tr("yes"));?>
	        </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.enable_monitoring"); ?>
    </blockquote>

    <dl>
        <dt><?= $tr->tr("enable_ssd"); ?></dt>
        <dd>
            <select name="ssd" size="1" class="narrow">
		        <?= Utils::make_option( ! $fileactivity_cfg->isSSDEnabled(), "no", $tr->tr("no"));?>
		        <?= Utils::make_option($fileactivity_cfg->isSSDEnabled(), "yes", $tr->tr("yes"));?>
	        </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.enable_ssd"); ?>
    </blockquote>
	
    <dl>
        <dt><?= $tr->tr("enable_unassigned"); ?></dt>
        <dd>
            <select name="unassigned_devices" size="1" class="narrow">
                <?= Utils::make_option($fileactivity_cfg->isUnassignedDevicesEnabled(), "yes", $tr->tr("yes"));?>
                <?= Utils::make_option( ! $fileactivity_cfg->isUnassignedDevicesEnabled(), "no", $tr->tr("no"));?>
            </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.enable_unassigned"); ?>
    </blockquote>

    <dl>
        <dt><?= $tr->tr("enable_cache"); ?></dt>
        <dd>
            <select name="cache" size="1" class="narrow">
                <?= Utils::make_option( ! $fileactivity_cfg->isCacheEnabled(), "no", $tr->tr("no"));?>
                <?= Utils::make_option($fileactivity_cfg->isCacheEnabled(), "yes", $tr->tr("yes"));?>
            </select>
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.enable_cache"); ?>
    </blockquote>

    <dl>
        <dt><?= $tr->tr("display_events"); ?></dt>
        <dd>
            <input type="number" name="display_events" min="1" step="1" class="narrow" value="<?= htmlspecialchars(strval($fileactivity_cfg->getDisplayEvents()));?>" placeholder="250">
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.display_events"); ?>
    </blockquote>

    <dl>
        <dt><?= $tr->tr("rollover"); ?></dt>
        <dd>
            <input type="number" name="max_records" min="1" step="1" class="narrow" value="<?= htmlspecialchars(strval($fileactivity_cfg->getMaxRecords()));?>" placeholder="20000"> <?= $tr->tr("current_usage"); ?>: <?= Utils::size_readable(Utils::getActivitySize()); ?>
        </dd>
    </dl>
    <blockquote class="inline_help">
        <?= $tr->tr("settings.help.rollover"); ?>
    </blockquote>

    <dl>
        <dt><?= $tr->tr("exclusions"); ?></dt>
        <dd><span><button type="button" id="add-exclusion" class="exclusion-button" onclick="addExclusion()"><?= $tr->tr("add"); ?></button></span></dd>
    </dl>
    <div id="exclusions-container">
        <?php foreach ($fileactivity_cfg->getExclusions() as $exclusion) { ?>
            <dl><dt>&nbsp;</dt>
            <dd>
                <input type="text" name="exclusions[]" oninput="validateRE2(this)" value="<?= htmlspecialchars($exclusion); ?>">
                <i class="fa fa-exclamation-circle regex-error"></i>
                <span><button type="button" class="exclusion-button" onclick="removeExclusion(this)"><?= $tr->tr("remove"); ?></button></span>
            </dd></dl>
        <?php } ?>
    </div>

    <script>
        function addExclusion() {
            var container = document.getElementById('exclusions-container');
            var newExclusion = document.createElement('dl');
            newExclusion.innerHTML = `<dt>&nbsp;</dt>
                                       <dd>
                                           <input type="text" oninput="validateRE2(this)" name="exclusions[]" value="">
                                           <i class="fa fa-exclamation-circle regex-error"></i>
                                           <span><button type="button" class="exclusion-button" onclick="removeExclusion(this)"><?= $tr->tr("remove"); ?></button></span>
                                       </dd>`;
            container.appendChild(newExclusion);
            formChanged();
        }

        function removeExclusion(button) {
            button.parentElement.parentElement.remove();
            formChanged();
        }

        function formChanged() {
            formHasUnsavedChanges = true;
            var form = $('#file_activity');
            form.find('input[value="Apply"],input[value="Apply"],input[name="cmdEditShare"],input[name="cmdUserEdit"]').not('input.lock').prop('disabled',false);
            form.find('input[value="Done"],input[value="Done"]').not('input.lock').val("Reset").prop('onclick',null).off('click').click(function(){formHasUnsavedChanges=false;refresh(form.offset().top);});
        }

        function validateRE2(input) {
            var errorIcon = input.nextElementSibling;

            // Validate using RE2JS
            try {
                var re2 = RE2JS.RE2JS.compile(input.value.trim());
                // If the pattern is valid, remove any error class
                input.classList.remove('regex-error');
                input.setCustomValidity(""); // Clear any custom validity message
                errorIcon.style.display = 'none'; // Hide the error icon
            } catch (e) {
                // If the pattern is invalid, add an error class
                input.classList.add('regex-error');
                input.setCustomValidity("Invalid regex pattern: " + e.message);
                errorIcon.style.display = 'inline'; // Show the error icon
            }
        }
    </script>

    <dl>
        <dt><strong><?= $tr->tr("save"); ?></strong></dt>
        <dd>
            <span><input type="submit" id="apply" value="<?= $tr->tr("apply"); ?>"><input type="button" value="<?= $tr->tr("done"); ?>" onclick="done()"></span>
        </dd>
    </dl>
</form>

<form method="POST" action="/plugins/file.activity/data.php/default">
<dl>
    <dt><strong><?= $tr->tr("settings.apply_defaults"); ?></strong></dt>
    <dd>
        <span><input type="submit" value="<?= $tr->tr("default"); ?>"></span>
    </dd>
</dl>
</form>

<form method="POST" action="/update.php" target="progressFrame">
<input type="hidden" name="#command" value="/plugins/file.activity/scripts/rc.file.activity">
<input type="hidden" name="#arg[1]" value="clear">
<dl>
    <dt><strong><?= $tr->tr("settings.clear_data_description"); ?></strong></dt>
    <dd>
        <span><input type="submit" value="<?= $tr->tr('settings.clear_data'); ?>"></span>
    </dd>
</dl>
</form>
</div>

<?= $utils->getLicenseBlock(); ?>