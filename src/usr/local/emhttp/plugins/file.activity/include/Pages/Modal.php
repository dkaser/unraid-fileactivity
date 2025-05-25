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

$tr = $tr ?? new Translator();

if ( ! defined(__NAMESPACE__ . '\PLUGIN_NAME')) {
    throw new \RuntimeException("PLUGIN_NAME not defined");
}

$usage_cfg       = parse_ini_file("/boot/config/plugins/" . PLUGIN_NAME . "/usage.cfg", false, INI_SCANNER_RAW) ?: array();
$modal_displayed = $usage_cfg['modal_displayed'] ?? "no";

if ($modal_displayed === "yes") {
    // If the modal has already been displayed, we don't need to show it again.
    return;
}

?>

<script>
    swal({
        title: "<?= $tr->tr("metrics.metrics"); ?>",
        text: "<?= $tr->tr("metrics.modal"); ?>",
        type: "info",
        confirmButtonText: "Agree",
        showCancelButton: true,
        cancelButtonText: "Decline",
        html: true
        },
        function(isConfirmed){
            if (isConfirmed) {
                // User agreed, submit the form to allow usage.
                let form = document.getElementById("acceptMetrics");
                form.submit();
            } else {
                // User declined, submit the form to deny usage.
                let form = document.getElementById("denyMetrics");
                form.submit();
            }
        });
</script>

<form method="POST" name="acceptMetrics" id="acceptMetrics" action="/update.php" target="progressFrame">
<input type="hidden" name="#file" value="/boot/config/plugins/<?= PLUGIN_NAME; ?>/usage.cfg">
<input type="hidden" name="usage_allowed" value="yes">
<input type="hidden" name="modal_displayed" value="yes">
</form>

<form method="POST" name="denyMetrics" id="denyMetrics" action="/update.php" target="progressFrame">
<input type="hidden" name="#file" value="/boot/config/plugins/<?= PLUGIN_NAME; ?>/usage.cfg">
<input type="hidden" name="usage_allowed" value="no">
<input type="hidden" name="modal_displayed" value="yes">
</form>