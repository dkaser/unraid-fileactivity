const minDate = {};
const maxDate = {};

DataTable.ext.search.push(function (settings, data, dataIndex) {
    if (minDate[settings.sTableId] === undefined || maxDate[settings.sTableId] === undefined) {
        return true;
    }

    const compareFormat = 'yyyy-MM-dd HH:mm';

    minVal = minDate[settings.sTableId].val();
    maxVal = maxDate[settings.sTableId].val();
    dateVal = new Date(data[0]);

    if (minVal === null && maxVal === null) {
        return true;
    }

    let min = (minVal === null) ? luxon.DateTime.fromMillis(0) : luxon.DateTime.fromJSDate(minVal);
    let max = (maxVal === null) ? luxon.DateTime.now().plus({ hours: 1}) : luxon.DateTime.fromJSDate(maxVal);
    let date = luxon.DateTime.fromJSDate(dateVal);

    min = min.minus({ minutes: min.offset });
    max = max.minus({ minutes: max.offset });

    min = min.toJSDate();
    max = max.toJSDate();
    date = date.toJSDate();
    if (
        (min <= date && date <= max)
    ) {
        return true;
    }
    return false;
});

DataTable.feature.register('dateRange', function (settings, opts) {
    console.log(settings);
    let toolbar = document.createElement('div');
    toolbar.appendChild(document.createTextNode('From: '));

    minInput = document.createElement('input');
    minInput.id = 'min-' + settings.sTableId;
    minInput.name = 'min-' + settings.sTableId;
    minInput.type = 'text';
    toolbar.appendChild(minInput);

    toolbar.appendChild(document.createTextNode(' To: '));
    maxInput = document.createElement('input');
    maxInput.id = 'max-' + settings.sTableId;
    maxInput.name = 'max-' + settings.sTableId;
    maxInput.type = 'text';
    toolbar.appendChild(maxInput);

    const calcTime = luxon.DateTime.now();

    const minTime = calcTime.minus({ minutes: 30 }).toJSDate();

    const dateSettings = {
        format: 'D HH:mm',
        buttons: {
            clear: true
        }
    }

    minDate[settings.sTableId] = new DateTime(minInput, dateSettings);
    maxDate[settings.sTableId] = new DateTime(maxInput, dateSettings);

    minDate[settings.sTableId].val(minTime);

    minInput.addEventListener('change', () => settings.api.draw());
    maxInput.addEventListener('change', () => settings.api.draw());

    return toolbar;
});

function getDatatableConfig(url, refreshText, tableName) {
    return {
        tableName: tableName,
        ajax: {
            url: url,
            dataSrc: ''
        },
        columns: [
            { name: "Timestamp", data: 'timestamp' },
            { name: "action", data: 'action' },
            { name: "path", data: 'filePath' },
            { name: "disk", data: 'disk', visible: true, orderable: false }
        ],
        columnControl: {
            target: 0,
            content: [{
                extend: 'dropdown',
                content: ['searchClear', 'search'],
                icon: 'search'
            }]
        },
        columnDefs: [
            {
                targets: 0,
                render: DataTable.render.datetime(),
                className: 'dt-head-left',
                columnControl: {
                    target: 0,
                    content: []
                }
            },
            {
                targets: [1,3],
                columnControl: {
                    target: 0,
                    content: [{
                        extend: 'dropdown',
                        content: ['searchClear', 'searchList'],
                        icon: 'search'
                    }]
                }
            }
        ],
        paging: true,
        pageLength: 50,
        ordering: true,
        rowGroup: {
            dataSrc: 'disk'
        },
        orderFixed: [3, 'asc'],
        layout: {
            topStart: {
                buttons: [
                    {
                        text: refreshText,
                        action: function ( e, dt, node, config ) {
                            dt.ajax.reload();
                        }
                    },
                    {
                        text: "Clear Filters",
                        action: function ( e, dt, node, config ) {
                            minDate[dt.settings()[0].sTableId].val(null);
                            maxDate[dt.settings()[0].sTableId].val(null);
                            dt.search('');
                            dt.columns().ccSearchClear();
                            dt.draw();
                        }
                    }
                ],
                pageLength: {
                    menu: [25, 50, 100, 200, -1]
                }
            },
            topEnd: {
                dateRange: {}
            }
        },
    };
}
