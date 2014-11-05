define(['settings'],
function (Settings) {
  return new Settings({
    datasources: {
      influx: {
        default: true,
        type: 'influxdb',
        url: "<--PROTO-->://<--ADDR-->:<--PORT-->/db/<--DB_NAME-->",
        username: "<--USER-->",
        password: "<--PASS-->"
      },
      elasticsearch: {
        type: 'elasticsearch',
        url: "<--PROTO-->://<--ADDR-->:9200",
        index: 'grafana-dash',
        grafanaDB: true
      }
    },
    search: {
      max_results: 100
    },
    window_title_prefix: 'Kubernetes - ',
    default_route: "<--DASHBOARD-->",
    timezoneOffset: null,
    unsaved_changes_warning: true,
  });
});
