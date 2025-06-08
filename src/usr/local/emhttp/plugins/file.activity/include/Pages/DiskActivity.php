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

<link href="/plugins/file.activity/assets/select2.min.css" rel="stylesheet">
<script src="/plugins/file.activity/assets/select2.min.js"></script>
<link href="/plugins/file.activity/assets/datatables.min.css" rel="stylesheet">
<script src="/plugins/file.activity/assets/datatables.min.js"></script>

<table id='diskTable'>
    <thead>
        <tr>
            <th><strong><?= $tr->tr("date"); ?></strong></th>
            <th><strong><?= $tr->tr("action"); ?></strong></th>
            <th><strong><?= $tr->tr("file_path"); ?></strong></th>
            <th>Test</th>
        </tr>
        <tr>
            <th></th>
            <th></th>
            <th></th>
            <th></th>
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
$(document).ready( function () {
    $('#diskTable').DataTable({
        ajax: {
            url: '/plugins/file.activity/data.php/disk',
            dataSrc: ''
        },
        columns: [
            { name: "Timestamp", data: 'timestamp' },
            { name: "action", data: 'action',
                orderable: false
             },
            { name: "path", data: 'filePath' },
            { name: "disk", data: 'disk', visible: false }
        ],
        paging: false,
        ordering: false,
        rowGroup: {
            dataSrc: 'disk'
        },
        layout: {
            topStart: {
                buttons: [
                    {
                        text: '<?= $tr->tr("refresh"); ?>',
                        action: function ( e, dt, node, config ) {
                            dt.ajax.reload();
                        }
                    }
                ]
            }
        },
        initComplete: function () {
            this.api()
                .columns([1])
                .every(function () {
                    var title = this.header();
                    //replace spaces with dashes
                    title = $(title).html().replace(/[\W]/g, '-');
                    title = title + '-disk';
                    var column = this;
                    var select = $('<select id="' + title + '" class="select2" ></select>')
                       .appendTo($("#diskTable thead tr:eq(1) th").eq(column.index()).empty())
                        .on( 'change', function () {
                        //Get the "text" property from each selected data 
                        //regex escape the value and store in array
                        var data = $.map( $(this).select2('data'), function( value, key ) {
                            return value.text ? '^' + $.fn.dataTable.util.escapeRegex(value.text) + '$' : null;
                                    });
                        
                        //if no data selected use ""
                        if (data.length === 0) {
                            data = [""];
                        }
                        
                        //join array into string with regex or (|)
                        var val = data.join('|');
                        
                        //search for the option(s) selected
                        column
                                .search( val ? val : '', true, false )
                                .draw();
                        } );
    
                    column.data().unique().sort().each( function ( d, j ) {
                        select.append( '<option value="'+d+'">'+d+'</option>' );
                    } );
                
                //use column title as selector and placeholder
                $('#' + title).select2({
                    multiple: true,
                    closeOnSelect: true,
                    width: '166px',
                });
                
                //initially clear select otherwise first option is selected
                $('#' + title).val(null).trigger('change');
                });

            this.api()
                .columns([2])
                .every(function () {
                    let column = this;
                    let title = column.footer().textContent;
    
                    // Create input element
                    let input = document.createElement('input');
                    input.placeholder = title;
                    $("#diskTable thead tr:eq(1) th").eq(column.index()).empty().append(input);
    
                    // Event listener for user input
                    input.addEventListener('keyup', () => {
                        if (column.search() !== this.value) {
                            column.search(input.value).draw();
                        }
                    });
                });
        }
    });
} );
</script>

