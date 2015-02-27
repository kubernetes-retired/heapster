define(['settings'],
function (Settings) {
  return new Settings({
    datasources: {
	'metrics': {
            type: 'influxdb',
            url: window.location.origin+"/db/<--DB_NAME-->",
            username: "<--USER-->",
            password: "<--PASS-->"
	},
	'grafana': {
            type: 'influxdb',
            url: window.location.origin+"/db/<--GRAFANA_DB_NAME-->",
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
