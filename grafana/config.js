define(['settings'],
function (Settings) {
  return new Settings({
    datasources: {
	'metrics': {
            type: 'influxdb',
            url: "<--URL-->/<--DB_NAME-->",
            username: "<--INFLUXDB_USER-->",
            password: "<--INFLUXDB_PASS-->"
	},
	'grafana': {
            type: 'influxdb',
            url: "<--URL-->/<--GRAFANA_DB_NAME-->",
            username: "<--INFLUXDB_USER-->",
            password: "<--INFLUXDB_PASS-->",
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
