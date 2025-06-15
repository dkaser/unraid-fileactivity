const minDate = {};
const maxDate = {};

DataTable.ext.search.push(function (settings, data, dataIndex) {
    if (minDate[settings.sTableId] === undefined || maxDate[settings.sTableId] === undefined) {
        return true;
    }

    const minVal = minDate[settings.sTableId].selectedDates;
    const maxVal = maxDate[settings.sTableId].selectedDates;
    const dateVal = new Date(data[0]);

    const minValEmpty = !Array.isArray(minVal) || !minVal.length;
    const maxValEmpty = !Array.isArray(maxVal) || !maxVal.length;

    if (minValEmpty && maxValEmpty) {
        return true;
    }

    let min = (minValEmpty) ? luxon.DateTime.fromMillis(0).toJSDate() : minVal[0];
    let max = (maxValEmpty) ? luxon.DateTime.now().plus({ hours: 1}).toJSDate() : maxVal[0];

    if (
        (min <= dateVal && dateVal <= max)
    ) {
        return true;
    }
    return false;
});

DataTable.feature.register('dateRange', function (settings, opts) {
    console.log(settings);
    let toolbar = document.createElement('div');
    toolbar.appendChild(document.createTextNode('From: '));

    const minInput = document.createElement('input');
    minInput.id = 'min-' + settings.sTableId;
    minInput.name = 'min-' + settings.sTableId;
    minInput.type = 'text';
    toolbar.appendChild(minInput);

    toolbar.appendChild(document.createTextNode(' To: '));
    const maxInput = document.createElement('input');
    maxInput.id = 'max-' + settings.sTableId;
    maxInput.name = 'max-' + settings.sTableId;
    maxInput.type = 'text';
    toolbar.appendChild(maxInput);

    const calcTime = luxon.DateTime.now();

    const minTime = calcTime.minus({ minutes: 30 }).toJSDate();

    const dateSettings = {
        enableTime: true,
        dateFormat: "Y-m-d H:i",
    }

    minDate[settings.sTableId] = new flatpickr(minInput, dateSettings);
    maxDate[settings.sTableId] = new flatpickr(maxInput, dateSettings);

    minDate[settings.sTableId].setDate(minTime);

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
                            minDate[dt.settings()[0].sTableId].clear();
                            maxDate[dt.settings()[0].sTableId].clear();
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
