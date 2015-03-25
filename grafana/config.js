define(['settings'],
function (Settings) {
  return new Settings({
    datasources: {
	'metrics': {
            type: 'influxdb',
            url: '@INFLUXDB_METRICS_URL@',
            username: '@INFLUXDB_USER@',
            password: '@INFLUXDB_PASS@'
	},
	'grafana': {
            type: 'influxdb',
            url: '@INFLUXDB_GRAFANA_URL@',
            username: '@INFLUXDB_USER@',
            password: '@INFLUXDB_PASS@',
	    grafanaDB: true
	}
    },
    search: {
      max_results: 100
    },
    window_title_prefix: 'Heapster - ',
    default_route: '@DASHBOARD@',
    timezoneOffset: null,
    unsaved_changes_warning: true,
  });
});
