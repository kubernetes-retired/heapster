define(['settings'],
function (Settings) {
  return new Settings({
    datasources: {
	'metrics': {
            type: 'influxdb',
            url: "<--PROTO-->://<--ADDR-->:<--PORT-->/db/<--DB_NAME-->",
            username: "<--USER-->",
            password: "<--PASS-->"
	},
	'grafana': {
            type: 'influxdb',
            url: "<--PROTO-->://<--ADDR-->:<--PORT-->/db/<--GRAFANA_DB_NAME-->",
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
