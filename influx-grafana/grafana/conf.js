define(['settings'],
function (Settings) {
  return new Settings({
    //elasticsearch: "http://"+window.location.hostname+":9200",
    datasources: {
      influx: {
        default: true,
        type: 'influxdb',
        url: "://:/db/",
        username: "",
        password: ""
      },
      elasticsearch: {
        type: 'elasticsearch',
        url: "://:9200",
        index: 'grafana-dash',
        grafanaDB: true
      }
    },
    search: {
      max_results: 100
    },
    window_title_prefix: 'Kubernetes - ',
    default_route: '/dashboard/file/default.json',
    timezoneOffset: null,
    unsaved_changes_warning: true,
  });
});
