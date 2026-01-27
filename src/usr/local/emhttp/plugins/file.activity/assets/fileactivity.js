const minDate = {};
const maxDate = {};
const autoRefreshIntervals = {};

DataTable.ext.search.push((settings, data, dataIndex) => {
  if (
    minDate[settings.sTableId] === undefined ||
    maxDate[settings.sTableId] === undefined
  ) {
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

  const min = minValEmpty ? luxon.DateTime.fromMillis(0).toJSDate() : minVal[0];
  const max = maxValEmpty
    ? luxon.DateTime.now().plus({ hours: 1 }).toJSDate()
    : maxVal[0];

  if (min <= dateVal && dateVal <= max) {
    return true;
  }
  return false;
});

DataTable.feature.register("dateRange", (settings, opts) => {
  const toolbar = document.createElement("div");
  toolbar.appendChild(document.createTextNode("From: "));

  const minInput = document.createElement("input");
  minInput.id = `min-${settings.sTableId}`;
  minInput.name = `min-${settings.sTableId}`;
  minInput.type = "text";
  toolbar.appendChild(minInput);

  toolbar.appendChild(document.createTextNode(" To: "));
  const maxInput = document.createElement("input");
  maxInput.id = `max-${settings.sTableId}`;
  maxInput.name = `max-${settings.sTableId}`;
  maxInput.type = "text";
  toolbar.appendChild(maxInput);

  const dateSettings = {
    enableTime: true,
    dateFormat: "Y-m-d H:i",
  };

  minDate[settings.sTableId] = new flatpickr(minInput, dateSettings);
  maxDate[settings.sTableId] = new flatpickr(maxInput, dateSettings);

  minInput.addEventListener("change", () => settings.api.draw());
  maxInput.addEventListener("change", () => settings.api.draw());

  return toolbar;
});

DataTable.feature.register("autoRefresh", (settings, opts) => {
  const toolbar = document.createElement("div");
   
  const select = document.createElement("select");
  select.id = `autorefresh-${settings.sTableId}`;
  select.name = `autorefresh-${settings.sTableId}`;
  
  const options = [
    { value: "0", label: "Auto Refresh Disabled" },
    { value: "2000", label: "2 seconds" },
    { value: "5000", label: "5 seconds" },
    { value: "10000", label: "10 seconds" },
    { value: "30000", label: "30 seconds" },
    { value: "60000", label: "1 minute" },
    { value: "120000", label: "2 minutes" }
  ];
  
  for (const opt of options) {
    const option = document.createElement("option");
    option.value = opt.value;
    option.textContent = opt.label;
    select.appendChild(option);
  }
  
  toolbar.appendChild(select);
  
  // Initialize interval tracking
  autoRefreshIntervals[settings.sTableId] = null;
  
  select.addEventListener("change", (e) => {
    const interval = Number.parseInt(e.target.value);
    
    // Clear existing interval if any
    if (autoRefreshIntervals[settings.sTableId]) {
      clearInterval(autoRefreshIntervals[settings.sTableId]);
      autoRefreshIntervals[settings.sTableId] = null;
    }
    
    // Set new interval if not disabled
    if (interval > 0) {
      autoRefreshIntervals[settings.sTableId] = setInterval(() => {
        settings.api.ajax.reload(null, false); // false = stay on current page
      }, interval);
    }
  });
  
  return toolbar;
});

function getDatatableConfig(url) {
  return {
    ajax: {
      url: url,
      dataSrc: "",
    },
    columns: [
      { name: "Timestamp", data: "timestamp" },
      { name: "action", data: "action" },
      { name: "path", data: "filePath" },
      { name: "pid", data: "pid" },
      { name: "processPath", data: "processPath" },
      { name: "containerName", data: "containerName" },
      { name: "disk", data: "disk", visible: true, orderable: false },
    ],
    order: [[0, 'desc']],
    columnControl: {
      target: 0,
      content: [
        {
          extend: "dropdown",
          content: ["searchClear", "search"],
          icon: "search",
        },
      ],
    },
    columnDefs: [
      {
        targets: '_all',
        render: DataTable.render.text(),
        className: "dt-head-left",
      },
      {
        targets: 0,
        render: DataTable.render.datetime(),
        columnControl: {
          target: 0,
          content: [],
        },
      },
      {
        targets: [1, 5, 6],
        columnControl: {
          target: 0,
          content: [
            {
              extend: "dropdown",
              content: ["searchClear", "searchList"],
              icon: "search",
            },
          ],
        },
      },
    ],
    paging: true,
    pageLength: 50,
    ordering: true,
    layout: {
      topStart: {
        buttons: [
          {
            text: translator.tr("refresh"),
            action: (e, dt, node, config) => {
              dt.ajax.reload();
            },
          },
          {
            text: translator.tr("clear_filters"),
            action: (e, dt, node, config) => {
              minDate[dt.settings()[0].sTableId].clear();
              maxDate[dt.settings()[0].sTableId].clear();
              dt.search("");
              dt.columns().ccSearchClear();
              dt.draw();
            },
          },
        ],
        pageLength: {
          menu: [25, 50, 100, 200, -1],
        },
      },
      topEnd: {
        autoRefresh: {},
        dateRange: {},
      },
    },
  };
}
