# prometheus-alertmanager-http-server
Very simple HTTP server which run BASH script for [Prometheus](https://prometheus.io) alerts from [Prometheus Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/).

Server process JSON payload and store value of **status** key which could be **firing** or **resolved** for every alert name defined in rule file. Last status is cached (and first hit is ignored) so BASH script is launched only when status change and not every `repeat_interval` as defined in alertmanager.yml. Rules in rule file should be reporting firing all the time as normal condition for this user case.

# Prerequisites
* [Go](https://go.dev/doc/)

# Install
Just copy, compile with `go build` and edit BASH script to your needs.
```
mkdir /opt/prometheus-alertmanager-http-server
cp -va prometheus-alertmanager-http-server.go /opt/prometheus-alertmanager-http-server/
cp -va prometheus-alertmanager-http-server-notify.sh /opt/prometheus-alertmanager-http-server/
cp -va prometheus-alertmanager-http-server.service /etc/systemd/system/
cd /opt/prometheus-alertmanager-http-server
go build prometheus-alertmanager-http-server.go
chmod 755 prometheus-alertmanager-http-server
systemctl daemon-reload
systemctl enable prometheus-alertmanager-http-server.service
systemctl start prometheus-alertmanager-http-server.service
```

# Configuration
Add rule file to `rule_files:` section in prometheus.yml.

Sample:
```
groups:
- name: Plugs
  rules:
  - alert: Plug Washing Machine
    ### avg_over_time help me to ignore short wifi/telemetry data outages
    expr: avg_over_time(tasmota_status_power{status_topic="plug_washing-machine"}[50s]) == 0
    labels:
      severity: critical
    annotations:
      summary: "{{ $labels.status_net_ip_address }}"
      description: "[{{ $labels.instance }}] of job {{ $labels.job }} is down."
```

Add `receiver: url` to `route:` section in alertmanager.yml and url block to `receivers:` section.
```
- name: 'url'
  webhook_configs:
  - url: http://localhost:6666/
```
Restart services prometheus and prometheus-alertmanager.

# Logging
```
journalctl -f -u prometheus-alertmanager-http-server.service
```
Also look to /dev/shm/. There could be files just for DEBUG purposes.
