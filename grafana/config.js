define(['settings'],
function (Settings) {
  return new Settings({
    datasources: {
	'metrics': {
            type: 'influxdb',
            url: "<--URL-->/<--DB_NAME-->",
            username: "<--USER-->",
            password: "<--PASS-->"
	},
	'grafana': {
            type: 'influxdb',
            url: "<--URL-->/<--GRAFANA_DB_NAME-->",
            username: "<--USER-->",
            password: "<--PASS-->",
	    grafanaDB: true
	}
    },
    search: {
      max_results: 100
    },
    window_title_prefix: 'Heapster - ',
    default_route: "<--DASHBOARD-->",
    timezoneOffset: null,
    unsaved_changes_warning: true,
  });
});
