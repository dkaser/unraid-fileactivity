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
<table id='diskTable' class="stripe compact">
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

