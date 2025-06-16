<?php

namespace EDACerton\FileActivity;

use EDACerton\PluginUtils\Translator;

/*
    Copyright 2015-2016, Lime Technology
    Copyright 2015-2016, Bergware International.
    Copyright 2015-2025, Dan Landon
    Copyright 2025  Derek Kaser

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

?>
<link type="text/css" rel="stylesheet" href="/plugins/file.activity/assets/style.css">

<script src="/plugins/file.activity/assets/translate.js"></script>
<script>
    const translator = new Translator("/plugins/file.activity");
</script>

<link href="/plugins/file.activity/assets/datatables.min.css" rel="stylesheet">
<script src="/plugins/file.activity/assets/datatables.min.js"></script>
<script src="/plugins/file.activity/assets/luxon.min.js"></script>
<script src="/plugins/file.activity/assets/flatpickr.min.js"></script>
<link rel="stylesheet" href="/plugins/file.activity/assets/flatpickr.min.css">

<script src="/plugins/file.activity/assets/fileactivity.js"></script>

<table id='logTable' class="stripe compact">
    <thead>
        <tr>
            <th><strong><?= $tr->tr("date"); ?></strong></th>
            <th><strong><?= $tr->tr("action"); ?></strong></th>
            <th><strong><?= $tr->tr("file_path"); ?></strong></th>
            <th><strong><?= $tr->tr("group"); ?></strong></th>
        </tr>
    </thead>
    <tbody>
    </tbody>
    <tfoot>
        <tr>
            <td>&nbsp;</td>
            <td>&nbsp;</td>
            <td>&nbsp;</td>
            <td>&nbsp;</td>
        </tr>
    </tfoot>
</table>

<script>
$(document).ready( async function () {
    await translator.init();
    $('#logTable').DataTable(getDatatableConfig('/plugins/file.activity/data.php/share'));
    $('#diskTable').DataTable(getDatatableConfig('/plugins/file.activity/data.php/disk'));
} );
</script>

